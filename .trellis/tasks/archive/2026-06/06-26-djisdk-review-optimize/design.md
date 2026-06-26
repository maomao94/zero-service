# design: djisdk 代码审阅优化

## Architecture Decisions

### D1: handlers struct 聚合

**问题**：`Client` 和 `clientOptions` 各自持有 17 个 `onXxx` 回调，`buildClient` 逐字段手工映射。新增 handler 需改 4 处。

**决定**：引入未导出 `handlers` struct 聚合全部回调 + `onlineChecker`，`Client` 和 `clientOptions` 均按值持有。

```go
// 新增结构
type handlers struct {
    onFlightTaskProgress               func(ctx context.Context, gatewaySn string, data *FlightTaskProgressEvent)
    onFlightTaskReady                  func(ctx context.Context, gatewaySn string, data *FlightTaskReadyEvent)
    // ... 17 个回调 ...
    onDrcUp                            DrcUpHandler
    onlineChecker                      func(gatewaySn string) bool
}

// Client 变化（缩减到核心字段）
type Client struct {
    mqttClient   mqttx.Client
    handlers     handlers        // ← 聚合
    pendingTTL   time.Duration
    replyOptions ReplyOptions
    drcManager   *drcManager
}

// clientOptions 同样持有 handlers
type clientOptions struct {
    handlers        handlers
    pendingTTL      time.Duration
    replyOptions    ReplyOptions
    drcConfig       DrcConfig
    drcManagerOpts  []drcManagerOption
}

// buildClient 一次赋值
func buildClient(mqttClient mqttx.Client, opt *clientOptions) *Client {
    c := &Client{
        mqttClient:   mqttClient,
        handlers:     opt.handlers,  // ← 一行替代 17 行逐字段赋值
        pendingTTL:   opt.pendingTTL,
        replyOptions: opt.replyOptions,
    }
    // ...
}
```

**Why not exported?** `handlers` 是内部实现细节。外部通过 `WithXxx` option 注入，不需要也不应暴露此 struct。

**Why by value (not pointer)?** 零分配，与当前行为一致。`Client` 创建后 handler 不可变，无复制竞态。

**新增 handler 改动点变化**：4 处 → 2 处
- `handlers` struct 加字段
- `WithXxx` option 函数
- ~~clientOptions struct 加字段~~ → handlers 已内嵌
- ~~Client struct 加字段~~ → handlers 已内嵌
- ~~buildClient 映射行~~ → 自动覆盖

### D2: client.go 文件拆分

**决定**：按 Go 惯例，拆出两个文件：

| 文件 | 内容 | 行数（估算） |
|-------|------|------------|
| `option.go` | `ClientOption` 类型、`clientOptions` struct、`handlers` struct、全部 `WithXxx` 函数、`defaultClientOptions`、`applyOptions`、`DefaultReplyOptions`、`ReplyOptions` | ~190 |
| `handler.go` | `HandleEvents`、`HandleOsd`、`HandleState`、`HandleStatus`、`HandleRequests`、`HandleDrcUp`、`tryDispatchEventNotify`、`tryDispatchStatusNotify`、`replyEvent`、`replyStatus`、`replyToRequest`、`extractDeviceSnFromTopic`、`logFields`、topic handler 注册（`SubscribeAll`） | ~560 |
| `client.go` | `Client` struct、`MustNewClient`/`NewClient`/`buildClient`、`SendCommand`/`SendCommandFireAndForget`、`SetProperty`、全部命令方法、DRC Manager API、`Close`、`replyRouters`/router 工厂 | ~1130 → ~700（移走 handler/option） |

**Why these boundaries?**
- `option.go`：Go 惯例——"config/option" 是天然的独立关注点。用户明确指出"config 可以拆"。
- `handler.go`：handler 分发逻辑与 Client 命令方法互不依赖，拆出不引入循环引用。
- `client.go`：保留 Client 核心——构造、命令、DRC API。这些方法与 `Client` 类型定义在同一文件最符合 Go 惯例（如 `net/http` 的 `server.go`）。

**Why NOT 按命令域拆分 (cmd_wayline.go etc)?**
- 命令方法多是 1-3 行的 `SendCommand` 薄封装，拆散后定位反而困难
- Go 社区不鼓励一个 type 的方法分散到 10 个文件
- 当前命令方法按 DjiCloud API 分组已有清晰的代码分区注释，足矣

### D3: DrcConfig 默认值保护

**问题**：`newDrcManager` 中 `time.NewTicker(0)` panic；`HeartbeatTimeout=0` 导致 IsAlive 异常。

**决定**：两层防护

1. **公开默认函数**：`DefaultDrcConfig()` 返回 `DrcConfig{HeartbeatInterval: 2s, HeartbeatTimeout: 300s}`

