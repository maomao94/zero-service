package mcpx

import (
	"context"

	"zero-service/common/authctx"
	"zero-service/common/trace"
)

const ctxMetaKey = "_meta"

// CollectFromCtx collects non-empty authentication context values for MCP _meta.
func CollectFromCtx(ctx context.Context) map[string]any {
	meta := make(map[string]any)
	for _, key := range authctx.ContextKeys {
		if v := authctx.GetByKey(ctx, key); v != "" {
			meta[key] = v
		}
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

// ExtractFromMeta restores authentication values from MCP _meta.
func ExtractFromMeta(ctx context.Context, meta map[string]any) context.Context {
	if len(meta) == 0 {
		return ctx
	}
	for _, key := range authctx.ContextKeys {
		if v := authctx.ClaimString(meta, key); v != "" {
			ctx = authctx.WithKey(ctx, key, v)
		}
	}
	return ctx
}

// WithMeta stores the original MCP _meta map in context.
func WithMeta(ctx context.Context, meta map[string]any) context.Context {
	return context.WithValue(ctx, ctxMetaKey, meta)
}

// GetMeta returns the original MCP _meta map stored in context.
func GetMeta(ctx context.Context) map[string]any {
	if meta, ok := ctx.Value(ctxMetaKey).(map[string]any); ok {
		return meta
	}
	return nil
}

// ExtractTraceFromMeta restores W3C trace context from MCP _meta.
func ExtractTraceFromMeta(ctx context.Context, meta map[string]any) context.Context {
	if len(meta) == 0 {
		return ctx
	}
	return trace.Extract(ctx, trace.NewAnyCarrier(meta))
}
