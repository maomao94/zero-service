# djisdk 包规范（DJI MQTT SDK）

> `common/djisdk/` 封装大疆上云 MQTT 协议，提供云平台侧的 Client 类型：MQTT 连接管理、Topic 通配订阅、设备上行分发（事件/遥测/状态/请求/DRC）、下行命令发送（services/property/drc/down）和应答路由。

## 文件组织

| 文件 | 职责 |
|------|------|
| `doc.go` | 包级 godoc，协议方向说明 |
| `client.go` | `Client` 结构体、`Config` 结构体、`MustNewClient`/`NewClient`/`buildClient`、`SendCommand`/`SendCommandFireAndForget`/`SetProperty`、全部命令方法、DRC Manager API、`Close` |
| `option.go` | `ReplyConfig`/`DefaultReplyConfig`、`handlers` 聚合 struct、`ClientOption` 类型、`clientOptions`、全部 `WithXxx` option 函数、`defaultClientOptions`/`applyOptions` |
| `handler.go` | 上行分发：`HandleEvents`/`HandleOsd`/`HandleState`/`HandleStatus`/`HandleRequests`/`HandleDrcUp`、`tryDispatch*`/`reply*`、`SubscribeAll`、reply router 工厂 |
| `protocol.go` | 全部消息结构体：请求/应答/事件/遥测/属性 |
| `protocol_drc.go` | DRC 专属协议结构体、上行反序列化 dispatch、摘要函数 |
| `method.go` | DJI Cloud API method 字符串常量，按功能模块分组 |
| `topic.go` | MQTT Topic 构造函数和通配符 Pattern 函数 |
| `error.go` | `DJIError` 类型 + `IsDJIError` 断言 |
| `error_descriptions.go` | 错误码中文描述映射表 |
| `drc.go` | DRC 会话管理器（`drcManager`）、`drcDeviceSession`、`DrcConfig`/`DefaultDrcConfig` + 零值防护、SessionHook |

## Client 构造

### 工厂方法

两个构造入口（`client.go`）：

```go
// go-zero 风格：Config struct 携带连接与 SDK 配置，opts 仅注册 handler
c := djisdk.MustNewClient(djisdk.Config{
    MqttConfig: mqttCfg,
    PendingTTL: 30 * time.Second,
    Reply:      djisdk.DefaultReplyConfig(),
    Drc:        djisdk.DrcConfig{HeartbeatInterval: 2 * time.Second, HeartbeatTimeout: 300 * time.Second},
}, handlerOpts...)

// 复用已有 mqttx.Client
c := djisdk.NewClient(existingMqttClient, opts...)
```

`MustNewClient` 内部：`defaultClientOptions()` → Config 字段直写 `clientOptions` → 用户 opts 覆写 → `buildClient`。不通过 option→apply 间接转换。

### Config 结构体

```go
type Config struct {
    MqttConfig mqttx.MqttConfig
    PendingTTL time.Duration `json:",default=30s"`
    Reply      ReplyConfig   `json:",optional"`
    Drc        DrcConfig     `json:",optional"`
}
```

- `PendingTTL`: 0 使用默认 30s（`> 0` 才覆写）
- `Reply`: 值类型，零值 `{false,false,false}` 表示全部禁用；yaml 加载时 go-zero 填 `default=true`
- `Drc`: 值类型 + `json:",optional"`。零值（`HeartbeatInterval=0`）不创建 drcManager；yaml 配置后 `buildClient` 自动启用
- 应用层 `config.Config` 直接内嵌 `Dji djisdk.Config`，一行传参：`djisdk.MustNewClient(c.Dji, handlerOpts...)`

### Option 列表（配置类）

| Option | 类型 | 说明 |
|--------|------|------|
| `WithPendingTTL(ttl)` | `time.Duration` | services_reply 等待超时（默认 30s） |
| `WithReplyConfig(cfg)` | `ReplyConfig` | 全局开关 events_reply/status_reply/requests_reply |
| `WithOnlineChecker(checker)` | `func(gatewaySn string) bool` | 命令发送前在线预检 |
| `WithDrcConfig(cfg)` | `DrcConfig` | 启用 DRC 会话管理（心跳间隔、超时） |
| `WithDrcSessionEnabled(hook)` | `DrcSessionEnabledHook` | DRC 会话启用回调 |
| `WithDrcSessionDisabled(hook)` | `DrcSessionDisabledHook` | DRC 会话停用回调 |
| `WithDrcSessionExpired(hook)` | `DrcSessionExpiredHook` | DRC 会话过期回调 |

## Handler 注册（Option 模式）

Handler 通过 `WithXxx` option 在构造时注入。注册是 last-wins，不支持并发注册。

