# mqttx

MQTT 客户端封装（HTTP 风格 API + 泛型请求-应答 + OTel 链路追踪）。

- 底层：`eclipse/paho.mqtt.golang`（MQTT 3.1.1）
- 能力：主题订阅与重连恢复、消息分发、协议中立的 `RequestReply`、TLS、按 topic 的日志限频、trace 自动注入与续接

## 快速开始

```go
cli := mqttx.MustNewClient(mqttx.MqttConfig{
	Broker: []string{"tcp://127.0.0.1:1883"},
	Qos:    1,
	Timeout: 2000, // 毫秒，连接与发布等待上限
})

// 消费：模板可含 + / #，handler 拿到的是原始 payload
if err := cli.AddHandler("device/+/data", mqttx.ConsumeHandlerFunc(
	func(ctx context.Context, payload []byte, topic, topicTemplate string) error {
		logx.WithContext(ctx).Infof("topic=%s payload=%s", topic, payload)
		return nil
	},
)); err != nil {
	return err
}

// 生产：ctx 带 span 时自动注入 trace 头
if err := cli.Publish(ctx, "device/1/data", payload); err != nil {
	return err
}
```

> 连接成功后注册的 handler 会立即订阅；重连后由客户端自动恢复订阅，业务侧不需要重新注册。

## 消息格式

MQTT 3.1.1 **没有自定义 header 位**，因此 trace 头只能落在 payload 里。约定如下：

| payload 形态 | 发送侧行为 | 线上字节 |
| --- | --- | --- |
| JSON 对象，未带 `traceparent` | 末尾追加顶层 `headers` | 原字节 + `,"headers":{...}` |
| JSON 对象，已带 `traceparent`（如 ASDU） | 不注入 | 原字节，一个字节不改 |
| JSON 对象，已占用顶层 `headers`（非 trace 头） | 不注入、不覆盖 | 原字节，一个字节不改 |
| 非 JSON 对象（二进制/数组/标量）或非法 JSON | 不注入 | 原字节 |
| ctx 无有效 span | 不注入 | 原字节 |

注入后的线上格式（业务字段位置不变，只在末尾追加）：

```json
{
  "a": 1,
  "msg": "hi",
  "headers": {
    "traceparent": "00-4bf92f3577b34da6a3ce929d0e6e4736-00f067aa0ba902b7-01"
  }
}
```

消费侧拿到的永远是**原始 payload 字节**（不会再去掉或改写 `headers`），需要业务字段时直接 `json.Unmarshal`；trace 头由客户端内部解析，业务不用关心。

## 链路追踪

### 发送：`Publish` 自动注入

```go
// ctx 里有有效 span → 注入；没有 → 原样发送。调用方不需要区分
err := cli.Publish(ctx, topic, payload)
```

规则按成本递增依次判断，命中即返回：

1. `ctx == nil` 或 `SpanContextFromContext(ctx).IsValid()` 为 false → 不注入；
2. payload 首尾不是 `{` `}` → 不注入（二进制、数组、标量、截断）；
3. payload 里已出现 `"traceparent"` 字面量 → 不注入（高频 ASDU 的零分配快速路径）；
4. `json.Unmarshal` 探测失败（非法 JSON）或顶层已有 `headers` → 不注入；
5. 其余情况：把 `traceparent`（及可选的 `tracestate`）追加到顶层 `headers`。

性能参考（本机 i7-9750H，600B ASDU 报文）：

| 路径 | 耗时 | 分配 |
| --- | --- | --- |
| ctx 无 span | ~15 ns | 0 |
| 已带 traceparent（ASDU 热路径） | ~250 ns | 0 |
| 需要注入的普通 JSON | ~3 µs | 9 次 |

### 接收：自动续接

```text
payload → 解析顶层 headers.traceparent → Extract 到 ctx
        → 起 mqtt-consume span（ClientID / topic / topic_template / message_id / qos）
        → 分发到 handler（payload 原样传入）
```

- 无 trace 头时不报错，`mqtt-consume` 自己成为根 span；
- 采样位为 0 的有效 `traceparent` 照常续接，只是下游不记录 span；
- `headers` 类型不是字符串映射、`traceparent` 格式非法时静默忽略，业务消息照常分发。

### 与 Kafka 的一致性

| 维度 | Kafka（`go-queue`） | MQTT（mqttx） |
| --- | --- | --- |
| 注入位置 | record header（`msg.Headers`） | payload 顶层 `headers` 字段 |
| 触发方式 | `PushWithKey` 内部按 ctx `Inject` | `Publish` 内部按 ctx 注入 |
| 业务字节 | 不变 | 未带 trace 时末尾追加 `headers`，已带则不变 |
| 消费端 | `kq` 自动 `Extract` 到 ctx | `processMessage` 自动 `Extract` |

Kafka 的 record header 只在单次 broker 投递内有效；`bridge` 类服务转发时通常只透传 payload，所以**跨跳传播以消息体内 `headers` 为准**。

## 请求-应答

`ReplyRouter` 提供协议中立的 `tid` 关联，不内置 topic 与业务 schema：

```go
router := mqttx.NewReplyRouter(
	mqttx.ReplyDecoderFunc[string](func(_ context.Context, payload []byte, _, _ string) (mqttx.ReplyMessage[string], error) {
		var msg struct {
			Tid   string `json:"tid"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			return mqttx.ReplyMessage[string]{}, err
		}
		return mqttx.ReplyMessage[string]{Tid: msg.Tid, Value: msg.Value}, nil
	}),
	mqttx.WithReplyRouterTTL(10*time.Second),
)

res, err := router.RequestReply(ctx, tid, func() error {
	return cli.Publish(ctx, requestTopic, reqBody)
})
```

注册方式二选一：`NewClient(..., mqttx.WithReplyRouter("reply/+", router))`，或客户端已存在时 `cli.AddReplyRouter("reply/+", router)`。

## 配置

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `Broker` | 必填 | 如 `tcp://host:1883`、`ssl://host:8883` |
| `ClientID` | 自动生成 | 空值时用 UUID 去横线 |
| `Qos` | 1 | 大于 2 时回退为 1 |
| `Timeout` | 30000ms | 连接与发布等待上限 |
| `KeepAlive` | 60000ms | 心跳间隔 |
| `SubscribeTopics` | 空 | 初始化即订阅的主题 |
| `TLS.CAFile` | 空 | 自签 CA，配置后替换系统根证书池 |
| `TLS.CertFile` / `TLS.KeyFile` | 空 | 双向认证，必须成对配置 |
| `TLS.ServerName` | 空 | 域名与连接地址不一致时手工指定 |
| `TLS.InsecureSkipVerify` | false | 仅限排障临时开启 |

TLS 配置错误直接返回 error，不会降级为明文连接。

## 变更说明

- 移除了 `Message{topic,payload,headers}` 包装结构与 `PublishWithTrace` 接口：现在统一走 `Publish`，trace 自动注入，消费端不再需要“先判断是不是包装再解包”。
- 移除后**不兼容**旧的包装报文：仍在发 `{"topic":...,"payload":"<base64>"}` 的客户端需要同步升级。

## 验证

```bash
go build ./common/mqttx/...
go test ./common/mqttx/... -count=1
go test -race ./common/mqttx/... -count=1
go test ./common/mqttx/ -run '^$' -bench 'BenchmarkInjectTraceHeaders' -benchmem
```
