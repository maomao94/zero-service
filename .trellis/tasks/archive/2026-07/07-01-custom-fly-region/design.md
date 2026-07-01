# 技术设计：自定义飞行区

## 架构概览

```
平台API (grpc SetCustomFlyRegion)
  │
  ├─ 1. 接收结构化参数 (Polygon坐标/nfz圆心+半径)
  ├─ 2. djisdk/geofence.go: 生成 DJI GeoJSON FeatureCollection
  ├─ 3. ossx: 上传 GeoJSON 文件至 Minio
  ├─ 4. gormx: 写入 fly_region 表 + fly_region_sync_status 表
  ├─ 5. djisdk.Client.FlightAreasUpdate(): 通知设备拉取
  │
  ▼
设备收到通知 → flight_areas_get → RequestHandler 查 DB 返回文件列表
```

## 文件变更清单

### 1. `common/djisdk/geofence.go` (新增)

提供 orb 对象与 DJI GeoJSON 之间的双向转换。

```go
// 数据结构（对应 DJI 协议格式）
type GeofenceFeatureCollection struct { ... }
type GeofenceFeature struct {
    ID           string            `json:"id"`
    Type         string            `json:"type"`         // "Feature"
    GeofenceType string            `json:"geofence_type"` // "dfence" | "nfz"
    Geometry     GeofenceGeometry  `json:"geometry"`
    Properties   GeofenceProperties `json:"properties"`
}
type GeofenceGeometry struct {
    Type        string    `json:"type"`         // "Polygon" | "Point"
    Coordinates any       `json:"coordinates"`
}
type GeofenceProperties struct {
    Radius  float64 `json:"radius"`           // dfence=0, nfz 10~
    SubType string  `json:"subType,omitempty"` // "Circle" for nfz
    Enable  bool    `json:"enable"`
}

// 转换函数
func OrbPolygonToGeofenceFeature(id string, polygon orb.Polygon, enabled bool) GeofenceFeature
func OrbPointToNFZFeature(id string, point orb.Point, radius float64, enabled bool) GeofenceFeature
func NewGeofenceFeatureCollection(features ...GeofenceFeature) GeofenceFeatureCollection
func ParseGeofenceCollection(raw []byte) (*GeofenceFeatureCollection, error)
func (fc *GeofenceFeatureCollection) ToJSON() ([]byte, error)
// 反向: GeoJSON → orb
func (fc *GeofenceFeatureCollection) ExtractPolygons() (map[string]orb.Polygon, error)
func (fc *GeofenceFeatureCollection) ExtractNFZPoints() (map[string]NFZPoint, error)
type NFZPoint struct { Point orb.Point; Radius float64 }
```

**设计决策**：
- 使用 `encoding/json` 的 raw message 处理 coordinates 的类型差异（Polygon 是 `[][][]float64`，Point 是 `[]float64`）
- 函数返回 orb 类型，与 `common/gisx/` 生态兼容
- 不做坐标转换（WGS84），直接透传

### 2. `app/djicloud/internal/config/config.go` (修改)

在 Config 中增加可选 OSS 配置：

```go
type Config struct {
    // ... 现有字段 ...
    Oss *OssConfig `json:",optional"`
}

type OssConfig struct {
    Category   int64  `json:",default=1"`    // 1=Minio
    Endpoint   string `json:",optional"`
    AccessKey  string `json:",optional"`
    SecretKey  string `json:",optional"`
    BucketName string `json:",optional"`     // 默认 geofence
    Region     string `json:",optional"`
}
```

ossx 包当前无需优化——现有接口和 Minio 实现已满足需求（`PutObject` 上传 GeoJSON 字节流，`SignUrl` 生成下载链接）。

### 3. `app/djicloud/djicloud.proto` (修改)

在"平台自有接口"分区新增：

```protobuf
// ==================== 自定义飞行区（平台能力） ====================

// SetCustomFlyRegion 设置自定义飞行区。
// 接收结构化参数（dfence 多边形 + nfz 禁飞区），生成 DJI GeoJSON 上传 OSS，
// 写入 DB 后触发指定设备飞行区更新。
rpc SetCustomFlyRegion(SetCustomFlyRegionReq) returns (SetCustomFlyRegionRes);

message SetCustomFlyRegionReq {
  // device_sn 目标机巢设备序列号（gateway_sn）。
  string device_sn = 1;
  // dfences 飞行区围栏列表（Polygon 几何）。
  repeated FlyRegionDFence dfences = 2;
  // nfzs 禁飞区列表（Point + Circle 类型）。
  repeated FlyRegionNFZ nfzs = 3;
}

message FlyRegionDFence {
  // id 围栏唯一标识，建议 UUID。
  string id = 1;
  // coordinates 多边形顶点坐标，[lng, lat] 顺序，首尾点相同。
  repeated PointCoordinate coordinates = 2;
  // enable 是否启用。
  bool enable = 3;
}

message FlyRegionNFZ {
  // id 禁飞区唯一标识，建议 UUID。
  string id = 1;
  // center 圆心坐标 [lng, lat]。
  PointCoordinate center = 2;
  // radius 半径，单位米，范围 [10, 无限)。
  double radius = 3;
  // enable 是否启用。
  bool enable = 4;
}

message PointCoordinate {
  double lng = 1; // WGS84 经度
  double lat = 2; // WGS84 纬度
}

message SetCustomFlyRegionRes {
  int32 code = 1;
  string message = 2;
}
```

### 4. `app/djicloud/model/gormmodel/dji_fly_region.go` (新增)

两张表，遵循现有模型规范（嵌入 `gormx.LegacyBaseModel`）：

