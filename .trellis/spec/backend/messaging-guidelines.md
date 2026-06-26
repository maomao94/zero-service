# 消息队列与 Trace 传播规范

## Scenario: go-queue Kafka trace header propagation

### 1. Scope / Trigger
- Trigger: 修改 Kafka/go-queue producer、consumer、bridge 服务、消息 handler 签名或 OpenTelemetry trace 传播时必须读取本节。
- Applies to: `github.com/zeromicro/go-queue/kq` v1.2.2、`github.com/segmentio/kafka-go` message headers、项目内 `bridgekafka`/`ieccaller`/`iecstash` 等 Kafka 路径。
- Why: `kq.Pusher.PushWithKey` 会把 OTel trace 写入 Kafka 标准 headers，但 `kq.ConsumeHandler` 只暴露 `ctx/key/value`，且 `CommitInOrder` 路径不会从 headers 恢复 trace context。

### 2. Signatures
- Producer: `func (p *kq.Pusher) PushWithKey(ctx context.Context, key, v string) error`
- Consumer handler: `type kq.ConsumeHandler interface { Consume(ctx context.Context, key, value string) error }`
- Handler adapter: `kq.WithHandle(func(ctx context.Context, key, value string) error)`
- Config: `kq.KqConf.CommitInOrder bool`

### 3. Contracts
- `PushWithKey` constructs `kafka.Message{Key: []byte(key), Value: []byte(v)}` and injects OTel text-map fields into `kafka.Message.Headers`.
- Default W3C propagation writes `traceparent`, optional `tracestate`, and optional `baggage` when the global propagator includes baggage.
- Kafka headers are not part of message `Value`; downstream code that only receives `key/value` cannot inspect raw headers.
- Non-ordered go-queue consumer path extracts headers into `ctx` before invoking `Consume`.
- Ordered go-queue consumer path (`CommitInOrder: true`) in v1.2.2 invokes `Consume(context.Background(), key, value)` and does not extract Kafka headers into `ctx`.

### 4. Validation & Error Matrix
- `ctx` has active upstream span + `PushWithKey` used -> Kafka message must contain `traceparent` header.
- `Push` used -> same as `PushWithKey`, because `Push` delegates to `PushWithKey`.
- `KPush` used -> no OTel header injection; do not expect trace propagation.
- Consumer `CommitInOrder: false` -> handler `ctx` should contain extracted upstream span context.
- Consumer `CommitInOrder: true` on go-queue v1.2.2 -> handler `ctx` will not contain extracted upstream span context.
- Business handler needs raw Kafka headers -> go-queue `ConsumeHandler` is insufficient; use a custom `kafka-go` reader or wrapper that passes `kafka.Message`.

### 5. Good/Base/Bad Cases
- Good: bridge service uses `PushWithKey(ctx, key, value)` and consumes with a path that extracts headers before calling business logic.
- Base: service only needs key/value and accepts trace loss; document that `CommitInOrder: true` disables consumer-side trace recovery.
- Bad: assuming `Consume(ctx, key, value)` can read Kafka headers directly, or assuming `CommitInOrder: true` preserves upstream trace in `ctx`.

### 6. Tests Required
- Unit: construct a `kafka.Message`, inject through a `propagation.TextMapCarrier`, and assert `traceparent` exists in `msg.Headers`.
- Unit: extract from `msg.Headers` and assert `trace.SpanContextFromContext(ctx).IsValid()`.
- Integration/config: if a service requires ordered commits and trace continuity, add a regression test or documented verification proving the ordered path extracts headers before handler invocation.
- Search assertion: when changing Kafka trace paths, run `rg -n "CommitInOrder|PushWithKey|KPush|Consume\(ctx context.Context, key, value string\)" app common` and inspect impacted services.

### 7. Wrong vs Correct

#### Wrong

```go
func (h *Handler) Consume(ctx context.Context, key, value string) error {
	// Wrong: Kafka headers are not available through this signature.
	traceparent := valueHeader(ctx, "traceparent")
	_ = traceparent
	return nil
}
```

```yaml
KafkaConsumeConfig:
  CommitInOrder: true # Wrong when consumer-side trace continuity is required with go-queue v1.2.2.
```

#### Correct

```go
// Producer side: trace context is injected into kafka.Message.Headers.
err := pusher.PushWithKey(ctx, key, value)
```

```yaml
KafkaConsumeConfig:
  CommitInOrder: false # Correct when relying on go-queue v1.2.2 automatic header extraction.
```

If ordered commits and trace continuity are both required, do not rely on `kq.ConsumeHandler` alone. Introduce a reader/wrapper path that receives `kafka.Message`, extracts trace from `msg.Headers`, then calls business logic with the extracted `ctx` and commits in the required order.

## Scenario: Kafka bridge module (bridgekafka)