2. **构造时兜底**：`newDrcManager` 内部对零值字段填充默认值
```go
func newDrcManager(client *Client, cfg DrcConfig, opts ...drcManagerOption) *drcManager {
    if cfg.HeartbeatInterval <= 0 {
        cfg.HeartbeatInterval = 2 * time.Second
    }
    if cfg.HeartbeatTimeout <= 0 {
        cfg.HeartbeatTimeout = 300 * time.Second
    }
    // ...
}
```

**Why not only `WithDrcConfig` option 中做?** `WithDrcConfig` 只是透传；实际使用在 `newDrcManager`，防护点应靠近使用点。两层任一触发均可防止 panic。

**Why 这些默认值?** 与大疆 DRC 协议推荐值及 `config.DrcConfig` 的 go-zero `default` 标签一致（2s 间隔 / 300s 超时）。

### D4: Config struct 统一初始化配置

**问题**：当前 `MustNewClient(config mqttx.MqttConfig, opts...)` 接收 MQTT 配置，但 `PendingTTL`/`ReplyOptions`/`DrcConfig` 需通过独立 option 传入，导致调用方出现三行"配置类 option"与 handler option 混在同一个列表。

**决定**：新增 `Config` struct，内嵌 `mqttx.MqttConfig`，将非 handler 的配置项收束：

```go
type Config struct {
    mqttx.MqttConfig
    PendingTTL   time.Duration
    ReplyOptions ReplyOptions
    Drc          *DrcConfig  // nil = DRC 禁用
}
```

`MustNewClient` 签名变更：

```go
// before
func MustNewClient(config mqttx.MqttConfig, opts ...ClientOption) *Client

// after
func MustNewClient(config Config, opts ...ClientOption) *Client
```

内部实现：Config 字段在 `MustNewClient` 中转为 `clientOptions`（若零值则填默认值），再与 handler opts 合并。

```go
func MustNewClient(config Config, opts ...ClientOption) *Client {
    opts = append([]ClientOption{
        withPendingTTL(config.PendingTTL),
        withReplyOptions(config.ReplyOptions),
    }, opts...)
    if config.Drc != nil {
        opts = append(opts, withDrcConfig(*config.Drc))
    }
    opt := applyOptions(opts...)
    return buildClient(mqttx.MustNewClient(config.MqttConfig, replyRouters(opt.pendingTTL)...), &opt)
}
```

**With* option 降级**：`WithPendingTTL`/`WithReplyOptions`/`WithDrcConfig` 改为未导出 `with` 前缀，仅 `NewClient` 内部调用（外部通过 Config 设置）。handler 类 option 保持导出。

**调用方变化**（servicecontext.go）：

```go
// before
djiOpts := []djisdk.ClientOption{
    djisdk.WithPendingTTL(c.PendingTTL),
    djisdk.WithReplyOptions(djisdk.ReplyOptions{...}),
    djisdk.WithDrcConfig(djisdk.DrcConfig{...}),
}

// after
djiCli := djisdk.MustNewClient(djisdk.Config{
    MqttConfig:   c.MqttConfig,
    PendingTTL:   c.PendingTTL,
    ReplyOptions: djisdk.ReplyOptions{...},
    Drc:          &djisdk.DrcConfig{...},
}, handlerOpts...)
```

**Why 内嵌 MqttxConfig?** 用户明确"直接继承 mqtt 配置"——`Config` 就是 MQTT 配置的超集。

**Why Drc 是指针?** `nil` 表示未启用，与当前 `HeartbeatInterval=0` 判断启用逻辑一致（`buildClient` 中 `opt.drcConfig.HeartbeatInterval > 0` 才创建 `drcManager`）。

## Compatibility

- 所有导出 API (`WithXxx` option 函数、`Client` 方法、`Client` 类型名、`NewClient`) **签名不变**
- `MustNewClient` 签名为 **breaking change**：`mqtxx.MqttConfig` → `Config`；调用方 servicecontext.go 同步更新
- `WithPendingTTL`/`WithReplyOptions`/`WithDrcConfig` 降为未导出（`NewClient` 仍可使用内部版本）
- `droneEmergencyStopLogic.go:36` 等引用 `DrcNextSeq` 的逻辑文件不受影响
- 删除的 `internal/drc/` 无生产引用，无 breaking change
- `subscribeAll` 从 `client.go` 移至 `handler.go`，方法签名不变

## Rollback

- 若 handlers 聚合出现问题：恢复 `handlers` 字段到 `Client` + `clientOptions` 拍平，无需回滚其他变更
- 若文件拆分出现问题：合并回 `client.go` 即可，无逻辑变更
- `git revert` 可整体回滚
