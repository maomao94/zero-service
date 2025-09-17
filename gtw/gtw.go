package main

import (
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"zero-service/gtw/internal/config"
	"zero-service/gtw/internal/handler"
	"zero-service/gtw/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/gateway"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	_ "zero-service/common/nacosx"

	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/gtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// grpc-gateway
	server := gateway.MustNewServer(c.GatewayConf, func(server *gateway.Server) {
		server.Use(rest.ToMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				//r.Header.Set("Grpc-Metadata-user-id", "test")
				origin := r.Header.Get("Origin")
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token,X-Token,X-User-Id")
				w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS,DELETE,PUT")
				w.Header().Add("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
				w.Header().Add("Access-Control-Allow-Credentials", "true")
				next.ServeHTTP(w, r)
			})
		}))
	})
	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server.Server, ctx)
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
					httpx.ErrorCtx(r.Context(), w, err)
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