### 1. Scope / Trigger
- Trigger: 新增 Kafka 桥接模块或修改 go-queue producer 配置时必须读取本节。
- Applies to: `app/bridgekafka/` 模块，以及所有使用 `kq.Pusher` 做多 topic 动态推送的场景。
- Why: `kq.NewPusher` topic 在创建时固定，不支持运行时动态切换。需要预配置 topic 列表、创建多个 Pusher 实例并存 map。

### 2. Signatures
- Pusher 创建: `kq.NewPusher(brokers []string, topic string, opts ...PushOption) *kq.Pusher`
- 推送: `pusher.Push(ctx, value string) error` / `pusher.PushWithKey(ctx, key, value string) error`
- 消费注册: `kq.MustNewQueue(conf kq.KqConf, handler kq.ConsumeHandler) queue.MessageQueue`

### 3. Contracts

**Kafka push config 类型**（`common/configx/kqConfig.go`）:

项目提供三种 Kafka 配置类型，统一放在 `common/configx` 包中共享，各服务不在本地重复定义：

| 类型 | 用途 | 字段 |
|------|------|------|
| `KafkaPushConf` | 单 topic push（xfusionmock） | `Brokers []string`, `Topic string` |
| `KafkaMultiPushConf` | 同集群多 topic push（bridgekafka） | `Brokers []string`, `Topics []string` |
| `KafkaConsumerConf` | 消费配置（bridgekafka/iecstash） | 全字段（Brokers/Topic/Group/Conns/...），不含 `service.ServiceConf` |

**Design Decision: 单 topic 和多 topic push 不合并** — 合并为 `Topics []string` 会强制所有单 topic 配置写成数组语法（`Topics: [asdu]`），增加不必要的 YAML 层级。拆分为两种类型，单 topic 保持 `Topic` 标量。

**KafkaConsumerConf 与 ServiceConf 注入**:

`KafkaConsumerConf` 不内嵌 `service.ServiceConf`，`Name`/`Log`/`Mode` 等由 RPC 服务配置注入：

```go
// common/configx/kqConfig.go
func (c KafkaConsumerConf) ToKqConf(svcConf service.ServiceConf) kq.KqConf {
    return kq.KqConf{
        ServiceConf: svcConf,
        Brokers:     c.Brokers,
        Group:       c.Group,
        Topic:       c.Topic,
        // ... 其余字段直接拷贝
    }
}
```

服务启动时注入：
```go
// bridgekafka.go — 多 consumer
for i := range c.KafkaConsumeConfig {
    kc := c.KafkaConsumeConfig[i]
    if len(kc.Brokers) == 0 || kc.Topic == "" {
        continue
    }
    fullConf := kc.ToKqConf(c.ServiceConf)
    serviceGroup.Add(kq.MustNewQueue(fullConf, handler.NewKafkaStreamHandler(...)))
}

// iecstash.go — 单 consumer
if kc.Brokers != nil && kc.Topic != "" {
    fullConf := c.KafkaASDUConfig.ToKqConf(c.ServiceConf)
    serviceGroup.Add(kq.MustNewQueue(fullConf, kafka.NewAsdu(ctx)))
}
```

**Producer 配置**（bridgekafka 示例）:
```yaml
KafkaPushConfig:
  Brokers:
    - 127.0.0.1:9094
  Topics:
    - asdu
    - alarm
    - event
```

**Consumer 配置**（bridgekafka 多 consumer）:
```yaml
KafkaConsumeConfig:
  - Brokers:
      - 127.0.0.1:9094
    Topic: asdu
    Group: bridge-kafka-asdu
    Conns: 3
    Consumers: 3
    Processors: 18
```

**Consumer 配置**（iecstash 单 consumer）:
```yaml
KafkaASDUConfig:
  Brokers:
    - 127.0.0.1:9094
  Topic: asdu
  Group: iec-stash
```

**ServiceContext 中创建 Pusher map**（bridgekafka）:
```go
pushers := make(map[string]*kq.Pusher)
for _, topic := range c.KafkaPushConfig.Topics {
    pushers[topic] = kq.NewPusher(c.KafkaPushConfig.Brokers, topic, kq.WithSyncPush())
}
```

**Logic 层按 topic 路由**:
```go
func (l *PublishLogic) Publish(in *PublishReq) (*PublishRes, error) {
    pusher, ok := l.svcCtx.Pushers[in.Topic]
    if !ok {
        return nil, fmt.Errorf("kafka topic %s not configured", in.Topic)
    }
    if in.Key != "" {
        return nil, pusher.PushWithKey(l.ctx, in.Key, string(in.Value))
    }
    return nil, pusher.Push(l.ctx, string(in.Value))
}
```

