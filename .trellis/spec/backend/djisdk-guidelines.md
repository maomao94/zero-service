# djisdk 包规范（DJI MQTT SDK）

> `common/djisdk/` 封装大疆上云 MQTT 协议，提供云平台侧的 Client 类型：MQTT 连接管理、Topic 通配订阅、设备上行分发（事件/遥测/状态/请求/DRC）、下行命令发送（services/property/drc/down）和应答路由。

## 文件组织

| 文件 | 职责 |
|------|------|
| `doc.go` | 包级 godoc，协议方向说明 |
| `client.go` | `Client` 结构体、Option 模式、Handler 注册、命令发送、应答路由、在线检查 |
| `protocol.go` | 全部消息结构体：请求/应答/事件/遥测/属性 |
| `protocol_drc.go` | DRC 专属协议结构体、上行反序列化 dispatch、摘要函数 |
| `method.go` | DJI Cloud API method 字符串常量，按功能模块分组 |
| `topic.go` | MQTT Topic 构造函数和通配符 Pattern 函数 |
| `error.go` | `DJIError` 类型 + `IsDJIError` 断言 |
| `error_descriptions.go` | 错误码中文描述映射表 |

## Client 构造

### 工厂方法

两个构造入口（`client.go:101-114`）：

```go
// 自建 MQTT 连接
c := djisdk.MustNewClient(mqttConfig, opts...)

// 复用已有 mqttx.Client
c := djisdk.NewClient(existingMqttClient, opts...)
```

`MustNewClient` 在 MQTT 连接失败时 panic，调用方不能优雅降级。

### Option 列表

| Option | 类型 | 说明 |
|--------|------|------|
| `WithPendingTTL(ttl)` | `time.Duration` | services_reply 等待超时（默认 30s） |
| `WithReplyOptions(ro)` | `ReplyOptions` | 全局开关 events_reply/status_reply/requests_reply |

## Handler 注册即事件分发

### Handler 类型签名

| Handler | 签名 | 方向 | 对应 Topic |
|---------|------|------|-----------|
| `OnFlightTaskProgress` | `func(ctx, gatewaySn, *FlightTaskProgressEvent)` | up | events |
| `OnFlightTaskReady` | `func(ctx, gatewaySn, *FlightTaskReadyEvent)` | up | events |
| `OnReturnHomeInfo` | `func(ctx, gatewaySn, *ReturnHomeInfoEvent)` | up | events |
| `OnCustomDataFromPsdk` | `func(ctx, gatewaySn, *CustomDataFromPsdkEvent)` | up | events |
| `OnCustomDataFromEsdk` | `func(ctx, gatewaySn, *CustomDataFromEsdkEvent)` | up | events |
| `OnHmsEventNotify` | `func(ctx, gatewaySn, *HmsEventData)` | up | events |
| `OnRemoteLogFileUploadProgress` | `func(ctx, gatewaySn, *RemoteLogFileUploadProgressEvent)` | up | events |
| `OnOtaProgress` | `func(ctx, gatewaySn, *OtaProgressEvent)` | up | events |
| `OnTopoUpdate` | `func(ctx, gatewaySn, *TopoUpdateData)` | up | events |
| `OnOsd` | `func(ctx, deviceSn, *OsdMessage)` | up | osd |
| `OnState` | `func(ctx, deviceSn, *StateMessage)` | up | state |
| `OnStatus` | `StatusHandler` 见下 | up | status |
| `OnRequest` | `RequestHandler` 见下 | up | requests |
| `OnDrcUp` | `DrcUpHandler` 见下 | up | drc/up |
| `OnEvent(method, EventMethodFallback)` | 见下 | up | events(兜底) |

有返回值的 Handler：
- `StatusHandler` `func(ctx, gatewaySn, *StatusMessage) int` — 返回 result 码，ReplyOptions 控制是否发 status_reply
- `RequestHandler` `func(ctx, gatewaySn, *RequestMessage) (result int, output any, err error)` — 返回 result + output，ReplyOptions 控制是否发 requests_reply
- `DrcUpHandler` `func(ctx, gatewaySn, *DrcUpMessage, parsed any) error` — parsed 已由 DrcUnmarshalUpData 反序列化为具体类型或 `DrcUnknownUpData`
- `EventMethodFallback` `func(ctx, *EventMessage) (result int, err error)` — 未命中预置 On* 时调用

### 事件分发层级

`HandleEvents`（`client.go:195-224`）的三级分发：

1. **预置 On\* 分支**（`tryDispatchEventNotify`，`client.go:230-343`）：switch on method，命中已注册的 On* handler 则执行，返回 handled=true
2. **EventMethodFallback 兜底**（`eventMethodFallbacks[method]`）：预置未命中时调用注册的 fallback
3. **静默丢弃**：预置未命中且无 fallback 时不做任何操作

