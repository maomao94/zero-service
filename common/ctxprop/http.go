package ctxprop

import (
	"context"
	"net/http"

	"zero-service/common/ctxdata"
)

// InjectToHTTPHeader 从 context values 提取所有字段，注入到 HTTP header。
// 用于 MCP 客户端 Transport：将上下文字段传播到 MCP 服务器。
func InjectToHTTPHeader(ctx context.Context, header http.Header) {
	for _, f := range ctxdata.PropFields {
		if v, ok := ctx.Value(f.CtxKey).(string); ok && v != "" {
			header.Set(f.HttpHeader, v)
		}
	}
}

// ExtractFromHTTPHeader 从 HTTP header 提取所有字段，注入到 context values。
// 用于 MCP 服务端 handler：将 HTTP header 中的字段恢复到 context。
func ExtractFromHTTPHeader(ctx context.Context, header http.Header) context.Context {
	if len(header) == 0 {
		return ctx
	}
	for _, f := range ctxdata.PropFields {
		if v := header.Get(f.HttpHeader); v != "" {
			ctx = context.WithValue(ctx, f.CtxKey, v)
		}
	}
	return ctx
}