**KafkaMessage 转发契约**（streamevent.proto）:
```protobuf
message KafkaMessage {
  string sessionId = 1;  // 必需：会话标识
  string msgId     = 2;  // 必需：消息唯一 ID
  string topic     = 3;  // 必需：Kafka topic
  string group     = 4;  // 必需：消费者组
  string key       = 5;  // 可选：消息 key
  bytes  value     = 6;  // 必需：消息体
  string sendTime  = 7;  // 必需：发送时间
}
```
- handler 转发时必须填充全部 7 个字段。
- `SessionId` 可用 topic 名作为标识。
- `MsgId` 使用 `tool.SimpleUUID()` 生成。

### 4. Validation & Error Matrix
- gRPC topic 不在 `KafkaPushConfig.Topics` 中 -> 返回 `kafka topic %s not configured` 错误
- `KafkaPushConfig.Brokers` 为空 -> 不创建任何 Pusher（无报错，等待下次配置）
- `KafkaConsumeConfig[i].Brokers` 为空或 `Topic` 为空 -> 跳过该 consumer（continue）
- `KafkaASDUConfig`（单 consumer）Brokers 为空或 Topic 为空 -> 不启动消费者
- handler 中 `streamEventCli` 为 nil -> 跳过 gRPC 转发（不报错）
- KafkaMessage 遗漏 `Group` 或 `SessionId` -> 下游 streamevent 无法识别消息来源（可追溯性下降）
- `KafkaConsumerConf` 的 `ServiceConf` 未注入 -> `Name` 字段为空，go-zero 运行可能异常

### 5. Good/Base/Bad Cases
- Good: 配置 3-5 个核心 topic，每个创建独立 Pusher。gRPC Publish 按 topic 路由到对应 Pusher。consumer side 完整填充 KafkaMessage 7 字段。
- Base: 配置 1 个 topic，功能可用但灵活性受限。新增 topic 需修改配置重启。
- Bad: 尝试在运行时动态创建 Pusher（无并发控制）、遗漏 KafkaMessage 的 Group/SessionId 字段、bridge 模块中引入 socket 推送（mqtt 桥接已覆盖该需求）。

### 6. Tests Required
- Unit: `PublishLogic.Publish` 传入未配置 topic -> 断言返回错误。
- Unit: `PublishLogic.Publish` 传入空 key -> 断言调用 `Push` 而非 `PushWithKey`。
- Unit: `PublishLogic.Publish` 传入非空 key -> 断言调用 `PushWithKey`。
- Integration: 启动 bridgekafka 服务，向已配置 topic 发送消息，验证 Kafka 中有消息。

### 7. Wrong vs Correct

#### Wrong
```go
// Wrong: 忽略 go-queue Pusher topic 固定特性，试图动态传 topic
err := pusher.PushWithKey(ctx, key, value) // topic 在 NewPusher 时已固定
```

```go
// Wrong: 转发 KafkaMessage 时遗漏字段
&streamevent.KafkaMessage{
    Topic: h.topic,
    Value: []byte(value),
    // 遗漏 SessionId、MsgId、Group
}
```

#### Correct
```go
// Correct: 用 Pusher map 实现多 topic 支持
pusher, ok := l.svcCtx.Pushers[in.Topic]
if !ok {
    return nil, fmt.Errorf("kafka topic %s not configured", in.Topic)
}
if in.Key != "" {
    err = pusher.PushWithKey(l.ctx, in.Key, string(in.Value))
} else {
    err = pusher.Push(l.ctx, string(in.Value))
}
```

```go
// Correct: 完整填充 KafkaMessage 所有字段
&streamevent.KafkaMessage{
    SessionId: h.topic,
    MsgId:     msgId,
    Topic:     h.topic,
    Group:     h.group,
    Key:       key,
    Value:     []byte(value),
    SendTime:  sendTime,
}
```

### Design Decision: 单一 Publish RPC，不单独暴露 trace 方法

**Context**: bridgemqtt 有 `Publish` 和 `PublishWithTrace` 两个 RPC。bridgekafka 只保留 `Publish`。

**Decision**: go-queue `Pusher.PushWithKey` 内部已通过 OpenTelemetry 自动向 Kafka headers 注入 `traceparent`，无需调用方额外传 traceId。因此 bridgekafka proto 只需一个 `Publish` RPC，key 有值走 `PushWithKey`（带 trace header），无值走 `Push`（自动生成时间戳 key，同样带 trace header）。

### Design Decision: bridgekafka 不做 socket 转发

**Context**: bridgemqtt 同时转发到 streamevent 和 socketpush。bridgekafka 最初设计也包含 socket 转发，但讨论后移除。

**Decision**: socket 推送由 bridgemqtt 覆盖，bridgekafka 保持轻量，只做 streamevent gRPC 转发。新增 bridge 模块时默认不包含 socket 转发，除非有明确需求。

