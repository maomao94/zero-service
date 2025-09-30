package mqttx

type Message struct {
	Topic   string            `json:"topic"`
	Payload []byte            `json:"payload"`
	Headers map[string]string `json:"headers,omitempty"` // 自定义的 header 容器
}

func NewMessage(topic string, payload []byte) *Message {
	return &Message{
		Topic:   topic,
		Payload: payload,
		Headers: make(map[string]string),
	}
}

func (m *Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

func (m *Message) SetHeader(key, val string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = val
}