17 个 handler 回调聚合在未导出 `handlers` struct（`option.go`），`Client` 和 `clientOptions` 均按值持有。`buildClient` 一行 `c.handlers = opt.handlers` 完成映射。

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
- `StatusHandler` `func(ctx, gatewaySn, *StatusMessage) int` — 返回 result 码
- `RequestHandler` `func(ctx, gatewaySn, *RequestMessage) (result int, output any, err error)` — 返回 result + output
- `DrcUpHandler` `func(ctx, gatewaySn, *DrcUpMessage, parsed any) error` — parsed 已反序列化

### 事件分发

`HandleEvents`（`handler.go`）：
1. **预置分支**（`tryDispatchEventNotify`）：switch on method，命中已注册 handler 则执行
2. **默认**：未命中打印 payload 日志

### 添加新事件类型

改动点：**2 处**（handler 聚合后从 4 处降为 2 处）
1. `handlers` struct 加字段（`option.go`）
2. `WithXxx` option 函数（`option.go`）

handler 注册自动通过 `buildClient` 的 `c.handlers = opt.handlers` 传播到 `Client`。

### 注册示例

```go
opts := []djisdk.ClientOption{
    djisdk.WithFlightTaskProgressHandler(myHandler),
    djisdk.WithOsdHandler(myOsdHandler),
    djisdk.WithDrcConfig(djisdk.DrcConfig{
        HeartbeatInterval: 2 * time.Second,
        HeartbeatTimeout:  300 * time.Second,
    }),
    djisdk.WithDrcSessionEnabled(func(gatewaySn, sessionID string) { ... }),
}
c := djisdk.MustNewClient(djisdk.Config{MqttConfig: mqttCfg, Reply: djisdk.DefaultReplyConfig()}, opts...)
```

## DRC 会话管理

DRC Manager 内置于 Client（`drc.go`），通过 `Config.Drc`（值类型）或 `WithDrcConfig` option 激活。

### 暴露的 API（在 Client 上）

| 方法 | 说明 |
|------|------|
| `EnableDrc(ctx, gatewaySn, opts...)` | 启用设备 DRC 模式，支持 `WithDrcMaxTimeout(d)` |
| `DisableDrc(ctx, gatewaySn)` | 停用设备 DRC 模式 |
| `DrcNextSeq(gatewaySn)` | 获取递增序号（杆量控制用） |
| `DrcStatus(gatewaySn)` | 查询设备 DRC 状态快照 |

**无 manager 时调用上述方法返回错误**，不静默通过。

### 零值防护

`newDrcManager` 入口对 `HeartbeatInterval`/`HeartbeatTimeout` 零值自动填充默认值（2s/300s），防止 `time.NewTicker(0)` panic。`DefaultDrcConfig()` 提供公开默认构造。

### 心跳桥接

`HandleDrcUp` 收到 `heart_beat` 上行时，自动调用 `drcManager.OnDeviceHeartbeat()` 刷新存活时间，再调用外部 `onDrcUp` handler。

## 命令下发

### 阻塞模式（SendCommand）

```go
tid, err := c.SendCommand(ctx, gatewaySn, method, data)
```

流程：UUID (tid/bid) → ServiceRequest → Publish → mqttx.RequestReply 阻塞等待 services_reply。

### 即发即忘（SendCommandFireAndForget）

不等待应答，只返回 Publish 错误。

### DRC 下行（publishDrcDown）

发布到 drc/down topic，即发即忘。用于杆量控制、心跳、紧急停桨。

### 在线预检查

通过 `WithOnlineChecker` 设置后，`SendCommand` 每次调用前检查 `onlineChecker(gatewaySn)`。

## 错误处理

- `djisdk.NewDJIError(code)` — 通过 protobuf 枚举名 + 中文描述构造 `DJIError`
- `djisdk.IsDJIError(err)` — 类型断言
- `PlatformResultOK(0)` / `PlatformResultHandlerError(1)` / `PlatformResultTimeout(2)` — reply 包 result 取值

## 常见陷阱

1. **添加事件类型改 2 处**：`handlers` struct 加字段 + `WithXxx` option 函数
2. **`MustNewClient` 接受 `Config` 值类型**：`Config.Reply`/`Config.Drc` 均为值类型，零值有明确语义（Reply 全禁用 / Drc 禁用）
3. **`Config.Drc` 零值不创建 drcManager**：`buildClient` 中 `HeartbeatInterval > 0` 为守门条件
4. **DRC 心跳通知由 Client 内部处理**：`WithDrcUpHandler` 只需注册业务逻辑，无需手动管理 `OnDeviceHeartbeat`
5. **`ReplyConfig` 是全局开关**：不影响 handler 执行，只控制是否发布 _reply 消息
6. **应用层用 `Config` 嵌入消重**：`config.Config` 直接 `Dji djisdk.Config`，一行 `MustNewClient(c.Dji, handlerOpts...)` 创建