### Design Decision: Kafka 配置类型集中在 common/configx

**Context**: 多个服务（bridgekafka、iecstash、xfusionmock）各自定义 Kafka 配置结构体，字段重复且命名不一致。

**Options Considered**:
1. 每个服务自定义 Kafka 配置类型
2. 统一放在 `common/configx/`，所有服务引用

**Decision**: 选择 Option 2。三种类型覆盖全部场景：`KafkaPushConf`（单 topic push）、`KafkaMultiPushConf`（多 topic push）、`KafkaConsumerConf`（消费，ServiceConf 注入）。

**Naming Convention**: 所有类型使用 `Kafka` 前缀 + `Conf` 后缀，与 go-zero 的 `RpcServerConf`/`ServiceConf` 命名风格一致。

**Example**:
```go
// bridgekafka config
type Config struct {
    zrpc.RpcServerConf
    KafkaPushConfig    configx.KafkaMultiPushConf      `json:",optional"`
    KafkaConsumeConfig []configx.KafkaConsumerConf     `json:",optional"`
}

// iecstash config
type Config struct {
    zrpc.RpcServerConf
    KafkaASDUConfig configx.KafkaConsumerConf
}

// xfusionmock config
type Config struct {
    zrpc.RpcServerConf
    KafkaPointConfig  configx.KafkaPushConf
    KafkaAlarmConfig  configx.KafkaPushConf
}
```

**Anti-pattern**: 在服务 `internal/config/` 中定义私有 Kafka 配置结构体。除非该配置包含服务独有的业务字段，否则应使用 `configx` 类型。

## Scenario: MQTT request/reply routing with mqttx

### 1. Scope / Trigger
- Trigger: 修改 `common/mqttx` 的 reply/request 路由、MQTT handler 注册、订阅恢复、topic 命名或 `ReplyRouter` API 时必须读取本节。
- Applies to: `common/mqttx.Client`、`handlerManager`、`messageDispatcher`、`ReplyRouter[T]`、`ReplyDecoder[T]`、调用方协议包（如 DJI SDK 适配层）。
- Why: MQTT reply 抽象要复用 `antsx.ReplyPool` 的 request/reply `tid` 语义，同时避免把 DJI payload、设备 ID、topic builder、业务错误码泄漏进公共 `mqttx`。

### 2. Signatures
- Handler: `Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error`
- Handler registration: `func (c Client) AddHandler(topicTemplate string, handler ConsumeHandler) error` / `func (c Client) AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error`（`Client` 是接口，`mqttClient` 是未导出实现）
- Subscription is automatic: `AddHandler` with `AutoSubscribe=true` (default) and `onConnect -> restoreSubscriptions` cover all subscription paths. There is no public `Subscribe` method.
- Reply registration: `func WithReplyRouter[T any](topicTemplate string, router *ReplyRouter[T]) ClientOption`
- Reply message: `type ReplyMessage[T any] struct { Tid string; Value T }`
- Reply decoder: `type ReplyDecoder[T any] interface { Decode(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[T], error) }`
- Function adapter: `type ReplyDecoderFunc[T any] func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[T], error)`
- Typed request/reply (public): `func RequestReply[T any](ctx context.Context, c Client, topicTemplate string, tid string, send func() error, ttl ...time.Duration) (T, error)`
- Typed request/reply (router): `func (r *ReplyRouter[T]) RequestReply(ctx context.Context, tid string, send func() error, ttl ...time.Duration) (T, error)`
- Reply handler lookup: `type replyHandlerGetter interface { getReplyHandler(topicTemplate string) ConsumeHandler }`（`mqttx.RequestReply` 内部断言，公共 `Client` 接口不包含私有方法）
- Client constructor: `func NewClient(cfg MqttConfig, opts ...ClientOption) (Client, error)` / `func MustNewClient(cfg MqttConfig, opts ...ClientOption) Client`

