package mcpx

import (
	"context"
	"net/http"

	"zero-service/common/ctxdata"
	"zero-service/common/ctxprop"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
)

// ExtractCtxFromHeader 从 HTTP Header 提取用户上下文字段，注入 context。
// header 为 nil 时直接返回原 ctx。
func ExtractCtxFromHeader(ctx context.Context, header http.Header) context.Context {
	return ctxprop.ExtractFromHTTPHeader(ctx, header)
}

// WithCtxProp 包装 MCP tool handler，自动从请求中提取用户上下文注入 ctx。
//
// 支持三种认证路径（按优先级）：
//  1. Streamable transport：req.Extra 由 SDK 填充，从 Header/TokenInfo 提取。
//  2. SSE + mcpx.Client：用户上下文由客户端注入 _meta 字段，从 req.Params._meta 提取。
//  3. SSE 直连 JWT：req.Extra 为 nil 且无 _meta，从连接级 TokenInfo 提取（fallback）。
//
// 使用方式：mcp.AddTool(server, tool, mcpx.WithCtxProp(handler))
func WithCtxProp[In, Out any](
	h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args In) (*mcp.CallToolResult, Out, error) {
		if req.Extra != nil {
			// Path 1: Streamable transport — SDK fills Extra with TokenInfo + Header.
			ctx = ctxprop.ExtractFromHTTPHeader(ctx, req.Extra.Header)
			if ti := req.Extra.TokenInfo; ti != nil {
				if authType, _ := ti.Extra[ctxdata.CtxAuthTypeKey].(string); authType == "user" {
					ctx = ctxprop.ExtractFromClaims(ctx, ti.Extra)
				}
			}
			logx.WithContext(ctx).Debugf("[mcpx] WithCtxProp: userId=%s, authType=%s",
				ctxdata.GetUserId(ctx), getAuthType(req))
		} else if meta := req.Params.GetMeta(); len(meta) > 0 {
			// Path 2: SSE with _meta — mcpx.Client injects user context per-message.
			ctx = ctxprop.ExtractFromMeta(ctx, meta)
			logx.WithContext(ctx).Debugf("[mcpx] WithCtxProp(meta): userId=%s", ctxdata.GetUserId(ctx))
		} else {
			// Path 3: SSE direct user JWT — use GET request's TokenInfo in ctx.
			if ti := auth.TokenInfoFromContext(ctx); ti != nil {
				if authType, _ := ti.Extra[ctxdata.CtxAuthTypeKey].(string); authType == "user" {
					ctx = ctxprop.ExtractFromClaims(ctx, ti.Extra)
				}
				logx.WithContext(ctx).Debugf("[mcpx] WithCtxProp(fallback): userId=%s, authType=%s",
					ctxdata.GetUserId(ctx), getAuthTypeFromTokenInfo(ti))
			}
		}
		return h(ctx, req, args)
	}
}

// getAuthType 从 req.Extra.TokenInfo 提取认证类型标识。
func getAuthType(req *mcp.CallToolRequest) string {
	if req.Extra == nil || req.Extra.TokenInfo == nil {
		return "none"
	}
	return getAuthTypeFromTokenInfo(req.Extra.TokenInfo)
}

// getAuthTypeFromTokenInfo 从 TokenInfo 提取认证类型标识。
func getAuthTypeFromTokenInfo(ti *auth.TokenInfo) string {
	if ti == nil {
		return "none"
	}
	if t, ok := ti.Extra[ctxdata.CtxAuthTypeKey].(string); ok {
		return t
	}
	return "unknown"
}
