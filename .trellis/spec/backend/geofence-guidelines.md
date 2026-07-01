# DJI 飞行区（Geofence）编码规范

> DJI 自定义飞行区 GeoJSON 生成、OSS 存储、设备同步全流程规范。

## 概述

飞行区功能允许平台定义禁飞区（nfz）和围栏（dfence），通过 GeoJSON 文件下发给 DJI 机巢设备。全流程涉及：RPC 入参 → GeoJSON 构造 → OSS 上传 → DB 落库 → MQTT 通知设备 → 设备拉取文件 → 设备上报同步状态。

## 文件组织

| 文件 | 职责 |
| --- | --- |
| `common/djisdk/geofence.go` | GeoJSON 类型定义与构造函数（接口层，无外部依赖） |
| `common/djisdk/geofence_test.go` | GeoJSON 序列化/反序列化测试 |
| `common/djisdk/protocol.go` | MQTT 协议类型：`FlightAreasFile`、`FlightAreasGetReplyData`、`FlightAreasSyncProgressEvent` |
| `app/djicloud/djicloud.proto` | RPC 契约：`FlyRegionFeature`、`SubmitCustomFlyRegionReq/Res`、`DeleteCustomFlyRegionReq/Res`、`DeleteCustomFlyRegionByFileIdReq/Res`、`ListFlyRegionsReq/Res`、`ListFlyRegionSyncStatusReq/Res` |
| `app/djicloud/internal/logic/submitcustomflyregionlogic.go` | 提交飞行区：proto → GeoJSON → OSS → DB → MQTT |
| `app/djicloud/internal/logic/deletecustomflyregionlogic.go` | 清除设备下所有飞行区 |
| `app/djicloud/internal/logic/deletecustomflyregionbyfileidlogic.go` | 按 file_id 删除单个飞行区 |
| `app/djicloud/internal/logic/listflyregionslogic.go` | 分页查询飞行区配置（支持 sign_url 返回 OSS 签名地址） |
| `app/djicloud/model/gormmodel/dji_fly_region.go` | DB 模型：`DjiFlyRegion`、`DjiFlyRegionSyncStatus` |
| `app/djicloud/internal/hooks/mqtt_request_up.go` | 设备 `flight_areas_get` 拉取处理（签名 URL 生成） |
| `app/djicloud/internal/hooks/event_notify_up.go` | `flight_areas_sync_progress` 同步状态回调落库 |

## GeoJSON 构造规范

### 核心类型

参考文件：`common/djisdk/geofence.go:12-31`

```go
type GeofenceFeatureCollection struct {
    Type     string            `json:"type"`     // "FeatureCollection"
    Features []GeofenceFeature `json:"features"`
}

type GeofenceFeature struct {
    ID           string             `json:"id"`            // 区域唯一 ID（UUID）
    Type         string             `json:"type"`          // "Feature"
    GeofenceType string             `json:"geofence_type"` // "dfence" 或 "nfz"
    Geometry     json.RawMessage    `json:"geometry"`      // Polygon 或 Point
    Properties   GeofenceProperties `json:"properties"`
}

type GeofenceProperties struct {
    Radius  float64 `json:"radius"`           // 圆形时 > 0，多边形时为 0
    SubType string  `json:"subType,omitempty"` // "Circle" 仅圆形使用
    Enable  bool    `json:"enable"`            // 区域使能
}
```

### 构造函数

- `NewGeofencePolygonFeature(id, geofenceType, coordinates, enabled)` — `geofence.go:42`
  - `coordinates` 为 `[][2]float64`，会自动包装为 `[][][2]float64`（Polygon rings）
  - Polygon 首尾点**不需要**调用者手动闭合
- `NewGeofenceCircleFeature(id, geofenceType, lng, lat, radius, enabled)` — `geofence.go:61`
  - geometry.type 固定为 "Point"
  - properties.subType 固定为 "Circle"，radius 直接放入 properties
- `NewGeofenceFeatureCollection(features...)` — `geofence.go:34`

