package broadcast

import "strings"

// ==================== Broadcast Topic ====================
//
// Broadcast Topic 用于集群节点间的广播命令下发与 ack 确认，基础路径格式: {prefix}/broadcast[_reply]/{...}。
// prefix 由业务 app 传入（如 oryxserver 传 "oryx/server"、ieccaller 传 "iec"），本包不硬编码业务前缀。
// 广播通道无通配需求（要求每个节点都能收到，具体订阅即可）；ack 通道按实例隔离，并额外提供
// 通配订阅模式用于全景订阅/监控。

// Prefix 用 "/" 连接多个分段构造主题前缀。
//
// 示例: Prefix("oryx", "server") => "oryx/server"
func Prefix(parts ...string) string {
	return strings.Join(parts, "/")
}

// BroadcastTopic 返回集群广播命令主题。
// 路径格式: {prefix}/broadcast（如 oryx/server/broadcast、iec/broadcast）
// 方向: 集群内任一节点 → 全部节点
// 用途: 请求方将该节点需要其他节点执行的命令广播到集群（如停止转推、IEC104 命令执行）。
func BroadcastTopic(prefix string) string {
	return Prefix(prefix, "broadcast")
}

// BroadcastTopicPattern 返回广播主题的订阅模式。
// 路径格式: {prefix}/broadcast（与 BroadcastTopic 同值：具体订阅，无通配）
// 方向: 集群内任一节点 → 全部节点（各节点具体订阅接收）
// 用途: 消费注册的统一锚点，与 BroadcastTopic 保持一一对应防止主题漂移。
func BroadcastTopicPattern(prefix string) string {
	return BroadcastTopic(prefix)
}

// BroadcastAckTopic 返回指定实例的广播 ack 回复主题。
// 路径格式: {prefix}/broadcast_reply/{instanceID}
// 方向: 执行节点 → 请求节点
// 用途: 任意节点收到广播并执行后将 ack 回复到请求方的专属实例通道，请求方按 tid 关联等待。
func BroadcastAckTopic(prefix, instanceID string) string {
	return Prefix(prefix, "broadcast_reply", instanceID)
}

// BroadcastAckTopicPattern 返回广播 ack 主题的通配订阅模式。
// 路径格式: {prefix}/broadcast_reply/+
// 方向: 执行节点 → 请求节点（订阅侧可通配监听）
// 用途: 集群内单实例订阅自身 ack 通道；监控/调试可通配订阅所有实例的 ack。
func BroadcastAckTopicPattern(prefix string) string {
	return Prefix(prefix, "broadcast_reply", "+")
}
