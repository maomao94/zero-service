// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"net/http"

	"zero-service/aiapp/ssegtw/internal/config"
	"zero-service/aiapp/ssegtw/internal/handler"
	"zero-service/aiapp/ssegtw/internal/svc"

	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/ssegtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	server := rest.MustNewServer(c.RestConf, rest.WithCustomCors(func(header http.Header) {
		origin := header.Get("Origin")
		if origin != "" {
			header.Set("Access-Control-Allow-Origin", origin)
		}
		header.Set("Vary", "Origin")

		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		header.Set("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, X-Token, X-User-Id")
		header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
	}, nil, "*"))

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
