package tools

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// EchoArgs echo 工具参数
type EchoArgs struct {
	Message string `json:"message" jsonschema:"要回显的消息"`
	Prefix  string `json:"prefix,omitempty" jsonschema:"可选的前缀，添加到回显消息前"`
}

// RegisterEcho 注册 echo 工具
func RegisterEcho(server *sdkmcp.Server) {
	echoTool := &sdkmcp.Tool{
		Name:        "echo",
		Description: "回显用户提供的消息",
	}

	echoHandler := func(ctx context.Context, req *sdkmcp.CallToolRequest, args EchoArgs) (*sdkmcp.CallToolResult, any, error) {
		prefix := "Echo: "
		if len(args.Prefix) > 0 {
			prefix = args.Prefix
		}

		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{
				&sdkmcp.TextContent{Text: prefix + args.Message},
			},
		}, nil, nil
	}

	sdkmcp.AddTool(server, echoTool, echoHandler)
}
