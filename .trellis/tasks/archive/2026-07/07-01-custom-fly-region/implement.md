# 实现计划：自定义飞行区

## 执行顺序

### Step 1: `common/djisdk/geofence.go` — GeoJSON ↔ orb 工具

- [ ] 创建文件，定义 `GeofenceFeatureCollection`、`GeofenceFeature`、`GeofenceGeometry`、`GeofenceProperties`、`NFZPoint` 结构体
- [ ] 实现 `OrbPolygonToGeofenceFeature(orb.Polygon) GeofenceFeature`
- [ ] 实现 `OrbPointToNFZFeature(orb.Point, radius) GeofenceFeature`
- [ ] 实现 `NewGeofenceFeatureCollection(...GeofenceFeature) GeofenceFeatureCollection`
- [ ] 实现 `(GeofenceFeatureCollection) ToJSON() ([]byte, error)`
- [ ] 实现 `ParseGeofenceCollection([]byte) (*GeofenceFeatureCollection, error)`
- [ ] 实现 `(GeofenceFeatureCollection) ExtractPolygons() (map[string]orb.Polygon, error)`
- [ ] 实现 `(GeofenceFeatureCollection) ExtractNFZPoints() (map[string]NFZPoint, error)`

**验证**: `go build ./common/djisdk/`

### Step 2: `app/djicloud/internal/config/config.go` — OSS 可选配置

- [ ] 新增 `OssConfig` 结构体（Category/Endpoint/AccessKey/SecretKey/BucketName/Region）
- [ ] 在 `Config` 中增加 `Oss *OssConfig` 可选字段

**验证**: `go build ./app/djicloud/internal/config/`

### Step 3: `app/djicloud/djicloud.proto` — 新增 RPC

- [ ] 新增 message: `PointCoordinate`、`FlyRegionDFence`、`FlyRegionNFZ`
- [ ] 新增 message: `SetCustomFlyRegionReq`、`SetCustomFlyRegionRes`
- [ ] 新增 RPC: `rpc SetCustomFlyRegion(SetCustomFlyRegionReq) returns (SetCustomFlyRegionRes);`
- [ ] 重新生成 proto 代码: `goctl rpc protoc djicloud.proto --go_out=. --go-grpc_out=. --zrpc_out=.`

**验证**: `go build ./app/djicloud/`

### Step 4: `app/djicloud/model/gormmodel/dji_fly_region.go` — 新增模型

- [ ] 创建文件，定义 `DjiFlyRegion` 结构体（GatewaySn, FileName, FileURL, FileSize, Checksum, GeofenceJSON）
- [ ] 定义 `DjiFlyRegionSyncStatus` 结构体（GatewaySn, FlyRegionID, SyncStatus, SyncReason）
- [ ] 在 `servicecontext.go` 的 AutoMigrate 列表中注册新模型

**验证**: `go build ./app/djicloud/model/gormmodel/`

### Step 5: `app/djicloud/internal/svc/servicecontext.go` — OSS 初始化

- [ ] 当 `c.Oss != nil` 时初始化 `MinioTemplate`
- [ ] 将 `ossTemplate` 注入 `ServiceContext`
- [ ] 将 DB 引用注入 `RequestHandler`（闭包方式）

**验证**: `go build ./app/djicloud/internal/svc/`

### Step 6: `app/djicloud/internal/hooks/` — RequestHandler 注入 DB

- [ ] 修改 `NewDeviceRequestHandler` 接受 `*gormx.DB` 参数
- [ ] `FlightAreasGet` handler 查询 DB 获取最新 `DjiFlyRegion` 文件列表
- [ ] 组装 `FlightAreasFile` 返回
- [ ] 更新 `register.go` 中 handler 注册调用

**验证**: `go build ./app/djicloud/internal/hooks/`

### Step 7: `app/djicloud/internal/logic/setcustomflyregionlogic.go` — 核心逻辑

- [ ] 实现 `SetCustomFlyRegion` 方法（7 步流程）
- [ ] 参数校验
- [ ] GeoJSON 生成（调用 djisdk/geofence.go）
- [ ] OSS 上传 + 签名 URL
- [ ] DB 写入 DjiFlyRegion + DjiFlyRegionDevice
- [ ] MQTT 通知设备 FlightAreasUpdate

**验证**: `go build ./app/djicloud/internal/logic/`

### Step 8: 全局编译验证

- [ ] `go build ./...`
- [ ] lint 检查（如有）

## 回滚点

- Step 1-3 独立可测，无依赖关系
- Step 4-5 依赖 Step 3（proto 生成）
- Step 6 依赖 Step 4（模型）
- Step 7 依赖 Step 1-6 全部

## 风险文件

- `mqtt_request_up.go`: 修改 handler 签名会联动 `register.go`
- `servicecontext.go`: 新增 OSS 初始化可能影响启动流程
- `djicloud.proto`: proto 重新生成可能影响已有消息序号