### DJI 与标准 GeoJSON 的差异（重要）

DJI 飞行区 GeoJSON **不是** RFC 7946 标准 GeoJSON。差异点：

| 字段 | 标准 GeoJSON | DJI 格式 |
| --- | --- | --- |
| `Feature.geofence_type` | 不存在 | **必需**，值为 "dfence" 或 "nfz" |
| `Feature.properties.radius` | 不存在 | 多边形为 0，圆形为实际半径 |
| `Feature.properties.subType` | 不存在 | "Circle" 仅用于圆形 |
| 坐标顺序 | [经度, 纬度] | [经度, 纬度]（WGS84，与标准一致） |

**不可用标准 GeoJSON 解析器（如 `orb/geojson`）直接反序列化**——`geofence_type` 在 Feature 顶层而非 properties 内，标准解析器会静默丢弃此字段。当前使用 hand-roll struct + `encoding/json` 直接序列化，确保 DJI 设备端正确解析。

### UUID 约定

- 每个 feature 的 `id`：推荐使用 UUID（proto 注释提示），为空时自动生成。**每个 feature 独立一个 UUID**，不共用。
- 文件级 `fileId`：提交时生成一次，仅用于文件名和 DB 记录的 `file_id` 字段。**不是** feature 的 `id`。
- 文件名格式：`{geofence_type}_{fileId}.json`（如 `dfence_550e8400-xxxx.json`、`nfz_550e8400-xxxx.json`）
- `fileId` 在 DB 中为 uniqueIndex，保证唯一性

## 提交流程

参考文件：`app/djicloud/internal/logic/submitcustomflyregionlogic.go:36-104`

### 处理步骤

1. **校验**：`features` 非空
2. **生成 fileId**：`uuid.NewString()`，用于文件名和 DB 记录
3. **确定 geofenceType**：取首个 feature 的 `geofence_type`，空则默认 "geofence"
4. **遍历 features**：每个 feature 生成独立 UUID 作为 `id`（请求传入优先），按 geometry oneof 选择多边形或圆形构造函数
5. **构造 GeoJSON**：`NewGeofenceFeatureCollection(features...).ToJSON()`
6. **上传 OSS**：文件名 `{geofenceType}_{fileId}.json`，Content-Type `application/json`
7. **DB 落库**：写入 `DjiFlyRegion` 记录（FileId、FileName、GatewaySn、Checksum 等）
8. **MQTT 通知**：调用 `FlightAreasUpdate(gatewaySn)` 触发设备拉取

### 响应类型

`SubmitCustomFlyRegionRes` 在成功时返回 `file_id`，错误时复用 `code/message/tid/reason_code` 模式：

```go
// 成功
&djicloud.SubmitCustomFlyRegionRes{Code: 0, Message: "success", Tid: tid, FileId: fileId}
// 失败（DJI 错误）
&djicloud.SubmitCustomFlyRegionRes{Code: -1, Message: err.Error(), Tid: tid, FileId: fileId}
```

### DB 模型

参考文件：`app/djicloud/model/gormmodel/dji_fly_region.go:11-20`

```go
type DjiFlyRegion struct {
    gormx.LegacyBaseModel
    GatewaySn    string // 目标机巢序列号
    Name         string // 用户自定义飞行区名称（来自 gRPC 请求）
    FileId       string // 文件唯一标识(UUID)，uniqueIndex
    BucketName   string // OSS 存储桶（与 FileName 组成联合唯一索引 idx_bucket_file）
    FileName     string // OSS 文件 key（如 dfence_abc123.json），联合唯一索引 idx_bucket_file
    FileSize     int64
    Checksum     string // SHA256
    GeofenceJSON string // 原始 GeoJSON 内容
}
```

**唯一键**：
- `file_id` 为单独 uniqueIndex
- `bucket_name + file_name` 为联合唯一索引 `idx_bucket_file`

**写入策略**：Insert-only，每次提交新建记录，历史可追溯。查最新按 `create_time DESC`。

