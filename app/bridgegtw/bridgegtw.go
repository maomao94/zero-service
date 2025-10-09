package main

import (
	"flag"
	"fmt"
	"net/http"

	"zero-service/app/bridgegtw/internal/config"
	"zero-service/app/bridgegtw/internal/handler"
	"zero-service/app/bridgegtw/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/gateway"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/bridgegtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := gateway.MustNewServer(c.GatewayConf, func(server *gateway.Server) {
		server.Use(rest.ToMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
				//w.Header().Set(httpx.ContentType, httpx.JsonContentType)
			})
		}))
	})
	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server.Server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
