package mqttx

import "go.opentelemetry.io/otel/propagation"

// 确保 MessageCarrier 实现了 TextMapCarrier 接口
var _ propagation.TextMapCarrier = (*MessageCarrier)(nil)

// MessageCarrier OpenTelemetry TextMapCarrier 实现
// 将 Message 的 Headers 作为链路追踪上下文的载体
type MessageCarrier struct {
	msg *Message
}

// NewMessageCarrier 创建 MessageCarrier
func NewMessageCarrier(msg *Message) MessageCarrier {
	return MessageCarrier{msg: msg}
}

// Get 获取 Header（TextMapCarrier 接口实现）
func (c MessageCarrier) Get(key string) string {
	if c.msg == nil {
		return ""
	}
	return c.msg.GetHeader(key)
}

// Set 设置 Header（TextMapCarrier 接口实现）
func (c MessageCarrier) Set(key, value string) {
	if c.msg == nil {
		return
	}
	c.msg.SetHeader(key, value)
}

// Keys 获取所有 Header key（TextMapCarrier 接口实现）
func (c MessageCarrier) Keys() []string {
	if c.msg == nil || len(c.msg.Headers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(c.msg.Headers))
	for k := range c.msg.Headers {
		keys = append(keys, k)
	}
	return keys
}
