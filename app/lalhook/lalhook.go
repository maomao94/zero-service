package main

import (
	"flag"
	"fmt"
	"net/http"

	"zero-service/app/lalhook/internal/config"
	"zero-service/app/lalhook/internal/handler"
	"zero-service/app/lalhook/internal/svc"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/lalhook.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	server := rest.MustNewServer(c.RestConf, rest.WithCustomCors(func(header http.Header) {
		origin := header.Get("Origin") // 动态获取请求域名
		if origin != "" {
			header.Set("Access-Control-Allow-Origin", origin) // 指定允许的域
		}
		header.Set("Vary", "Origin") // 避免缓存污染

		header.Set("Access-Control-Allow-Credentials", "true")                                                                          // 允许携带 Cookie/Token
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")                                            // 支持的请求方法
		header.Set("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, X-Token, X-User-Id") // 支持的请求头
		header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")                                                     // 前端可以读取的响应头

	}, nil, "*"))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
