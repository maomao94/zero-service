package mcpx

import (
	"context"
	"net/http"

	"zero-service/common/ctxprop"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
)

// ExtractCtxFromHeader 从 HTTP Header 提取用户上下文字段，注入 context。
// header 为 nil 时直接返回原 ctx。
func ExtractCtxFromHeader(ctx context.Context, header http.Header) context.Context {
	return ctxprop.ExtractFromHTTPHeader(ctx, header)
}

// WithCtxProp 包装 MCP tool handler，自动从 req.Extra.Header 提取用户上下文注入 ctx。
// Streamable HTTP transport 在每次 POST 请求中填充 RequestExtra.Header，
// 包含 mcpx client 通过 ctxHeaderTransport 注入的用户上下文 HTTP 头。
// 使用方式：mcp.AddTool(server, tool, mcpx.WithCtxProp(handler))
func WithCtxProp[In, Out any](
	h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args In) (*mcp.CallToolResult, Out, error) {
		if req.Extra != nil {
			logx.Debugf("[mcpx] WithCtxProp: extracting headers, hasTokenInfo=%v", req.Extra.TokenInfo != nil)
			ctx = ctxprop.ExtractFromHTTPHeader(ctx, req.Extra.Header)
		}
		return h(ctx, req, args)
	}
}
