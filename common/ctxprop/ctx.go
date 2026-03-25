package ctxprop

import (
	"context"

	"zero-service/common/ctxdata"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// CollectFromCtx 从 context values 中提取所有 PropFields，收集为 map。
// 用于 MCP 客户端：将用户上下文注入 JSON-RPC 请求的 _meta 字段。
// 返回 nil 表示无可用字段。
func CollectFromCtx(ctx context.Context) map[string]any {
	m := make(map[string]any)
	for _, f := range ctxdata.PropFields {
		if v, ok := ctx.Value(f.CtxKey).(string); ok && v != "" {
			m[f.CtxKey] = v
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// ExtractFromMeta 从 _meta map 提取 PropFields，注入到 context values。
// 用于 MCP 服务端 handler：将 SSE 请求中 _meta 携带的用户上下文恢复到 context。
// _meta 中的值可能是 string 或 float64（JSON number），使用 ClaimString 统一处理。
func ExtractFromMeta(ctx context.Context, meta map[string]any) context.Context {
	if len(meta) == 0 {
		return ctx
	}
	for _, f := range ctxdata.PropFields {
		if v := ClaimString(meta, f.CtxKey); v != "" {
			ctx = context.WithValue(ctx, f.CtxKey, v)
		}
	}
	return ctx
}

// ExtractTraceFromMeta 从 _meta map 提取链路信息，注入到 context。
// 用于 MCP 服务端 handler：从 SSE 请求中 _meta 携带的链路信息（W3C traceparent）恢复链路上下文。
func ExtractTraceFromMeta(ctx context.Context, meta map[string]any) context.Context {
	if len(meta) == 0 {
		return ctx
	}
	carrier := &mapMetaCarrier{meta: meta}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// mapMetaCarrier _meta map 的 TextMapCarrier 实现
type mapMetaCarrier struct {
	meta map[string]any
}

func (c *mapMetaCarrier) Get(key string) string {
	if v, ok := c.meta[key].(string); ok {
		return v
	}
	return ""
}

func (c *mapMetaCarrier) Set(key string, value string) {
	c.meta[key] = value
}

func (c *mapMetaCarrier) Keys() []string {
	keys := make([]string, 0, len(c.meta))
	for k := range c.meta {
		keys = append(keys, k)
	}
	return keys
}

var _ propagation.TextMapCarrier = (*mapMetaCarrier)(nil)
