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
| `error.go` | `DJIError` 类型 + `IsDJIError` 断言、`PlatformError` 类型 + `ResultFromError` |
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
    Reply      ReplyConfig
    Drc        DrcConfig
}
```

- `PendingTTL`: 0 使用默认 30s（`> 0` 才覆写）
- `Reply`: 值类型，零值 `{false,false,false}` 表示全部禁用；yaml 加载时 go-zero 根据字段 tag 填充
- `Drc`: 值类型。零值（`HeartbeatInterval=0`）不创建 drcManager；yaml 配置后 `buildClient` 自动启用。`Drc.Address` 为 DRC 公网 MQTT Broker 地址（机巢公网可达），留空则复用 `MqttConfig.Broker[0]`
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

14 个 handler 回调聚合在未导出 `handlers` struct（`option.go`），`Client` 和 `clientOptions` 均按值持有。`buildClient` 一行 `c.handlers = opt.handlers` 完成映射。

### Handler Option 列表

| Option | Handler 签名 | 方向 | 对应 Topic |
|--------|-------------|------|-----------|
| `WithFlightTaskProgressHandler` | `func(ctx, gatewaySn, *FlightTaskProgressEvent) error` | up | events |
| `WithFlightTaskReadyHandler` | `func(ctx, gatewaySn, *FlightTaskReadyEvent) error` | up | events |
| `WithReturnHomeInfoHandler` | `func(ctx, gatewaySn, *ReturnHomeInfoEvent) error` | up | events |
| `WithCustomDataFromPsdkHandler` | `func(ctx, gatewaySn, *CustomDataFromPsdkEvent) error` | up | events |
| `WithCustomDataFromEsdkHandler` | `func(ctx, gatewaySn, *CustomDataFromEsdkEvent) error` | up | events |
| `WithHmsEventNotifyHandler` | `func(ctx, gatewaySn, *HmsEventData) error` | up | events |
| `WithRemoteLogFileUploadProgressHandler` | `func(ctx, gatewaySn, *RemoteLogFileUploadProgressEvent) error` | up | events |
| `WithOtaProgressHandler` | `func(ctx, gatewaySn, *OtaProgressEvent) error` | up | events |
| `WithUpdateTopoHandler` | `func(ctx, gatewaySn, *TopoUpdateData) error` | up | status |
| `WithOsdHandler` | `func(ctx, deviceSn, *OsdMessage) error` | up | osd |
| `WithStateHandler` | `func(ctx, deviceSn, *StateMessage) error` | up | state |
| `WithStatusHandler` | `StatusHandler` | up | status |
| `WithRequestHandler` | `RequestHandler` | up | requests |
| `WithDrcUpHandler` | `DrcUpHandler` | up | drc/up |

全部 Handler 统一返回 `error`，SDK 层通过 `ResultFromError(err)` 提取 `PlatformError.Code` 作为 _reply 的 result：
- `StatusHandler` `func(ctx, gatewaySn, *StatusMessage) error` — 返回 nil 即 OK，否则用 `ResultFromError` 取码
- `RequestHandler` `func(ctx, gatewaySn, *RequestMessage) (output any, err error)` — output 写入 _reply.data.output
- `DrcUpHandler` `func(ctx, gatewaySn, *DrcUpMessage, parsed any) error` — parsed 已反序列化

### 事件分发

`HandleEvents`（`handler.go`）：
1. **上下文注入**：解析完 `EventMessage` 后，将 `gateway_sn`/`method`/`tid`/`bid`/`need_reply`/`ts`/`ts_fmt` 注入 `ctx`，传递给 `tryDispatchEventNotify` 和所有 handler 回调
2. **预置分支**（`tryDispatchEventNotify`）：switch on method，命中已注册 handler 则执行。handler 返回 `error` 时通过 `ResultFromError(err)` 提取 `PlatformError.Code` 作为 events_reply 的 result
3. **未命中 method**：打印 `"no handler for event"`，不输出原始 payload
4. **默认**：未命中且无 handler 时，仅打印 method 和 payload 字节数

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

### DrcConfig 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `HeartbeatInterval` | `time.Duration` | 心跳间隔（默认 2s） |
| `HeartbeatTimeout` | `time.Duration` | 心跳超时（默认 300s） |
| `Address` | `string` | DRC 公网 Broker 地址（如 `public.example.com:1883`），机巢需公网可达。留空则复用主 `MqttConfig.Broker[0]`。见 `app/djicloud/internal/logic/drchelper.go:toDrcMqttBroker`

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

## 日志前缀

`common/djisdk/` 内所有 SDK 层日志和错误文本使用 `[dji-sdk]` 作为首级前缀，动作维度放在前缀后，例如：

- `client.go` / `handler.go`: `[dji-sdk] send_command`、`[dji-sdk] drc_up`
- `drc.go`: `[dji-sdk] drc_manager ...`、`[dji-sdk] drc_heartbeat ...`、`[dji-sdk] drc_clean ...`
- 应用层注册的 DRC hook 如果记录 SDK 回调失败，也保持 `[dji-sdk] drc_manager ...`，例如 `app/djicloud/internal/svc/servicecontext.go`

不要恢复迁移前的 `[drc-manager]`、`[drc-heartbeat]`、`[drc-clean]` 首级前缀；否则同一 SDK 包的日志检索口径会分裂。

## 命名规约

### Method 常量 → 客户端方法 → Proto RPC

四层命名以 **DJI 原始 method 字面值** 为锚点对齐：

| 层 | 示例 |
|---|---|
| DJI method 值 | `flighttask_undo` |
| SDK 常量 | `MethodFlightTaskUndo` |
| SDK 客户端方法 | `FlightTaskUndo` |
| Proto RPC | `FlightTaskUndo` |
| Proto Message | `FlightTaskUndoReq` |

规则：
- `Method` 常量名 = `Method` + DJI method CamelCase（去掉下划线，首字母大写）
- 客户端方法名 = 常量名去掉 `Method` 前缀
- Proto RPC 名 = 客户端方法名
- Proto Message 名 = RPC 名 + `Req`/`Res`

不得在约定层引入与 DJI 原始 method 不一致的别名（如用 `Cancel` 代替 `Undo`、用 `FloatUp` 代替 `UIResourceUpload`）。

例外：平台自有接口（非 DJI 标准 method）不受此约束，用易于理解的业务命名即可。

## SDK 内部错误日志

`client.go` 中所有下发方法（`SendCommand`、`SendCommandFireAndForget`、`PropertySet`、`publishDrcDown` 等）在返回 `error` 前，必须使用 `logx.WithContext(ctx).Errorf` 记录一条错误日志。这样应用层逻辑文件无需重复调用 `l.Errorf`，只需：

```go
tid, err := l.svcCtx.DjiClient.SomeMethod(l.ctx, ...)
if err != nil {
    return errRes(tid, err), nil
}
```

## 常见陷阱

1. **添加事件类型改 2 处**：`handlers` struct 加字段 + `WithXxx` option 函数
2. **`MustNewClient` 接受 `Config` 值类型**：`Config.Reply`/`Config.Drc` 均为值类型，零值有明确语义（Reply 全禁用 / Drc 禁用）
3. **`Config.Drc` 零值不创建 drcManager**：`buildClient` 中 `HeartbeatInterval > 0` 为守门条件
4. **DRC 心跳通知由 Client 内部处理**：`WithDrcUpHandler` 只需注册业务逻辑，无需手动管理 `OnDeviceHeartbeat`
5. **`ReplyConfig` 是全局开关**：不影响 handler 执行，只控制是否发布 _reply 消息
6. **应用层用 `Config` 嵌入消重**：`config.Config` 直接 `Dji djisdk.Config`，一行 `MustNewClient(c.Dji, handlerOpts...)` 创建
7. **上下文注入**：每个 handler 入口解析协议消息后必须调用 `logx.ContextWithFields` 注入 `gateway_sn`/`method`/`tid`/`bid`/`ts`/`ts_fmt`；消息文本保持干净（`"[dji-sdk] events"`），不重复 ctx 已有字段。时间戳使用 `tsFields` 辅助函数
8. **禁止 payload 明文**：`logFields` 已是当前唯一拼接函数；新增日志禁止打印完整 `payload`、`raw` 或 `value` 原文
9. **DRC 日志前缀不要私有化**：`drc.go` 内部组件用 `[dji-sdk] drc_manager` / `drc_heartbeat` / `drc_clean`，不要用 `[drc-manager]` 这类首级前缀
