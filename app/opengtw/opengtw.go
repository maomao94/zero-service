package main

import (
	"flag"
	"fmt"
	"net/http"

	"zero-service/app/opengtw/internal/config"
	"zero-service/app/opengtw/internal/handler"
	"zero-service/app/opengtw/internal/middleware"
	"zero-service/app/opengtw/internal/svc"
	_ "zero-service/common/carbonx"
	"zero-service/common/gtwx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/opengtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()

	server := rest.MustNewServer(c.RestConf,
		gtwx.CorsOption(),
		// 管理面 JWT 失败也返回 HTTP 200 信封，错误码统一为 100401。
		rest.WithUnauthorizedCallback(gtwx.UnauthorizedHandler),
		rest.WithNotFoundHandler(http.HandlerFunc(gtwx.NotFoundHandler)),
		rest.WithNotAllowedHandler(http.HandlerFunc(gtwx.MethodNotAllowedHandler)),
	)
	defer server.Stop()

	// 为所有请求（含鉴权失败、解析失败）准备并回显 X-Request-Id
	server.Use(middleware.RequestIDMiddleware)

	// 请求日志（method, path, duration 写入 context，供 ok/error handler 读取）
	server.Use(gtwx.RequestLogMiddleware)

	// 响应日志
	gtwx.SetLogOkHandler()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
