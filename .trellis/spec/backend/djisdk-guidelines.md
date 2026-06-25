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
| `drc.go` | DRC 会话管理器、DeviceSession、DrcConfig、SessionHook |

## Client 构造

### 工厂方法

两个构造入口（`client.go:239-257`）：

```go
// 自建 MQTT 连接
c := djisdk.MustNewClient(mqttConfig, opts...)

// 复用已有 mqttx.Client
c := djisdk.NewClient(existingMqttClient, opts...)
```

`MustNewClient` 在 MQTT 连接失败时 panic，调用方不能优雅降级。

构造函数使用 `applyOptions(opts...) → buildClient()` 模式，避免重复解析 options。

### Option 列表

| Option | 类型 | 说明 |
|--------|------|------|
| `WithPendingTTL(ttl)` | `time.Duration` | services_reply 等待超时（默认 30s） |
| `WithReplyOptions(ro)` | `ReplyOptions` | 全局开关 events_reply/status_reply/requests_reply |
| `WithOnlineChecker(checker)` | `func(gatewaySn string) bool` | 命令发送前在线预检 |
| `WithDrcConfig(cfg)` | `DrcConfig` | 启用 DRC 会话管理（心跳间隔、超时） |
| `WithDrcSessionEnabled(hook)` | `DrcSessionEnabledHook` | DRC 会话启用回调 |
| `WithDrcSessionDisabled(hook)` | `DrcSessionDisabledHook` | DRC 会话停用回调 |
| `WithDrcSessionExpired(hook)` | `DrcSessionExpiredHook` | DRC 会话过期回调 |

## Handler 注册（Option 模式）

Handler **不再通过 Setter 方法注册**，统一使用 `WithXxx` option 在构造时注入。注册是 last-wins，不支持并发注册。

### Handler Option 列表

| Option | Handler 签名 | 方向 | 对应 Topic |
|--------|-------------|------|-----------|
| `WithFlightTaskProgressHandler` | `func(ctx, gatewaySn, *FlightTaskProgressEvent)` | up | events |
| `WithFlightTaskReadyHandler` | `func(ctx, gatewaySn, *FlightTaskReadyEvent)` | up | events |
| `WithReturnHomeInfoHandler` | `func(ctx, gatewaySn, *ReturnHomeInfoEvent)` | up | events |
| `WithCustomDataFromPsdkHandler` | `func(ctx, gatewaySn, *CustomDataFromPsdkEvent)` | up | events |
| `WithCustomDataFromEsdkHandler` | `func(ctx, gatewaySn, *CustomDataFromEsdkEvent)` | up | events |
| `WithHmsEventNotifyHandler` | `func(ctx, gatewaySn, *HmsEventData)` | up | events |
| `WithRemoteLogFileUploadProgressHandler` | `func(ctx, gatewaySn, *RemoteLogFileUploadProgressEvent)` | up | events |
| `WithOtaProgressHandler` | `func(ctx, gatewaySn, *OtaProgressEvent)` | up | events |
| `WithUpdateTopoHandler` | `func(ctx, gatewaySn, *TopoUpdateData)` | up | status |
| `WithOsdHandler` | `func(ctx, deviceSn, *OsdMessage)` | up | osd |
| `WithStateHandler` | `func(ctx, deviceSn, *StateMessage)` | up | state |
| `WithStatusHandler` | `StatusHandler` | up | status |
| `WithRequestHandler` | `RequestHandler` | up | requests |
| `WithDrcUpHandler` | `DrcUpHandler` | up | drc/up |

有返回值的 Handler：
- `StatusHandler` `func(ctx, gatewaySn, *StatusMessage) int` — 返回 result 码，ReplyOptions 控制是否发 status_reply
- `RequestHandler` `func(ctx, gatewaySn, *RequestMessage) (result int, output any, err error)` — 返回 result + output，ReplyOptions 控制是否发 requests_reply
- `DrcUpHandler` `func(ctx, gatewaySn, *DrcUpMessage, parsed any) error` — parsed 已由 DrcUnmarshalUpData 反序列化为具体类型或 `DrcUnknownUpData`

### 事件分发