### 3. Contracts
- `topic` in handler/decoder signatures is the actual MQTT message topic from `msg.Topic()`.
- `topicTemplate` is the subscription topic/filter whose callback fired; it may contain MQTT `+` or `#` wildcards.
- Dispatcher uses `topicTemplate` as an exact lookup key. It must not scan all registered templates or run a second wildcard matcher; MQTT subscription matching belongs to the MQTT client.
- `WithReplyRouter` registers reply routing metadata only. MQTT subscription is restored through `onConnect -> restoreSubscriptions -> getAllTopicTemplates`.
- `getAllTopicTemplates()` returns de-duplicated subscription templates from regular handlers and reply routers. Ordering is not a contract.
- Same `topicTemplate` regular handlers run in registration order.
- For one message, reply router runs before regular handlers on the same `topicTemplate`; `ErrReplyNotMatched` is not logged as an error and does not block regular handlers.
- `tid` is the canonical unique request/reply message ID, aligned with `antsx.ReplyPool` logs. It is not DJI-specific business meaning.
- `RequestReply[T]` is a package-level generic function; it asserts the client implements private `replyHandlerGetter`, looks up the registered `*ReplyRouter[T]`, and calls `router.RequestReply` internally.
- Protocol packages own payload schema, topic builders, device identifiers, business result codes, and domain errors.
- Protocol SDKs that use `RequestReply[T]` must register reply routers at `mqttx.NewClient` / `MustNewClient` construction time with `WithReplyRouter`. Do not register request/reply topics through `AddHandler` as ordinary consumers.
- DJI request/reply topics use `djisdk.MustNewClient(cfg, djisdk.WithPendingTTL(ttl))` (production) or `djisdk.NewClient(mqttClient, djisdk.WithPendingTTL(ttl))` (tests/shared connection); both internally register `WithReplyRouter` for `services_reply` and `property/set_reply`. Replies decode to `*djisdk.ServiceReply` inside `common/djisdk`, then `SendCommand` and `SetProperty` call `mqttx.RequestReply[*ServiceReply]` by topic template.

### 4. Validation & Error Matrix
- `ReplyRouter` constructed with nil decoder + reply handled -> `ErrNilDecoder`.
- Decoder returns empty `ReplyMessage.Tid` -> `ErrEmptyReplyTid` (`ErrEmptyReplyID` may exist only as compatibility alias).
- Decoder returns error -> propagate decoder error from `HandleReply`/`Consume`.
- Decoded `tid` has pending entry -> `HandleReply` returns `(true, nil)` and resolves waiting `RequestReply`.
- Decoded `tid` has no pending entry -> `Consume` returns `ErrReplyNotMatched`; dispatcher suppresses it and continues regular handlers.
- `RequestReply` send function returns error -> pending entry is rejected/cleaned by `antsx.RequestReply`, and the send error is returned.
- `RequestReply[T]` called with nil client or a client implementation that does not provide the internal reply-handler lookup -> zero `T` plus `ErrNoReplyRouter`.
- `RequestReply[T]` called before the topic template has a registered reply router -> zero `T` plus `ErrNoReplyRouter`; the publish `send` function is not executed.
- `RequestReply[T]` type parameter does not match the registered `ReplyRouter[T]` -> zero `T` plus `ErrReplyType`.
- `ReplyRouter.Close()` while requests are pending -> pending `RequestReply` calls return `antsx.ErrReplyClosed`.
- No reply router and no regular handler for `topicTemplate` -> dispatcher calls `onNoHandler`.
- All errors are consolidated in `common/mqttx/errors.go`.

### 5. Good/Base/Bad Cases
- Good: Protocol package creates a typed `ReplyDecoder`, registers it with `WithReplyRouter`, then calls `mqttx.RequestReply[T](ctx, client, topicTemplate, tid, send)` and receives `T` directly. Protocol-specific error conversion stays outside `mqttx`.
- Good: DJI service wiring calls `djisdk.MustNewClient(cfg, djisdk.WithPendingTTL(ttl))` which internally creates `mqttx.Client` with DJI reply routers.
- Good: DJI test code calls `djisdk.NewClient(nil, djisdk.WithPendingTTL(ttl))` to create a Client without MQTT for handler testing.
- Base: A notification-only consumer uses `AddHandler(topicTemplate, handler)` and receives actual `topic` plus callback `topicTemplate`.
- Bad: Registering request/reply topics such as DJI `services_reply` or `property/set_reply` with `AddHandlerFunc`, exposing `ReplyRouter.do` publicly, calling `ReplyRouter.do` directly from outside the package, or running custom wildcard matching inside dispatcher.

### 6. Tests Required
- Unit: `ReplyDecoderFunc.Decode` returns `ReplyMessage{Tid, Value}` and preserves `topic/topicTemplate` args.
- Unit: `ReplyRouter.HandleReply` resolves a pending `tid` and returns matched status.
- Unit: nil decoder, decoder error, empty `Tid`, unmatched `tid`, send failure cleanup, reject, and close-pending behavior.
- Unit: `WithReplyRouter` adds reply topic template to `getAllTopicTemplates()`.
- Unit: `RequestReply[T]` resolves a pending `tid` through the registered router and returns typed `T`.
- Unit: `RequestReply[T]` returns `ErrNoReplyRouter` when no router is registered for the topic template.
- Unit: `RequestReply[T]` returns `ErrReplyType` when caller `T` does not match registered router `T`.
- Unit: protocol SDK request paths assert the expected reply topic template and TTL are passed to `RequestReply[T]`.
- Unit: protocol SDK subscription helpers assert request/reply topics are not registered as ordinary handlers.
- Unit: dispatcher calls reply router before regular handlers for the same `topicTemplate`, and still calls regular handlers after `ErrReplyNotMatched`.
- Unit: same `topicTemplate` regular handlers run in registration order.
- Unit: `getAllTopicTemplates()` includes regular and reply templates once; do not assert map iteration order.
- Search assertion: after changing public reply API, search `WithReplyRouter|NewReplyRouter|RequestReply|ReplyMessage\[` across repo and update all callers.

