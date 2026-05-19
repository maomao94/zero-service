package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Carrier 对标 go-queue 的 MessageCarrier，将 message metadata（map[string]string）
// 适配为 OpenTelemetry 链路传播载体，适用于 MQTT/Kafka/WSS 等消息系统。
type Carrier struct {
	headers map[string]string
}

var _ propagation.TextMapCarrier = (*Carrier)(nil)

func NewCarrier(headers map[string]string) Carrier {
	if headers == nil {
		headers = make(map[string]string)
	}
	return Carrier{headers: headers}
}

func (c Carrier) Get(key string) string {
	return c.headers[key]
}

func (c Carrier) Set(key, value string) {
	c.headers[key] = value
}

func (c Carrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}

// Inject 从 ctx 中提取 OTel span context 注入到 carrier（producer 端）。
func Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// Extract 从 carrier 恢复 OTel span context（consumer 端）。
func Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// AnyCarrier 用于 map[string]any 的 TextMapCarrier 实现（MCP _meta 等场景）。
type AnyCarrier struct {
	meta map[string]any
}

var _ propagation.TextMapCarrier = (*AnyCarrier)(nil)

func NewAnyCarrier(meta map[string]any) AnyCarrier {
	if meta == nil {
		meta = make(map[string]any)
	}
	return AnyCarrier{meta: meta}
}

func (c AnyCarrier) Get(key string) string {
	if v, ok := c.meta[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c AnyCarrier) Set(key, value string) {
	c.meta[key] = value
}

func (c AnyCarrier) Keys() []string {
	keys := make([]string, 0, len(c.meta))
	for k := range c.meta {
		keys = append(keys, k)
	}
	return keys
}
