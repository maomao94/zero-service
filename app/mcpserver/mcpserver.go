// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/mcp"
)

var configFile = flag.String("f", "etc/mcpserver.yaml", "the config file")

func main() {
	flag.Parse()

	var c mcp.McpConf
	conf.MustLoad(*configFile, &c)

	server := mcp.NewMcpServer(c)
	// 可选：禁用统计日志
	logx.DisableStat()

	defer server.Stop()

	// 注册一个简单的回显工具
	echoTool := mcp.Tool{
		Name:        "echo",
		Description: "回显用户提供的消息",
		InputSchema: mcp.InputSchema{
			Properties: map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "要回显的消息",
				},
				"prefix": map[string]any{
					"type":        "string",
					"description": "可选的前缀，添加到回显消息前",
					"default":     "Echo: ",
				},
			},
			Required: []string{"message"},
		},
		Handler: func(ctx context.Context, params map[string]any) (any, error) {
			var req struct {
				Message string `json:"message"`
				Prefix  string `json:"prefix,optional"`
			}

			if err := mcp.ParseArguments(params, &req); err != nil {
				return nil, fmt.Errorf("failed to parse params: %w", err)
			}

			prefix := "Echo: "
			if len(req.Prefix) > 0 {
				prefix = req.Prefix
			}

			return prefix + req.Message, nil
		},
	}

	server.RegisterTool(echoTool)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