### 7. Wrong vs Correct

#### Wrong
```go
// Wrong: public reply registration accepts any ConsumeHandler, so ordinary handlers
// can accidentally enter the reply path.
func WithReplyHandler(topicTemplate string, h ConsumeHandler) ClientOption { /* ... */ }
```

```go
// Wrong: dispatcher re-runs wildcard matching across all templates.
// MQTT client already matched the subscription and invoked the callback for topicTemplate.
for topicTemplate, handlers := range m.handlers {
    if topicFilterMatches(topicTemplate, msg.Topic()) {
        dispatch(handlers)
    }
}
```

```go
// Wrong: common/mqttx reply message leaks protocol-specific fields.
type ReplyMessage[T any] struct {
    DeviceID string
    Method   string
    Result   int
    Value    T
}
```

```go
// Wrong: calling ReplyRouter.do directly from outside the package.
// do is private; use mqttx.RequestReply[T] instead.
ack, err := router.do(ctx, tid, send)
```

#### Correct
```go
router := mqttx.NewReplyRouter[string](mqttx.ReplyDecoderFunc[string](
    func(ctx context.Context, payload []byte, topic string, topicTemplate string) (mqttx.ReplyMessage[string], error) {
        // Protocol package owns payload parsing and business error conversion.
        tid, value, err := decodeProtocolReply(payload)
        if err != nil {
            return mqttx.ReplyMessage[string]{}, err
        }
        return mqttx.ReplyMessage[string]{Tid: tid, Value: value}, nil
    },
))

client, err := mqttx.NewClient(cfg, mqttx.WithReplyRouter("thing/+/reply", router))
```

```go
// Correct: DJI registers both reply routers while constructing mqttx.Client.
djiClient := djisdk.MustNewClient(cfg, djisdk.WithPendingTTL(pendingTTL))
```

```go
// Correct: test code uses NewClient with nil/fake mqttx.Client.
djiClient := djisdk.NewClient(nil, djisdk.WithPendingTTL(pendingTTL))
// Or with a recording mock:
djiClient := djisdk.NewClient(&recordingMQTTClient{}, djisdk.WithPendingTTL(pendingTTL))
```

```go
// Correct: use mqttx.RequestReply[T] as the sole public request/reply entry point.
ack, err := mqttx.RequestReply[*ProtocolAckType](ctx, client, "thing/+/reply", tid, func() error {
    return publishRequest(ctx, client, tid)
})
if err != nil {
    return err
}
```

```go
// Correct: dispatcher uses the callback topicTemplate as the exact routing key.
replyHandler := manager.getReplyHandler(topicTemplate)
handlers := manager.getHandlers(topicTemplate)
```

## Scenario: Device heartbeat manager with active cloud heartbeat

### 1. Scope / Trigger
- Trigger: 新增或修改“设备上报心跳续期 + 云端主动下发心跳/控制报文”的运行时 manager 时必须读取本节。
- Applies to: 服务内 `internal/**` 的设备会话管理器、DRC/遥控/长连接状态机、MQTT/SDK 心跳 handler、后台 clean loop。
- Why: 这类模块通常同时持有 TTL cache 和 goroutine worker 两份状态，若删除路径不统一，会出现旧 worker 删除新 worker、cache 与 worker 不一致、context timer 泄漏或过期通知重复。

### 2. Signatures
- Manager close: `func (m *Manager) Close()` 必须取消 manager 级父 context。
- Enable/start: `func (m *Manager) Enable(ctx context.Context, deviceID string, opts ...Option) error` 或等价入口必须创建/刷新 state，并启动单设备 worker。
- Disable/stop: `func (m *Manager) Disable(ctx context.Context, deviceID string) error` 或等价入口必须同时删除 TTL cache 和停止当前 worker。
- Device heartbeat: `func (m *Manager) OnDeviceHeartbeat(ctx context.Context, deviceID string)` 或等价 handler 必须刷新 `LastHeartbeat` 并 `cache.Set` 续期。
- Worker state helper: `func (s *State) isAlive(now time.Time, timeout time.Duration) bool`、`func (s *State) isCurrentSessionAlive(sessionID string, now time.Time, timeout time.Duration) bool` 这类 helper 调用方必须已持有 `state.mu`。

