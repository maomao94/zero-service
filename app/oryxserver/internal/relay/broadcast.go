package relay

// MQTT 广播方法常量（cluster 模式跨节点停止转推，对齐 ieccaller 模式）。
// 主题定义与广播/ack body 由 common/mqttx/broadcast 统一提供（前缀 "oryx/server"，容器级命名）。
const (
	// MethodStreamRelayStop 广播方法名（对齐 gRPC full method）
	MethodStreamRelayStop = "/oryxserver.OryxServer/StreamRelayStop"
)
