# ieccaller 集群指令 ACK 回传设计

## 核心思路

集群部署下，当节点没有目标 IEC104 client 时，发送一条带 traceId 的广播消息，同时在本地注册 Kafka replypool 的等待项。持有目标 client 的节点消费到该广播消息后，通过 onASDU 拿到 IEC104 ACK 响应，再向 broadcast ACK topic 发送一条带同一 traceId 的响应广播。原始发起节点的消费协程收到响应广播后，按 traceId resolve replypool 中的 wait，把响应值返回给 publish 调用方。

## 架构

集群路径需要新增第二条 Kafka 通道（broadcast ACK topic），专门用于回传执行结果。现有 broadcast topic 继续作为请求 fanout 通道。所有实例都可以消费 ACK topic，但只有持有匹配 pending request 的实例会 resolve 等待。

本设计把持有 TCP 连接的节点上的本地 IEC104 command ACK 机制作为事实来源。原始发起节点不应该尝试 resolve 某个 per-client IEC104 `CommandReplyPool`，因为它可能根本不持有目标 client。原始发起节点应该持有一个 service-level Kafka replypool，以 traceId 作为 key。持有目标 IEC104 client 的实例用 `client.WithAck()` 执行指令，把 ACK value 转成和本地 gRPC 直连逻辑相同的 protobuf response，再把该 response 作为 JSON body 发布到 broadcast ACK topic。

## 数据流

1. gRPC command handler 调用 `ClientManager.GetClientOrNil`。
2. 如果目标 client 在本地，保持现有直连逻辑不变。
3. 如果目标 client 不在本地且当前为 cluster broadcast 模式，handler 调用支持 reply 的 broadcast helper，并传入 gRPC method name、request body 和 typed response target。
4. broadcast helper 生成 traceId，在本地 Kafka replypool 中注册 pending wait，发布带 traceId 的 broadcast request，并等待当前 request context。
5. broadcast consumers 继续按现有逻辑忽略来自本地 `BroadcastGroupId` 的消息；其他实例检查 method 和目标 client。
6. 持有目标 client 的实例消费到广播消息后，用 `client.WithAck()` 执行指令，通过 onASDU 拿到 IEC104 ACK 响应，再向 broadcast ACK topic 发布带同一 traceId 的响应广播（包含 status、response JSON、可选 error）。
7. 所有实例消费 broadcast ACK topic。原始发起节点的消费协程收到响应广播后，按 traceId resolve replypool 中的 wait。
8. 原始 helper 把 response JSON unmarshal 到传入的 typed response struct，然后返回给 gRPC handler。

## 契约

- Broadcast request body 保持 `types.BroadcastBody` 兼容；只有请求 ACK reply 时才附加 traceId 和 reply metadata。
- Broadcast ACK body 至少包含：traceId、origin group/id、method、success 标记、response body string、error string。
- Pending reply key 使用 traceId（UUID），不复用 command key，因为多个调用方可能并发发送相同 `coa:typeID:ioa` 指令。
- ACK timeout 应与现有 IEC104 command reply TTL 的 10 秒保持一致；如果 gRPC context deadline 更短，则以更短 deadline 为准。
- Error reply 必须保留足够信息，使原始发起节点可以通过现有 command ACK error wrapping 路径映射超时、重复 pending command、指令拒绝、异常 COT 等错误。

## 兼容性

- 现有 fire-and-forget `PushPbBroadcast` 调用方不应被迫改变行为。
- 现有本地直连 command 路径不应大规模重构；如有必要，只抽取共享 response conversion helper。
- 新增 ACK topic 配置应提供确定性的默认值，例如 `iec-broadcast-ack`，避免影响现有 standalone 部署。

## 风险

- 当前 Kafka broadcast 同时使用 `BroadcastGroupId` 作为 consumer group 和 origin marker。真实 Kafka consumer group 中，同一个 group 的多个实例不会全部收到同一条消息；但现有代码又依赖 ignore same group 的语义。实现时必须尊重当前模式，并在必要时说明集群部署对 group id 的要求。
- 如果持有 client 的实例已经成功执行现场指令，但发布 ACK reply 失败，原始发起节点会超时，即使现场指令可能已经成功。
- 通用 `SendCommand` 有意保持在 ACK reply 路径之外，因为它的 response 为空，并且当前本地路径也不使用 `client.WithAck()`。

## 回滚

因为公开 gRPC API 不变，回滚应主要限于撤销代码和配置新增项。也可以通过保持原有 fire-and-forget broadcast 使用方式或关闭 cluster mode 避免触发新 ACK reply 链路。
