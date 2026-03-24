package ctxprop

import (
	"context"

	"zero-service/common/ctxdata"
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
