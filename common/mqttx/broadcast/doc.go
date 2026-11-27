// Package broadcast 基于 mqttx 的集群广播 + 按实例 ack 协议泛化层。
//
// 背景：oryxserver（转推停止广播）与 ieccaller（IEC104 命令集群广播）各自实现了一套高度
// 同构的 MQTT 广播 + ack 协议（主题定义、请求/应答 body、ack 解码器、errorKind 归一、
// 防回环、ack 回发），本包将协议层泛化，业务 method 分发留在各 app 以 Executor 注册。
//
// 协议形状（线格式，{prefix} 由 app 传入）：
//   - 广播：{prefix}/broadcast（具体订阅，请求方发布）
//   - ack：  {prefix}/broadcast_reply/{instanceID}（每实例一个 ack 通道，请求方订阅等待）
//
// 约定：
//   - BroadcastBody.AckTopic 由 SDK 填充为请求方自己的 ack 主题；接收方以此判断是否为自身消息。
//   - 广播方只需保证 client 已创建；ack 应答 router 由 NewBroadcaster 内部创建并
//     AddReplyRouter 注册到 BroadcastAckTopic(prefix, instanceID)，BroadcastReply
//     直接调用它的 RequestReply 等待 ack。
//   - 消息统一走 Publish 发布，ctx 携带 span 时自动内联 trace 头，消费侧由 mqttx 自动续接。
//
// 用法示例（app 侧）：
//
//	client := mqttx.MustNewClient(cfg)
//	btc := broadcast.NewBroadcaster(client, id, broadcast.WithPrefix(prefix), broadcast.WithReplyTTL(10*time.Second))
//	btc.AddExecutor(method, myExecutor)
//	btc.AddBroadcastHandler()
//
// 本包运行于集群内每个业务节点：既发布也消费。
package broadcast