### 3. Contracts
- TTL cache key 使用设备 ID，value 使用单设备 `*State`；`cache.Set(deviceID, state)` 同时表示写入和续期。
- 每个 worker 必须携带不可复用的 session identity，例如 `sessionID`，不能只用设备 ID 表示 worker 身份。
- Manager 父 context 只负责全局关闭；单设备 cancel 只负责 disable、re-enable、deadline/TTL cleanup。
- `Close()` 不遍历逐个 worker cancel；所有 worker context 必须从 manager 父 context 派生，父 cancel 自动传播。
- worker goroutine 退出时必须释放自己的 child context，并只删除自己对应的 worker；重复调用 `CancelFunc` 是安全的。
- cache 删除和 worker 删除必须成对审查：主动 disable、绝对 deadline、TTL cleanup、cache miss、自退、re-enable 都要明确清理哪一份状态。
- 已经拿到 `*State` 指针的并发读者不受 `cache.Del` 影响；删除 cache 前后仍应在同一 session state 上写入 `Enabled=false`，让旧指针读者看到停用状态。

### 4. Validation & Error Matrix
- Enable on alive state -> 返回成功，不重复启动 worker。
- Enable on stale/expired state -> 先停止旧 worker，再写入新 state 和新 session worker。
- Disable on enabled state -> `state.Enabled=false`、`cache.Del(deviceID)`、停止当前 worker、触发 disabled hook。
- Device heartbeat cache miss -> 只记录/忽略，不重建 state，不通知 expired hook。
- Device heartbeat max deadline exceeded -> `state.Enabled=false`、`cache.Del`、停止当前 worker、触发 expired hook。
- Worker tick sees cache miss/stale session/expired -> worker 自退，并只删除自己。
- Clean loop sees cache miss or same-session expired/disabled -> identity-safe 删除 worker；cache miss/expired 触发 expired hook，disabled 不触发 expired hook。
- Clean loop sees stale worker while cache has newer session -> 只 cancel 旧 worker，不删除 cache，不 cancel 新 worker。

### 5. Good/Base/Bad Cases
- Good: `workers` map value 是 `*worker{sessionID, cancel}`，删除用 `CompareAndDelete(deviceID, worker)`；cache TTL 用设备心跳 `Set` 续期，worker cleanup 只按自身 identity 删除。
- Base: 没有 clean loop，依赖 worker 下一次 tick 自退；只适合 heartbeat interval 很短且可接受短时间 worker 残留的场景，并需在代码注释写明。
- Bad: `workers` map 只存 `context.CancelFunc`，worker 退出时直接 `Delete(deviceID)`；旧 goroutine 可能删除新会话 worker。
- Bad: `Close()` 先取消父 context 后又遍历所有 worker cancel，把父子 context 传播和单 worker 停止职责混在一起。

### 6. Tests Required
- Unit: stale worker cleanup must not delete or cancel current worker for the same device ID.
- Unit: cache-miss cleanup cancels worker and fires expired hook with the old session ID.
- Unit: same-session expired cleanup deletes cache, deletes worker, cancels worker, and fires expired hook.
- Unit: same-session disabled cleanup deletes cache and worker but does not fire expired hook.
- Unit: manager close relies on parent context; no per-worker cancel iteration is required.
- Unit: `State` alive helpers are called only while holding `state.mu`; tests should cover alive, expired, disabled, and session mismatch cases.
- Search assertion: when changing this pattern, search `cache.Del|cache.Set|workers.Store|LoadAndDelete|CompareAndDelete|context.WithDeadline|context.WithCancel` in the target package and inspect all lifecycle paths.

### 7. Wrong vs Correct

#### Wrong
```go
// Wrong: value has no worker identity; old goroutine can delete new worker.
workers.Store(deviceID, cancel)
defer workers.Delete(deviceID)
```

```go
// Wrong: cache.Del does not invalidate already-held *State pointers.
cache.Del(deviceID)
// state.Enabled remains true for goroutines that fetched state before Del.
```

#### Correct
```go
type heartbeatWorker struct {
    sessionID string
    cancel    context.CancelFunc
}

worker := &heartbeatWorker{sessionID: sessionID, cancel: cancel}
workers.Store(deviceID, worker)
defer workers.CompareAndDelete(deviceID, worker)
defer worker.cancel() // release child context resources on self-exit
```

```go
state.mu.Lock()
state.Enabled = false
state.mu.Unlock()
cache.Del(deviceID)
```

### Design Decision: 包级泛型函数 RequestReply[T]，不暴露 ReplyRouter.RequestReply

**Context**: Go 方法不支持类型参数。`Client` 是接口（非泛型），无法写 `Client.RequestReply[T any](...)` 方法。但 `ReplyRouter[T].RequestReply` 返回 `T`，调用方需要类型安全的请求/回复入口。

