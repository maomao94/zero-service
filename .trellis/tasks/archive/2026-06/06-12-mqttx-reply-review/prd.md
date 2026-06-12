# 审查并完善 mqttx reply 能力

## Goal

对 `common/mqttx` 做一次集中代码审查与修复，形成协议中立、Go 风格、可测试的 MQTT request/reply 路由能力。

调用方应能通过 `ReplyRouter[T]` 复用 reply topic 路由与 `tid` 匹配能力，同时继续由业务协议包负责 payload schema、topic builder、设备 ID、业务错误码和响应结果转换。

## Confirmed Facts

- `common/mqttx` 已有 `ReplyRouter[T]`、`ReplyMessage[T]`、`ReplyDecoder[T]`、`WithReplyHandler`、dispatcher 和 client 生命周期代码。
- `ReplyRouter[T]` 依赖 `common/antsx.ReplyPool[T]` 做 pending request 注册、超时、resolve/reject、关闭清理。
- `antsx.ReplyPool` 使用 `tid` 表示 request/reply 唯一消息 ID；`mqttx` 应同步该命名，不额外发明 `reply_id` 或 `correlation_id`。
- 当前 `WithReplyHandler(topic string, h ConsumeHandler)` 类型过宽，普通 handler 可以误注册到 reply 路径。
- 当前 `ReplyDecoder[T]` 是函数类型，不符合 Go 常见编解码接口习惯。
- dispatcher 应按订阅回调传入的 `topicTemplate` 精确查找 handler；MQTT client 已负责 wildcard 订阅命中。
- topic map 的迭代顺序不需要稳定；只要求同一个 topic 下普通 handler 的执行顺序稳定。
- `common/mqttx` 当前只有 `config_test.go`，reply router 与 dispatcher reply 行为测试不足。

## Requirements

- `mqttx` 只抽象 MQTT reply topic routing 与 `tid` 匹配，不引入 DJI 专用字段、payload schema、设备 ID、`method/result` 或业务错误转换。
- reply 唯一消息 ID 统一命名为 `tid`，保持与 `antsx` 一致；这里的 `tid` 是 request/reply 唯一消息 ID，不是 DJI 专属业务语义。
- public reply 注册 API 改为 `WithReplyRouter[T](topic string, router *ReplyRouter[T]) Option`，避免普通 `ConsumeHandler` 误进 reply 路径。
- `ReplyDecoder[T]` 改为 Go 风格接口，提供 `Decode(ctx, payload, topic, topicTemplate) (ReplyMessage[T], error)` 方法。
- 提供 `ReplyDecoderFunc[T]` 适配器，保留用函数快速实现 decoder 的便利性。
- `ReplyMessage[T]` 使用 `Tid string` 表达唯一消息 ID，避免 `ID`、`reply_id`、`correlation_id` 等命名分散。
- reply handler 与普通 handler 可注册在相同或重叠 topic filter 上；同一条消息先尝试 reply router，再继续分发给普通 handler。
- dispatcher 是运行时消息分发器，不负责合并 topic，也不做二次 wildcard 匹配；它使用触发回调的 `topicTemplate` 精准路由。
- 同一 topic 下多个普通 handler 必须按注册顺序执行；topic 列表顺序不作为契约。
- `getAllTopicTemplates()` 只保证包含普通 handler 订阅模板与 reply router 订阅模板，并去重。
- topic 合并/去重只属于订阅恢复阶段，不属于消息 dispatch 阶段。
- reply 全量单测覆盖 router、decoder、dispatch、topic matching 和 subscription topic collection。

## Acceptance Criteria

- [ ] `WithReplyRouter[T](topic string, router *ReplyRouter[T]) Option` 替代 public `WithReplyHandler(topic string, h ConsumeHandler)`。
- [ ] `ReplyDecoder[T]` 是接口，`ReplyDecoderFunc[T]` 是函数适配器。
- [ ] `ReplyMessage[T]` 使用 `Tid string`，`ReplyRouter.Do/Resolve/Reject/Has` 等唯一消息 ID 参数命名为 `tid`。
- [ ] dispatcher 按触发回调的 `topicTemplate` 精准路由，并有 reply+regular 同模板共存和无 handler 测试。
- [ ] reply router 测试覆盖成功匹配、未匹配、decode error、空 tid、nil decoder、Do 成功、send 失败清理、Reject、Close pending。
- [ ] dispatch 测试覆盖 reply router 与普通 handler 同 topic/重叠 topic 共存，且普通 handler 仍执行。
- [ ] handler 顺序测试只验证同一 topic 下普通 handler 的注册顺序，不验证 topic map 迭代顺序。
- [ ] `getAllTopicTemplates()` 测试只验证普通订阅模板 + reply 订阅模板包含和去重，不验证顺序。
- [ ] `go test -count=1 ./common/mqttx/` 通过。

## Out Of Scope

- 不迁移 `common/djisdk` 到新的 `mqttx.ReplyRouter`。
- 不在 `mqttx` 中实现 DJI topic builder、设备 ID 解析、payload schema、`method/result` 或业务错误转换。
- 不要求修改 `common/antsx` 日志字段命名，除非实现过程中发现真实不一致。
