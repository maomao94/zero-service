// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"flag"
	"fmt"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/mcp"
)

type EchoArgs struct {
	Message string `json:"message" jsonschema:"description=要回显的消息"`
	Prefix  string `json:"prefix,omitempty" jsonschema:"description=可选的前缀，添加到回显消息前"`
}

var configFile = flag.String("f", "etc/mcpserver.yaml", "the config file")

func main() {
	flag.Parse()

	var c mcp.McpConf
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	server := mcp.NewMcpServer(c)
	// 可选：禁用统计日志
	logx.DisableStat()

	defer server.Stop()

	// 注册一个简单的回显工具
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

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
