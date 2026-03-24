package main

import (
	"flag"
	"fmt"

	"zero-service/aiapp/mcpserver/internal/config"
	"zero-service/aiapp/mcpserver/internal/svc"
	"zero-service/aiapp/mcpserver/internal/tools"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/mcp"
)

var configFile = flag.String("f", "etc/mcpserver.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()
	logx.DisableStat()

	svcCtx := svc.NewServiceContext(c)
	server := mcp.NewMcpServer(c.McpConf)
	defer server.Stop()

	// 统一注册所有工具
	tools.RegisterAll(server, svcCtx)

	fmt.Printf("Starting MCP server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
