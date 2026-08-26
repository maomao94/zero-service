package broadcast

// BroadcastBody 广播请求体（协议中立层：只含关联/路由字段，业务数据走 opaque payload）。
// 字段说明：
//   - Tid:      请求方生成的关联 ID（SDK 发送时自动生成，调用方不构造）
//   - AckTopic: 请求方自己的 ack 回复主题，消费方按此回发（SDK 内部填充，防回环判定依赖它）
//   - Method:   业务方法名（对齐 gRPC full method，如 "/oryxserver.OryxServer/StopRelayPull"）
//   - Body:     opaque 业务 payload（string 承载字节，格式由业务 executor 自行约定，
//     如 ieccaller/oryxserver 的 gRPC 请求 protojson；SDK 不感知）
type BroadcastBody struct {
	Tid      string `json:"tId,omitempty"`
	AckTopic string `json:"ackTopic"`
	Method   string `json:"method"`
	Body     string `json:"body,omitempty"`
}

// BroadcastAckBody 广播 ack 响应体，用于集群模式下回传指令执行结果（协议中立层）。
// 字段说明：
//   - ResponseBody: opaque 业务结果 payload（string 承载字节，格式由业务 executor 自行约定，
//     如 ieccaller 的 protojson；SDK 不感知）
//   - ErrorKind:    错误类别（内置 timeout/duplicate/unknown，业务类别由 app 注册，见 errors.go）
type BroadcastAckBody struct {
	Tid          string `json:"tId"`
	Method       string `json:"method"`
	Success      bool   `json:"success"`
	ResponseBody string `json:"responseBody,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorKind    string `json:"errorKind,omitempty"`
}
