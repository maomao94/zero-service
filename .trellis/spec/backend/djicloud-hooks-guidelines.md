# djicloud MQTT Hooks 规范

> `app/djicloud/internal/hooks/` 负责构造 djisdk handler 闭包，向 `djisdk.Client` 注入 MQTT 上行分发逻辑：解析消息、写入数据库、推送 WebSocket。

## 文件组织

| 文件 | 处理的 Topic | 职责 |
|------|-------------|------|
| `register.go` | — | 通过 `WithDjiClientOptions` 返回 `[]djisdk.ClientOption`，依赖通过 `RegisterDjiClientOptions` 注入 |
| `sys_status_up.go` | `sys/.../status` | 处理 `update_topo` → 更新 `dji_device` + `dji_device_topo` |
| `telemetry_up.go` | `thing/.../osd` + `thing/.../state` | 设备 OSD 遥测 + State 快照 → 在线上报 + 快照 Upsert + WebSocket 推送 |
| `event_notify_up.go` | `thing/.../events` | 航线任务进度/就绪、返航信息、HMS 告警、远程日志进度、OTA、PSDK/ESDK 自定义数据、飞行区同步进度/告警推送 |
| `mqtt_request_up.go` | `thing/.../requests` | 设备主动请求：`flight_areas_get` 返回飞行区文件列表；旧 platform method（`airport_*`）返回 `ErrSkipRequestReply` 跳过回复 |
| `mqtt_drc_up.go` | `thing/.../drc/up` | DRC 上行（非高频 → 落库，心跳 → WebSocket 推送） |
| `store_helper.go` | — | 工具函数：时间转换、JSON 序列化、版本提取、任务状态文本 |
| `online_cache.go` | — | 在线缓存辅助函数 `IsOnline`、`OnlineValue`，同时用于设备和网关的在线态判断 |

## 注册模式（Option 模式）

Handler 不再通过 `RegisterDjiClient(c, opts)` 后置注册，改为通过 `WithDjiClientOptions` 返回 option 列表，在 Client 构造时注入：

```go
type RegisterDjiClientOptions struct {
    DB                 *gormx.DB
    OnlineCache        *collection.Cache
    PushCli            socketpush.SocketPushClient
    DisableOsdSQLTrace bool
}

// 在 svc/servicecontext.go 中
djiOpts = append(djiOpts, hooks.WithDjiClientOptions(hooks.RegisterDjiClientOptions{
    DB:          db,
    OnlineCache: onlineCache,
    PushCli:     pushCli,
    DisableOsdSQLTrace: c.Telemetry.DisableOsdSQLTrace,
})...)
djiCli := djisdk.MustNewClient(c.MqttConfig, djiOpts...)
```

### Option 分组

`WithDjiClientOptions` 输出按功能域组织：

| 分组函数 | 包含的 Handler |
|----------|---------------|
| `eventHandlerOptions` | FlightTaskProgress, FlightTaskReady, ReturnHomeInfo, CustomData(Psdk/Esdk), HmsEventNotify, RemoteLogFileUploadProgress, OtaProgress, FlightAreasSyncProgress, FlightAreasDroneLocation |
| `telemetryHandlerOptions` | OsdHandler, StateHandler, StatusHandler |
| `drcHandlerOptions` | DrcUpHandler |
| `requestHandlerOptions` | RequestHandler |
| `onlineCheckerOption` | OnlineChecker |

**规则**：DB / OnlineCache / PushCli 都由 svc 层创建后注入，hooks 包内不持有全局变量。

## DRC Mode Enter 流程

`drcmodeenterlogic.go` 通过 `drchelper.go:toDrcMqttBroker` 构建 DRC Broker 连接信息下发机巢：

```
Config.Dji.Drc.Address (公网地址，优先)
  → 回退到 Config.Dji.MqttConfig.Broker[0] (内网地址，机巢可能不可达)
```

- DRC 地址须为机巢公网可达的 IP 或域名，否则机巢返回 `514304 (连接失败)`
- `toDrcMqttBroker(cfg, drcAddress)` — `drcAddress` 非空时优先使用，空时回退到 `cfg.Broker[0]`
- ClientID 每次重新生成（`dji-cloud-drc-{uuid}`），不复用主 MQTT ClientID

## DRC Up 处理（mqtt_drc_up.go）

DRC 心跳通知已由 `djisdk.Client.HandleDrcUp` 内部处理（调用 `drcManager.OnDeviceHeartbeat`）。hooks 层的 `NewDrcUpHandler` 只需负责：
- 非高频消息落库（`DjiDrcUpEvent`）
- 心跳消息的 WebSocket 推送（`pushCli.BroadcastRoom`）

不需要再手动调用 `drcManager.OnDeviceHeartbeat`。

```go
// 仅负责 DB + WebSocket 推送，无 drcManager 参数
func NewDrcUpHandler(db *gormx.DB, pushCli socketpush.SocketPushClient) djisdk.DrcUpHandler
```

- 高频消息（`heart_beat`、`osd_info_push`、`hsi_info_push`、`delay_info_push`、`drc_initial_state_subscribe`）：不落库
- 非高频消息：根据 `DrcUnmarshalUpData` 解析结果写入 `DjiDrcUpEvent`（Insert-only）

## FlightAreasGet 处理（mqtt_request_up.go）

设备收到 `flight_areas_update` 通知后主动调用 `flight_areas_get` 拉取飞行区文件列表。

**查询策略**：直接查 `DjiFlyRegion` 最新记录（GORM 自动过滤软删除），不再通过 `DjiFlyRegionSyncStatus` 中转。

**签名策略**：不自存预签名 URL（会过期），在 `buildFlightAreasReply` 中通过 `ossTemplate.SignUrl` 实时生成。多文件时使用 `mr.Finish` 并发签名。

```go
var regions []gormmodel.DjiFlyRegion
db.Where("gateway_sn = ?", gatewaySn).Order("id DESC").Find(&regions)

files := make([]djisdk.FlightAreasFile, len(regions))
fns := make([]func() error, 0, len(regions))
for i := range regions {
    i, r := i, regions[i]
    fns = append(fns, func() error {
        u, _ := ossTemplate.SignUrl(ctx, "", bucket, r.FileName, 7*24*time.Hour)
        files[i] = djisdk.FlightAreasFile{Name: r.FileName, URL: u, Size: r.FileSize, Checksum: r.Checksum}
        return nil
    })
}
mr.Finish(fns...)
```

**依赖注入**：`NewDeviceRequestHandler(db, ossTemplate, ossBucket)` — DB 用于查询，OSS 用于实时签名。通过 `RegisterDjiClientOptions` 传入。

## FlightAreasSyncProgress 处理（event_notify_up.go）

设备下载 OSS 文件推送给飞机后上报同步进度。handler 按 `gateway_sn + file_name` 匹配 `DjiFlyRegion`，然后 INSERT 一条 `DjiFlyRegionSyncStatus` 记录（Insert-only，历史可追溯）。

```go
var region gormmodel.DjiFlyRegion
db.Where("gateway_sn = ? AND file_name = ?", gatewaySn, data.File.Name).Order("id DESC").First(&region)
db.Create(&gormmodel.DjiFlyRegionSyncStatus{
    GatewaySn:   gatewaySn,
    FlyRegionID: region.Id,
    SyncStatus:  data.Status,
    SyncReason:  data.Reason,
})
```
- 使用 `djisdk.DrcUpPayloadSummary` 记录简短摘要
