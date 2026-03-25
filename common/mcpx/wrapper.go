package mcpx

import (
	"context"
	"encoding/json"
	"net/http"

	"zero-service/common/ctxdata"
	"zero-service/common/ctxprop"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/timex"
)

// ExtractCtxFromHeader 从 HTTP Header 提取用户上下文字段，注入 context。
// header 为 nil 时直接返回原 ctx。
func ExtractCtxFromHeader(ctx context.Context, header http.Header) context.Context {
	return ctxprop.ExtractFromHTTPHeader(ctx, header)
}

// CallToolWrapper 包装 MCP tool handler，自动从请求中提取用户上下文注入 ctx。
//
// 支持三种认证路径（按优先级）：
//  1. Streamable transport：req.Extra 由 SDK 填充，从 Header/TokenInfo 提取。
//  2. SSE + mcpx.Client：用户上下文由客户端注入 _meta 字段，从 req.Params._meta 提取。
//  3. SSE 直连 JWT：req.Extra 为 nil 且无 _meta，从连接级 TokenInfo 提取（fallback）。
//
// 使用方式：mcp.AddTool(server, tool, mcpx.CallToolWrapper(handler))
func CallToolWrapper[In, Out any](
	h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args In) (result *mcp.CallToolResult, out Out, err error) {
		start := timex.Now()
		name := req.Params.Name
		defer func() {
			if err != nil {
				logx.WithContext(ctx).WithDuration(timex.Since(start)).Errorf("[mcpx] call tool %v failed, args=%s: %v",
					name, marshalArgs(args), err)
			} else {
				logx.WithContext(ctx).WithDuration(timex.Since(start)).Infof("[mcpx] call tool %v success",
					name)
			}
		}()

		if req.Params != nil {
			meta := req.Params.GetMeta()
			if meta != nil && len(meta) > 0 {
				ctx = ctxprop.ExtractTraceFromMeta(ctx, req.Params.GetMeta())
			}
		}
		if req.Extra != nil {
			logx.WithContext(ctx).Debugf("[mcpx] stream wrapper extra")
			// Path 1: Streamable transport — SDK fills Extra with TokenInfo + Header.
			ctx = ctxprop.ExtractFromHTTPHeader(ctx, req.Extra.Header)
			if ti := req.Extra.TokenInfo; ti != nil {
				if authType, _ := ti.Extra[ctxdata.CtxAuthTypeKey].(string); authType == "user" {
					ctx = ctxprop.ExtractFromClaims(ctx, ti.Extra)
				}
			}
		} else if meta := req.Params.GetMeta(); len(meta) > 0 {
			logx.WithContext(ctx).Debugf("[mcpx] sse wrapper meta")
			// Path 2: SSE with _meta — mcpx.Client injects user context + trace per-message.
			ctx = ctxprop.ExtractFromMeta(ctx, meta)
		} else {
			logx.WithContext(ctx).Debugf("[mcpx] wrapper default")
			// Path 3: SSE direct user JWT — use GET request's TokenInfo in ctx.
			if ti := auth.TokenInfoFromContext(ctx); ti != nil {
				if authType, _ := ti.Extra[ctxdata.CtxAuthTypeKey].(string); authType == "user" {
					ctx = ctxprop.ExtractFromClaims(ctx, ti.Extra)
				}
			}
		}

		return h(ctx, req, args)
	}
}

// marshalArgs 将 args 序列化为 JSON 字符串。
func marshalArgs(args any) string {
	data, err := json.Marshal(args)
	if err != nil {
		return "<marshal error>"
	}
	return string(data)
}
