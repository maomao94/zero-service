package main

import (
	"flag"
	"fmt"
	"net/http"
	"path/filepath"

	"zero-service/common/gtwx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"
	"zero-service/gtw/internal/config"
	"zero-service/gtw/internal/handler"
	"zero-service/gtw/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/gtw.yaml", "the config file")

func main() {
	//os.Setenv("TMPDIR", "/opt/data/tmp")
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	// grpc-gateway
	//server := gateway.MustNewServer(c.GatewayConf, func(server *gateway.Server) {
	//	server.Use(rest.ToMiddleware(func(next http.Handler) http.Handler {
	//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//			//r.Header.Set("Grpc-Metadata-user-id", "test")
	//			origin := r.Header.Get("Origin")
	//			w.Header().Set("Access-Control-Allow-Origin", origin)
	//			w.Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token,X-Token,X-User-Id")
	//			w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS,DELETE,PUT")
	//			w.Header().Add("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
	//			w.Header().Add("Access-Control-Allow-Credentials", "true")
	//			next.ServeHTTP(w, r)
	//		})
	//	}))
	//})

	gtwx.SetUnifiedErrorHandler()

	server := rest.MustNewServer(c.RestConf,
		gtwx.CorsOption(),
		// 业务错误统一返回 HTTP 200 + {code,msg,data}，鉴权失败、404、405 也走信封。
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

	if len(c.SwaggerPath) > 0 {
		// 静态文件路由，暴露 swagger.json
		server.AddRoute(rest.Route{
			Method: http.MethodGet,
			Path:   "/swagger/:fileName",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				type SwaggerFile struct {
					FileName string `path:"fileName"`
				}
				body := SwaggerFile{}
				err := httpx.Parse(r, &body)
				if err != nil {
					gtwx.WriteParseError(r.Context(), w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				filePath := filepath.Join(c.SwaggerPath, body.FileName)
				http.ServeFile(w, r, filePath)
			},
		})
	}
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
