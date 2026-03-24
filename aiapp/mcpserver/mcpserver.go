package main

import (
	"flag"
	"fmt"
	"net/http"

	"zero-service/aiapp/mcpserver/internal/config"
	"zero-service/aiapp/mcpserver/internal/svc"
	"zero-service/aiapp/mcpserver/internal/tools"
	"zero-service/common/mcpx"
	"zero-service/common/tool"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/mcpserver.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()
	logx.DisableStat()

	// 1. 创建 MCP SDK Server
	mcpServer := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    c.Mcp.Name,
		Version: c.Mcp.Version,
	}, nil)

	// 2. 注册所有工具
	svcCtx := svc.NewServiceContext(c)
	tools.RegisterAll(mcpServer, svcCtx)

	// 3. 创建 Streamable HTTP Handler
	streamHandler := sdkmcp.NewStreamableHTTPHandler(
		func(r *http.Request) *sdkmcp.Server { return mcpServer },
		&sdkmcp.StreamableHTTPOptions{
			SessionTimeout: c.Mcp.SessionTimeout,
		},
	)

	// 4. 包装 auth 中间件（双模式验证）
	verifier := mcpx.NewDualTokenVerifier(c.Auth.JwtSecrets, c.Auth.ServiceToken)
	authedHandler := auth.RequireBearerToken(verifier, nil)(streamHandler)

	// 5. 创建 go-zero REST server
	var restOpts []rest.RunOption
	if len(c.Mcp.Cors) > 0 {
		restOpts = append(restOpts, rest.WithCors(c.Mcp.Cors...))
	}
	httpServer := rest.MustNewServer(c.RestConf, restOpts...)
	defer httpServer.Stop()

	// 6. 注册 MCP 路由（GET + POST + DELETE for Streamable HTTP）
	endpoint := c.Mcp.MessageEndpoint
	httpServer.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: endpoint, Handler: authedHandler.ServeHTTP},
		{Method: http.MethodPost, Path: endpoint, Handler: authedHandler.ServeHTTP},
		{Method: http.MethodDelete, Path: endpoint, Handler: authedHandler.ServeHTTP},
	})

	fmt.Printf("Starting MCP server at %s:%d%s ...\n", c.Host, c.Port, endpoint)
	httpServer.Start()
}
