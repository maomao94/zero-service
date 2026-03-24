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
// 支持两种认证模式：
//   - 服务侧（ServiceToken）：用户上下文由网关通过 HTTP header 透传，
//     从 req.Extra.Header 提取（X-User-Id, X-User-Name 等）。
//   - 用户侧（JWT）：用户身份在 JWT claims 中（user-id, user-name 等），
//     从 req.Extra.TokenInfo.Extra 提取，覆盖 header 中的同名字段。
//
// 兼容 Streamable 和 SSE 两种 transport：
//   - Streamable：req.Extra 由 SDK 填充，直接使用。
//   - SSE：req.Extra 为 nil，从 session context 中提取 TokenInfo 作为 fallback。
//
// 使用方式：mcp.AddTool(server, tool, mcpx.WithCtxProp(handler))
func WithCtxProp[In, Out any](
	h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args In) (*mcp.CallToolResult, Out, error) {
		if req.Extra != nil {
			// Streamable transport: Extra 已填充，走原有路径
			// 1. 始终从 HTTP header 提取（服务侧透传的用户上下文）
			ctx = ctxprop.ExtractFromHTTPHeader(ctx, req.Extra.Header)

			// 2. 用户侧 JWT 认证：从 TokenInfo claims 提取，覆盖 header 值
			if ti := req.Extra.TokenInfo; ti != nil {
				if authType, _ := ti.Extra["type"].(string); authType == "user" {
					ctx = ctxprop.ExtractFromClaims(ctx, ti.Extra)
				}
			}

			logx.Debugf("[mcpx] WithCtxProp: userId=%s, authType=%s",
				ctxdata.GetUserId(ctx), getAuthType(req))
		} else {
			// SSE transport fallback: req.Extra 为 nil，从 session context 中提取 TokenInfo
			if ti := auth.TokenInfoFromContext(ctx); ti != nil {
				if authType, _ := ti.Extra["type"].(string); authType == "user" {
					ctx = ctxprop.ExtractFromClaims(ctx, ti.Extra)
				}
				logx.Debugf("[mcpx] WithCtxProp(fallback): userId=%s, authType=%s",
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
	if t, ok := ti.Extra["type"].(string); ok {
		return t
	}
	return "unknown"
}
