package mqttx

// Message MQTT 消息包装结构
// 用于封装消息和链路追踪上下文
type Message struct {
	// Topic 消息主题
	Topic string `json:"topic"`
	// Payload 消息内容
	Payload []byte `json:"payload"`
	// Headers 自定义 Header，用于传递链路追踪上下文等信息
	Headers map[string]string `json:"headers,omitempty"`
}

// NewMessage 创建消息
func NewMessage(topic string, payload []byte) *Message {
	return &Message{
		Topic:   topic,
		Payload: payload,
		Headers: make(map[string]string),
	}
}

// GetHeader 获取 Header 值
func (m *Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

// SetHeader 设置 Header
func (m *Message) SetHeader(key, val string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = val
}