## 删除流程

### 清空设备所有飞行区

参考文件：`app/djicloud/internal/logic/deletecustomflyregionlogic.go`

- RPC: `DeleteCustomFlyRegion(device_sn)` → `DeleteCustomFlyRegionRes`（含 `deleted_file_ids`）
- 查询该 `gateway_sn` 下所有记录，收集 `file_id` 列表
- 批量删除所有匹配记录
- 调用 `FlightAreasUpdate(gatewaySn)` 通知设备
- 响应复用 `code/message/tid/reason_code` 模式，额外返回 `deleted_file_ids`

### 按 file_id 删除单个飞行区

参考文件：`app/djicloud/internal/logic/deletecustomflyregionbyfileidlogic.go`

- RPC: `DeleteCustomFlyRegionByFileId(file_id)` → `DeleteCustomFlyRegionByFileIdRes`
- **不需要** `device_sn` 入参——通过 `file_id` 从 DB 查询出 `gateway_sn`
- 删除单条记录后，用查出的 `gateway_sn` 下发 MQTT 通知
- 返回被删除的 `file_id`

## 设备拉取流程

参考文件：`app/djicloud/internal/hooks/mqtt_request_up.go:41-82`

设备收到 `flight_areas_update` 信号后，主动调用 `flight_areas_get` 拉取文件列表：

1. 按 `gateway_sn` 查询所有未删除的 `DjiFlyRegion` 记录
2. 通过 `ossTemplate.SignUrl` 生成 7 天有效期的签名下载 URL
3. 返回 `FlightAreasGetReplyData{Files: []FlightAreasFile{...}}`

## 同步状态回调

参考文件：`app/djicloud/internal/hooks/event_notify_up.go`

设备下载并应用飞行区文件后上报 `flight_areas_sync_progress` 事件，平台按 `gateway_sn + file_name` 关联 `DjiFlyRegion` 记录，写入 `DjiFlyRegionSyncStatus`。

## 列表查询

参考文件：`app/djicloud/internal/logic/listflyregionslogic.go`

### 基本查询

- RPC: `ListFlyRegions(page, page_size, gateway_sn)` → `ListFlyRegionsRes`
- 按 `gateway_sn` 过滤（可选），`id DESC` 排序，`gormx.QueryPage` 分页
- 返回字段：`id`、`gateway_sn`、`name`、`file_id`、`file_name`、`file_size`、`checksum`、`create_time`

### sign_url 签名地址

设置 `sign_url=true` 时，使用 `mr.Finish` 并发为每个文件生成 OSS 签名下载地址（7 天有效期），签入 `FlyRegionInfo.url` 字段。签名失败时仅记录日志，不阻断返回。

```go
if needSign && r.BucketName != "" {
    u, err := l.svcCtx.OssTemplate.SignUrl(l.ctx, "", r.BucketName, r.FileName, 7*24*time.Hour)
    if err != nil {
        logx.WithContext(l.ctx).Errorf("[dji-cloud] ListFlyRegions: sign url failed: %v", err)
    } else {
        info.Url = u
    }
}
```

## 反模式

- **不要在 GeoJSON 的 properties 中放 `geofence_type`**。DJI 要求在 Feature 顶层，放 properties 会导致设备解析失败。
- **不要用标准 GeoJSON 库直接序列化**。`orb/geojson` 等不支持 `geofence_type` 顶层字段。使用 `common/djisdk/geofence.go` 的 hand-roll 构造。
- **不要混用多边形和圆形的 properties**。多边形：`radius=0`，无 `subType`。圆形：`radius>0`，`subType="Circle"`。
- **不要在删除 flight area 后忘记调用 `FlightAreasUpdate`**。设备不会自动感知 DB 变更，必须通过 MQTT 通知触发拉取。
- **file_id 不要与 feature 的 id 混淆**。`file_id` 是文件级标识（模型字段，文件名），`feature.id` 是 GeoJSON 内每个区域的独立 UUID。
