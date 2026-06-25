# djicloud MQTT Hooks 规范

> `app/djicloud/internal/hooks/` 负责构造 djisdk handler 闭包，向 `djisdk.Client` 注入 MQTT 上行分发逻辑：解析消息、写入数据库、推送 WebSocket。

## 文件组织

| 文件 | 处理的 Topic | 职责 |
|------|-------------|------|
| `register.go` | — | 通过 `WithDjiClientOptions` 返回 `[]djisdk.ClientOption`，依赖通过 `RegisterDjiClientOptions` 注入 |
| `sys_status_up.go` | `sys/.../status` | 处理 `update_topo` → 更新 `dji_device` + `dji_device_topo` |
| `telemetry_up.go` | `thing/.../osd` + `thing/.../state` | 设备 OSD 遥测 + State 快照 → 在线上报 + 快照 Upsert + WebSocket 推送 |
| `event_notify_up.go` | `thing/.../events` | 航线任务进度/就绪、返航信息、HMS 告警、远程日志进度、OTA、PSDK 自定义数据 |
| `mqtt_request_up.go` | `thing/.../requests` | 设备主动请求（组织绑定、飞行区查询），返回安全默认值 |
| `mqtt_drc_up.go` | `thing/.../drc/up` | DRC 上行（非高频 → 落库，心跳 → WebSocket 推送） |
| `store_helper.go` | — | 工具函数：时间转换、JSON 序列化、版本提取、任务状态文本 |
| `online_cache.go` | — | 在线缓存辅助函数 `IsOnline`、`OnlineValue` |

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
| `eventHandlerOptions` | FlightTaskProgress, FlightTaskReady, ReturnHomeInfo, CustomData(Psdk/Esdk), HmsEventNotify, RemoteLogFileUploadProgress, OtaProgress |
| `telemetryHandlerOptions` | OsdHandler, StateHandler, StatusHandler |
| `drcHandlerOptions` | DrcUpHandler |
| `requestHandlerOptions` | RequestHandler |
| `onlineCheckerOption` | OnlineChecker |

**规则**：DB / OnlineCache / PushCli 都由 svc 层创建后注入，hooks 包内不持有全局变量。

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
- 使用 `djisdk.DrcUpPayloadSummary` 记录简短摘要
