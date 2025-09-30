package mqttx

import "go.opentelemetry.io/otel/propagation"

// 确保实现接口
var _ propagation.TextMapCarrier = (*MessageCarrier)(nil)

type MessageCarrier struct {
	msg *Message
}

func NewMessageCarrier(msg *Message) MessageCarrier {
	return MessageCarrier{msg: msg}
}

func (c MessageCarrier) Get(key string) string {
	return c.msg.GetHeader(key)
}

func (c MessageCarrier) Set(key string, value string) {
	c.msg.SetHeader(key, value)
}

func (c MessageCarrier) Keys() []string {
	keys := make([]string, 0, len(c.msg.Headers))
	for k := range c.msg.Headers {
		keys = append(keys, k)
	}
	return keys
}
