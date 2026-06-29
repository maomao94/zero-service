# MQTT 客户端规范

> `common/mqttx/` 包是对 Eclipse Paho MQTT 客户端的封装，提供发布、订阅、handler 注册、reply-router 模式和 OTel 链路追踪。

## When to read

- 创建或修改 MQTT 客户端连接、配置、topic handler 注册
- 使用 `WithReplyRouter` 实现 request/reply 模式
- 排查 `ConsumeHandler` 未触发、`PublishWithTrace` trace 断裂或 reply pool 超时
- 如涉及消息队列（Kafka/Asynq）请改读 [`messaging-guidelines.md`](./messaging-guidelines.md)

## 包结构

```
common/mqttx/
├── client.go          # Client 接口 + mqttClient 实现：连接、发布、handler 注册、生命周期
├── config.go          # MqttConfig + ClientOption（WithOnReady, WithReplyRouter）
├── dispatcher.go      # handlerManager + messageDispatcher 消息分发
├── errors.go          # 哨兵错误：ErrNilDecoder, ErrNoReplyRouter, ErrReplyType, ErrReplyNotMatched
├── message.go         # Message 类型
├── reply_router.go    # ReplyRouter[T] 泛型 request/reply 匹配
├── request_replyer.go # RequestReply[T] 泛型入口函数
├── topic_log.go       # MessageLogFunc topic 日志
├── config_test.go     # 配置验证
└── reply_router_test.go
```

## 构造方式

```go
// 启动时推荐：panic 快速失败 + 自动注册关闭监听
cli := mqttx.MustNewClient(cfg, opts...)

// 测试或可选连接
cli, err := mqttx.NewClient(cfg, opts...)
```

`MustNewClient` 内部调用 `logx.Must` 和 `proc.AddShutdownListener`，业务层无需再处理关闭。

## Client 接口

```go
type Client interface {
    AddHandler(topicTemplate string, handler ConsumeHandler) error
    AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error
    Publish(ctx context.Context, topic string, payload []byte) error
    PublishWithTrace(ctx context.Context, topic string, payload []byte) (string, error)
    Close()
    GetClientID() string
}
```

`PublishWithTrace` 返回 MQTT message ID，用于日志关联和 reply pool。

## handler 注册

### 普通 handler（`AddHandler`）

```go
cli.AddHandler("device/+/data", mqttx.ConsumeHandlerFunc(func(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
    return processData(ctx, payload)
}))
```

- `topicTemplate` 支持 MQTT 通配符 `+` 和 `#`
- 同一 topic template 可注册多个 handler，按注册顺序依次执行
- `topic` 参数为实际匹配的完整 topic，`topicTemplate` 为注册时的模板

### Reply router handler（`WithReplyRouter`）

```go
router := mqttx.NewReplyRouter[*types.ReplyPayload](
    mqttx.ReplyDecoderFunc[*types.ReplyPayload](decodeReply),
    mqttx.WithReplyRouterTTL(10*time.Second),
    mqttx.WithReplyRouterName("my-reply-router"),
)

cli := mqttx.MustNewClient(cfg, mqttx.WithReplyRouter("device/+/reply", router))
```

`WithReplyRouter` 的 handler 优先级高于 `AddHandler` 的普通 handler。dispatch 路径见 `dispatcher.go`。

### Request/Reply 调用

```go
tid, _ := tool.SimpleUUID()
result, err := mqttx.RequestReply[*types.ReplyPayload](ctx, cli, "device/+/reply", tid, func() error {
    return cli.PublishWithTrace(ctx, "device/123/command", payload)
}, 5*time.Second)
```

- `RequestReply` 是包级泛型函数（Go 不支持泛型方法）
- tid 由调用方生成，保证不依赖 OTel trace context（异步场景 trace ID 可能断裂）
- `send` 函数负责发布请求，`ttl` 覆盖 `ReplyRouter` 的默认超时

## ReplyRouter 设计

```go
type ReplyRouter[T any] struct {
    pool   *antsx.ReplyPool[T]   // 底层 reply pool
    decode ReplyDecoder[T]       // 协议层解析
}

// 协议层负责：topic 解析、payload 反序列化、设备标识提取
type ReplyDecoder[T any] interface {
    Decode(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[T], error)
}
```

- `ReplyRouter` 实现 `ConsumeHandler` 接口
- 消息到达时调用 `decode.Decode` 提取 tid，然后 `Resolve(tid, value)`
- `ReplyRouter.Consume` 返回 `ErrReplyNotMatched` 表示解码成功但无匹配的 pending 请求——这是正常情况（如重复或历史消息）

## 常用模式

### 连接就绪后注册 handler

```go
cli := mqttx.MustNewClient(cfg, mqttx.WithOnReady(func(c mqttx.Client) {
    c.AddHandler("device/+/status", statusHandler)
}))
```

`WithOnReady` 在首次连接成功时执行一次。断线重连后**不会**重复执行——重连时 handler 注册在 `handlerManager` 中保持。

### 多 topic handler 注册

使用同一个 `Client` 实例注册多个 handler，无需创建多个 MQTT 客户端。

## 注意事项

### handler 内 context 不可持久化

`ConsumeHandler` 的 `ctx` 在 handler 返回后可能被 cancel。需在 handler 外使用时复制关键字段或使用 `context.WithoutCancel`。

### 通配符不匹配

`topicTemplate` 格式必须完全匹配 MQTT topic 层级：
- `+` 匹配**一个**层级
- `#` 匹配**零个或多个**层级，只能在结尾
- 推荐在 `config.go` 的 `SubscribeTopics` 中声明不通配的具体 topic，handler 注册时才用通配符

### reply pool 泄漏

`ReplyRouter` 内部使用 `antsx.ReplyPool`，TTL 到期自动清理。但 `RequestReply` 超时后 `send` 仍可能完成，reply 到达时 tid 已过期——返回 `ErrReplyNotMatched` 是正常行为，非 bug。

### 使用 `err == nil` 判断类型而非 `errors.Is`

```go
if routerError == mqttx.ErrReplyNotMatched { // ✅ 正确：sentinel 值比较
if errors.Is(routerError, mqttx.ErrReplyNotMatched) { // ❌ 不必要
```

从 `ConsumeHandler` 返回的 error 没有 wrap，直接比较即可。

### 参考文件

- `common/mqttx/client.go` — 核心接口和实现
- `common/mqttx/reply_router.go` — ReplyRouter[T] 实现
- `common/mqttx/request_replyer.go` — RequestReply[T] 泛型入口
- `common/mqttx/errors.go` — 哨兵错误定义
- `app/ieccaller/mqtt/broadcast.go` — 使用 reply-router 的实际案例
- `app/ieccaller/internal/svc/servicecontext.go` — ReplyRouter 注册和 decodeBroadcastAck 示例