**Options Considered**:
1. `Client.RequestReply` 返回 `any` + 私有 `replyCaller` 接口做类型擦除 — 调用方需要 `any` 类型断言
2. 包级泛型函数 `mqttx.RequestReply[T](ctx, client, topicTemplate, tid, send)` — 调用方拿到 `T`，无类型断言
3. 泛型 wrapper 接口 `RequestReplyer[T]` + `RequestReplyClient[T]` — 嵌套复杂
4. 暴露 `ReplyRouter.RequestReply` 为公有方法 — 泄漏内部实现，调用方需要额外保存 router 引用

**Decision**: 选择 Option 2。`ReplyRouter.RequestReply` 包内可见；`Client` 接口含私有 `getReplyHandler` 方法，保持非泛型并继续负责 MQTT 连接、订阅、dispatch 和 router 注册；`mqttx.RequestReply[T]` 是唯一公开 request/reply 入口，返回 `T`，调用方不做 `any` 类型断言。

**Type Erasure Pattern**:
```go
// Client 接口只暴露公共 MQTT 能力，外部包可实现测试 fake。
type Client interface {
    AddHandler(topicTemplate string, handler ConsumeHandler) error
    // ... 其他公有方法
}

type replyHandlerGetter interface {
    getReplyHandler(topicTemplate string) ConsumeHandler // 私有
}

// mqttClient 是未导出实现
type mqttClient struct { ... }
func (c *mqttClient) getReplyHandler(topicTemplate string) ConsumeHandler { ... }

// 包级泛型函数，通过内部接口断言访问路由
func RequestReply[T any](ctx context.Context, c Client, topicTemplate string, tid string, send func() error, ttl ...time.Duration) (T, error) {
    var zero T
    if c == nil {
        return zero, ErrNoReplyRouter
    }
    getter, ok := c.(replyHandlerGetter)
    if !ok {
        return zero, ErrNoReplyRouter
    }
    handler := c.getReplyHandler(topicTemplate)
    if handler == nil {
        return zero, ErrNoReplyRouter
    }
    router, ok := handler.(*ReplyRouter[T])
    if !ok {
        return zero, ErrReplyType
    }
    return router.RequestReply(ctx, tid, send, ttl...)
}
```

**Caller Side**:
```go
ack, err := mqttx.RequestReply[*types.BroadcastAckBody](ctx, svc.MqttClient, svc.broadcastAckTopic, tId, sendFunc)
```

### Design Decision: mqttx 包级错误集中在 errors.go

**Context**: 错误变量散落在 `reply_router.go` 和 `dispatcher.go`，不便查找和维护。

**Decision**: 所有包级错误统一放在 `common/mqttx/errors.go`，每个 error 附注释说明触发条件。`ErrEmptyReplyID` 保留为兼容别名。

**Example**:
```go
// errors.go
var (
    ErrNilDecoder    = errors.New("mqttx: reply decoder cannot be nil")
    ErrEmptyReplyTid = errors.New("mqttx: reply message tid cannot be empty")
    ErrNoReplyRouter = errors.New("mqttx: reply router not registered")
    ErrReplyType     = errors.New("mqttx: reply router type mismatch")
    ErrReplyNotMatched = errors.New("mqttx: reply not matched")
    ErrEmptyReplyID  = ErrEmptyReplyTid // compatibility alias
)
```

### Design Decision: ClientOptions 标准选项结构体模式

**Context**: 旧 `Option func(*mqttClient)` 直接操作未导出结构体，不透明且无法扩展。

**Decision**: 改用标准 Go 选项模式，定义 `ClientOptions` 结构体 + `ClientOption func(*ClientOptions)` 函数类型。每个 `With*` 函数返回 `ClientOption`，构造函数内部先解析选项再构建 `mqttClient`。

**Pattern**:
```go
type ClientOptions struct {
    onReady      func(Client)
    replyRouters []replyRouterRegistration
}

type ClientOption func(*ClientOptions)

func NewClient(cfg MqttConfig, opts ...ClientOption) (Client, error) {
    o := &ClientOptions{}
    for _, opt := range opts {
        if opt != nil { opt(o) }
    }
    c := &mqttClient{onReady: o.onReady}
    for _, reg := range o.replyRouters {
        c.handlerMgr.addReplyHandler(reg.topicTemplate, reg.handler)
    }
    // ...
}
```

### Convention: onReady 回调只触发首次连接

`WithOnReady(fn func(Client))` 注册的回调只在 MQTT 首次连接成功时执行一次。断线重连时不会再次触发，因为 handler 注册是持久化的（`handlerMgr` 保留在内存中）。重连后 `restoreSubscriptions` 负责重新订阅 topic。

```go
// 实现方式：ready bool 标志位
if !c.ready && c.onReady != nil {
    c.onReady(c)
    c.ready = true
}
```

不要用 `sync.Once` 代替 `ready bool`，因为并发场景简单且标志位更直观。
