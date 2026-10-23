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

	"zero-service/common/gtwx"
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

	server := rest.MustNewServer(c.RestConf,
		gtwx.CorsOption(),
		// 非流式接口统一返回 HTTP 200 + {code,msg,data}，鉴权失败、404、405 也走信封。
		rest.WithUnauthorizedCallback(gtwx.UnauthorizedHandler),
		rest.WithNotFoundHandler(http.HandlerFunc(gtwx.NotFoundHandler)),
		rest.WithNotAllowedHandler(http.HandlerFunc(gtwx.MethodNotAllowedHandler)),
	)

	// 请求 ID 优先复用调用方传入值，缺失时生成并回显
	server.Use(gtwx.RequestIDMiddleware)

	// 请求日志（method, path, duration 写入 context，供 ok/error handler 读取）
	server.Use(gtwx.RequestLogMiddleware)

	// 响应日志
	gtwx.SetLogOkHandler()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