添加新事件类型需要：增加 method 常量（`method.go`）、增加 handler 字段 + setter（`client.go`）、增加 `case` 分支（`tryDispatchEventNotify`）。

### 注册时机

Handler 通过 Setter 赋值（不是通过 interface 或 map），注册是最后一次写入生效（last-wins），不支持并发注册。所有注册应在 `SubscribeAll` 之前完成。

## 命令下发

### 阻塞模式（SendCommand）

`client.go:484-513`：

```go
tid, err := c.SendCommand(ctx, gatewaySn, method, data)
// tid 可用于追踪；err 包含 DJI 设备返回的 error code
```

流程：生成 UUID (tid/bid) → 构建 ServiceRequest → Publish 到 services topic → 通过 mqttx.RequestReply 阻塞等待 services_reply → 超时或非 0 result 返回 error。

### 即发即忘（SendCommandFireAndForget）

`client.go:523-540`：不发 services_reply，不等待，只返回可能的 Publish 错误。用于不需要确认的场景。

### DRC 下行（publishDrcDown）

`client.go:1078-1089`：发布到 drc/down topic，即发即忘，无 services_reply。用于杆量控制、心跳、紧急停桨。

### 属性设置（SetProperty）

`client.go:546-571`：类似 SendCommand，但使用 property/set 主题 + property/set_reply 应答。

参考文件：
- `common/djisdk/client.go:484-571`

### 在线预检查

通过 `SetOnlineChecker` 设置后，`SendCommand` 每次调用前检查 `onlineChecker(gatewaySn)`，离线时快速拒绝。

## DRC 协议处理

### 上行反序列化（protocol_drc.go）

`DrcUnmarshalUpData(method, data)` 按 method 分派到具体类型：
- `heart_beat` → `DrcHeartBeatUpData`
- `drc_initial_state_subscribe` → `DrcInitialStateSubscribeUpData`
- `hsi_info_push` → `DrcHsiInfoPushData`
- `delay_info_push` → `DrcDelayInfoPushData`
- `osd_info_push` → `DrcOsdInfoPushData`
- 未知 method → `DrcUnknownUpData`

### JSON key 兼容

`DrcHsiInfoPushData` 自定义 `UnmarshalJSON` 处理 `around_distance`/`around_distances` 字段名差异。

## Topic 函数（topic.go）

每个 Topic 通道有一对函数：

```go
// 用于 Publish 下发（指定 gatewaySn）
topic := djisdk.ServicesTopic(gatewaySn)  // → "thing/product/{gateway_sn}/services"

// 用于 Subscribe 通配模式（固定 Pattern）
pattern := djisdk.ServicesReplyTopicPattern()  // → "thing/product/+/services_reply"
```

支持的 Topic 组：`services`、`services_reply`、`events`、`events_reply`、`property/set`、`property/set_reply`、`osd`、`state`、`status`、`status_reply`、`requests`、`requests_reply`、`drc/down`、`drc/up`。

## 错误处理

- `djisdk.NewDJIError(code)` — 通过 protobuf 枚举名 + 中文描述构造 `DJIError`
- `djisdk.IsDJIError(err)` — 类型断言，返回 `(bool, *DJIError)`
- `PlatformResultOK(0)` / `PlatformResultHandlerError(1)` / `PlatformResultTimeout(2)` — status_reply/events_reply/requests_reply 的 data.result 取值。
  - `PlatformResultTimeout(2)` **只能用于**实际超时场景，不能做其他占位（与 DJI 协议约定对齐）。

参考文件：
- `common/djisdk/error.go`
- `common/djisdk/error_descriptions.go`
- `common/djisdk/protocol.go:7-11`

## 常见陷阱

1. **`tryDispatchEventNotify` 是 switch + 函数指针**：添加事件类型要改 3 处（method 常量、handler 字段 + setter、case 分支），容易遗漏。记得同时更新 `doc.go` 的协议方向说明。
2. **`MustNewClient` panic**：调用方如果无法接受 panic，应自行处理 MQTT 连接失败逻辑或使用 `NewClient` 复用已有连接。
3. **`OnUpdateTopo` 已废弃**：`client.go:430-432` 保留 `OnUpdateTopo` 作为 `OnTopoUpdate` 的别名，新代码请用后者。
4. **EventMethodFallback 只在预置 On* 未注册时生效**：预置 On* 注册后 EventMethodFallback 不会执行。
5. **ReplyOptions 是全局开关**：不影响 handler 本身的执行，只控制是否发布对应的 \_reply 消息。
6. **Handler 注册不是 goroutine safe**：所有 Setter 直接赋值函数指针字段，不支持并发注册。
