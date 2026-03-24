package tools

import (
	"context"

	"github.com/zeromicro/go-zero/mcp"
)

// EchoArgs echo 工具参数
type EchoArgs struct {
	Message string `json:"message" jsonschema:"要回显的消息"`
	Prefix  string `json:"prefix,omitempty" jsonschema:"可选的前缀，添加到回显消息前"`
}

// RegisterEcho 注册 echo 工具
func RegisterEcho(server mcp.McpServer) {
	echoTool := &mcp.Tool{
		Name:        "echo",
		Description: "回显用户提供的消息",
	}

	echoHandler := func(ctx context.Context, req *mcp.CallToolRequest, args EchoArgs) (*mcp.CallToolResult, any, error) {
		prefix := "Echo: "
		if len(args.Prefix) > 0 {
			prefix = args.Prefix
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: prefix + args.Message},
			},
		}, nil, nil
	}

	mcp.AddTool(server, echoTool, echoHandler)
}