```go
// DjiFlyRegion 飞行区配置主表
// 写策略: Insert-only（每次设置新建记录，历史可追溯）
// 查最新: ORDER BY create_time DESC LIMIT 1
type DjiFlyRegion struct {
    gormx.LegacyBaseModel
    GatewaySn  string `gorm:"column:gateway_sn;index;not null"`
    FileName   string `gorm:"column:file_name;not null"`      // OSS 文件名
    FileURL    string `gorm:"column:file_url;not null"`       // OSS 完整地址（含签名）
    FileSize   int64  `gorm:"column:file_size;not null;default:0"` // 字节数
    Checksum   string `gorm:"column:checksum;type:varchar(128);not null;default:''"` // sha256
    GeofenceJSON string `gorm:"column:geofence_json;type:text;default:''"` // 原始 GeoJSON（调试用）
}

// DjiFlyRegionSyncStatus 飞行区文件同步状态表
// 写策略: find-then-save-or-create（兼容 GaussDB/MySQL，不使用 Upsert）
// 记录每个机巢当前激活的飞行区文件及其同步状态
type DjiFlyRegionSyncStatus struct {
    gormx.LegacyBaseModel
    GatewaySn     string `gorm:"column:gateway_sn;uniqueIndex;not null"`
    FlyRegionID   int64  `gorm:"column:fly_region_id;index;not null;default:0"` // 关联 DjiFlyRegion.id
     SyncStatus    string `gorm:"column:sync_status;type:varchar(32);not null;default:''"` // notified | synchronized | failed
     SyncReason    int    `gorm:"column:sync_reason;not null;default:0"` // 失败原因码，0 表示无异常
}
```

### 5. `app/djicloud/internal/hooks/mqtt_request_up.go` (修改)

`FlightAreasGet` handler 改为从 DB 查询：

```go
case djisdk.MethodFlightAreasGet:
    // 查询该 gateway_sn 的 DjiFlyRegionSyncStatus → 取关联的 DjiFlyRegion
    // 组装 FlightAreasFile 列表（name, url, size, checksum）
    return &djisdk.FlightAreasGetReplyData{Files: files}, nil
```

**注意**：handler 函数签名是 `func(ctx, gatewaySn, *RequestMessage) (any, error)`，当前没有 DB 引用。需要将 handler 改为闭包注入 DB：

在 `hooks/register.go` 中通过 `WithRequestHandler(hooks.NewDeviceRequestHandler(db))` 传入 DB 依赖。

### 6. `app/djicloud/internal/logic/setcustomflyregionlogic.go` (新增)

核心流程：

```
SetCustomFlyRegion(in *SetCustomFlyRegionReq) -> SetCustomFlyRegionRes
  1. 参数校验: 至少一个 dfence 或 nfz
  2. 调用 djisdk geofence 工具: 
     - in.dfences[i] → OrbPolygonToGeofenceFeature
     - in.nfzs[i] → OrbPointToNFZFeature
     - NewGeofenceFeatureCollection → ToJSON
  3. 上传 OSS: ossTemplate.PutObject(ctx, tenantId, bucket, filename, "application/json", bytes.NewReader(jsonBytes))
  4. 生成签名 URL: ossTemplate.SignUrl(ctx, tenantId, bucket, filename, 7*24*time.Hour)
  5. 计算 checksum (sha256)
  6. 写入 DB（不使用 Upsert，兼容 GaussDB/MySQL）:
     - INSERT DjiFlyRegion (gateway_sn, file_name, file_url, file_size, checksum, geofence_json)
     - SELECT DjiFlyRegionSyncStatus WHERE gateway_sn → 有则 UPDATE，无则 INSERT
  7. 通知设备: l.svcCtx.DjiClient.FlightAreasUpdate(ctx, in.device_sn)
  8. 返回成功
```

### 7. `app/djicloud/internal/svc/servicecontext.go` (修改)

- 初始化 OSS MinioTemplate（当配置存在时）
- 将 `ossTemplate` 和 `flyRegionDeviceModel` 注入 ServiceContext

## 数据流

```
SetCustomFlyRegionReq
  → geofence.go: orb → GeoJSON []byte
  → ossx.PutObject(ctx, "tenant", "geofence", "geofence_xxx.json", ...) → File{Name, Link, Size, Md5}
  → ossx.SignUrl(...) → signedURL
  → sha256(GeoJSON bytes) → checksum
   → DB INSERT DjiFlyRegion + find-then-save-or-create DjiFlyRegionSyncStatus
  → mqtt: flight_areas_update (通知信号)
  ↓ (异步)
设备 flight_areas_get → RequestHandler → DB 查 DjiFlyRegionSyncStatus → 返回 FlightAreasFile[]
  → 设备下载 OSS 文件 → 同步进度 events (已由现有 event handler 框架处理)
```

## 注意事项

1. ossx 的 `Template()` 函数是按租户缓存的设计，但 djcloud 服务无需租户隔离。简化使用：直接在 `servicecontext.go` 中调用 `NewMinioTemplate` 创建单例，固定租户 ID。
2. OSS signed URL 有效期设为 7 天，与 DJI FlightTaskPrepare 的文件签名一致。
3. `FlightAreasGet` 返回给设备的是当前最新的飞行区文件（查 `DjiFlyRegionSyncStatus` 取关联的 `DjiFlyRegion`）。
4. Proto 中的 `PointCoordinate` 使用 `double lng/lat`，与 DJI 协议中的 WGS84 [lon, lat] 顺序一致。
5. geofence.go 不作坐标系转换，直接使用 WGS84。