`HandleEvents`（`client.go`）：
1. **预置 On\* 分支**（`tryDispatchEventNotify`）：switch on method，命中已注册的 On* handler 则执行
2. **默认行为**：未命中时打印 `logx.Infof("[dji-sdk] no handler for event method=%s, payload=%s")`，不做回调

添加新事件类型需要：增加 method 常量（`method.go`）、增加 option 函数（`client.go`）、增加 handler 字段（`clientOptions` + `Client`）、增加 `case` 分支（`tryDispatchEventNotify`）、在 `buildClient` 中映射。

### 注册示例

```go
opts := []djisdk.ClientOption{
    djisdk.WithPendingTTL(30 * time.Second),
    djisdk.WithFlightTaskProgressHandler(myProgressHandler),
    djisdk.WithOsdHandler(myOsdHandler),
    djisdk.WithDrcConfig(djisdk.DrcConfig{
        HeartbeatInterval: 2 * time.Second,
        HeartbeatTimeout:  300 * time.Second,
    }),
    djisdk.WithDrcSessionEnabled(func(gatewaySn, sessionID string) { ... }),
}
c := djisdk.MustNewClient(mqttConfig, opts...)
```

## DRC 会话管理

DRC Manager 已内置于 Client（`drc.go`），构造时通过 `WithDrcConfig` + SessionHook 激活。

### 暴露的 API（在 Client 上）

| 方法 | 说明 |
|------|------|
| `EnableDrc(ctx, gatewaySn, opts...)` | 启用设备 DRC 模式，支持 `WithDrcMaxTimeout(d)` |
| `DisableDrc(ctx, gatewaySn)` | 停用设备 DRC 模式 |
| `DrcNextSeq(gatewaySn)` | 获取递增序号（杆量控制用） |
| `DrcStatus(gatewaySn)` | 查询设备 DRC 状态快照 |

**无 manager 时调用上述方法返回错误**，不静默通过。

### 心跳桥接

`HandleDrcUp` 收到 `heart_beat` 上行时，自动调用 `drcManager.OnDeviceHeartbeat()` 刷新存活时间，再调用外部 `onDrcUp` handler。业务方只需通过 `WithDrcUpHandler` 注册持久化/推送逻辑，无需手动管理心跳通知。

### 并发模型

详见 `drc-concurrency.md`：mark-and-sweep 模式 + 无交叉加锁 + atomic 字段保护。

## 命令下发

### 阻塞模式（SendCommand）

```go
tid, err := c.SendCommand(ctx, gatewaySn, method, data)
```

流程：生成 UUID (tid/bid) → 构建 ServiceRequest → Publish 到 services topic → 通过 mqttx.RequestReply 阻塞等待 services_reply。

### 即发即忘（SendCommandFireAndForget）

不等待应答，只返回可能的 Publish 错误。

### DRC 下行（publishDrcDown）

发布到 drc/down topic，即发即忘。用于杆量控制、心跳、紧急停桨。

### 在线预检查

通过 `WithOnlineChecker` 设置后，`SendCommand` 每次调用前检查 `onlineChecker(gatewaySn)`。

## 错误处理

- `djisdk.NewDJIError(code)` — 通过 protobuf 枚举名 + 中文描述构造 `DJIError`
- `djisdk.IsDJIError(err)` — 类型断言
- `PlatformResultOK(0)` / `PlatformResultHandlerError(1)` / `PlatformResultTimeout(2)` — reply 包的 result 取值。

## 常见陷阱

1. **添加事件类型要改 4 处**：method 常量、option 函数、handler 字段（clientOptions + Client）、case 分支、buildClient 映射。
2. **`MustNewClient` panic**：无法接受 panic 的场景使用 `NewClient` 复用已有连接。
3. **ReplyOptions 是全局开关**：不影响 handler 执行，只控制是否发布 _reply 消息。
4. **DRC 心跳通知由 Client 内部处理**：`WithDrcUpHandler` 只需注册业务逻辑（DB+推送），不需要手动调用 `OnDeviceHeartbeat`。
5. **无 DrcConfig 时 DRC API 返回错误**：不会静默跳过，调用方需处理错误或确保配置正确。
