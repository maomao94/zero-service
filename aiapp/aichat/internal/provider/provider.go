package provider

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Provider 大模型 API 提供者接口
type Provider interface {
	// ChatCompletion 非流式对话补全
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	// ChatCompletionStream 流式对话补全
	ChatCompletionStream(ctx context.Context, req *ChatRequest) (StreamReader, error)
}

// StreamReader 流式响应读取器
type StreamReader interface {
	// Recv 读取下一个 chunk，io.EOF 表示结束
	Recv() (*StreamChunk, error)
	// Close 释放资源（关闭 HTTP response body 等）
	Close() error
}

// ToGrpcError 将 provider 错误转为 gRPC status error
func ToGrpcError(err error) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == 401 || apiErr.StatusCode == 403:
			return status.Errorf(codes.PermissionDenied, "upstream auth error: %s", apiErr.Body)
		case apiErr.StatusCode == 429:
			return status.Errorf(codes.ResourceExhausted, "upstream rate limit: %s", apiErr.Body)
		case apiErr.StatusCode == 400:
			return status.Errorf(codes.InvalidArgument, "upstream bad request: %s", apiErr.Body)
		default:
			return status.Errorf(codes.Unavailable, "upstream error (status %d): %s", apiErr.StatusCode, apiErr.Body)
		}
	}
	return status.Errorf(codes.Internal, "internal error: %v", err)
}

// McpToolsToOpenAI 将 MCP 工具转换为 OpenAI function calling 格式
func McpToolsToOpenAI(tools []*mcp.Tool) []ToolDef {
	defs := make([]ToolDef, len(tools))
	for i, t := range tools {
		defs[i] = ToolDef{
			Type: "function",
			Function: ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
	}
	return defs
}
