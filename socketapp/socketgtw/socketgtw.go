package main

import (
	"flag"
	"fmt"
	"net/http"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/common/nacosx"
	"zero-service/common/tool"
	"zero-service/socketapp/socketgtw/internal/config"
	"zero-service/socketapp/socketgtw/internal/handler"
	"zero-service/socketapp/socketgtw/internal/server"
	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/chain"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/socketgtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	// Print Go version
	tool.PrintGoVersion()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		socketgtw.RegisterSocketGtwServer(grpcServer, server.NewSocketGtwServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	c.Http.Name = c.Name
	socketTicketMiddleware := func() func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strutil.ContainsAny(r.URL.Path, []string{"/socket.io"}) {
					ticket := r.URL.Query().Get("ticket")
					if len(ticket) != 0 {
						r.Header.Set("Authorization", ticket)
					}
				}
				next.ServeHTTP(w, r)
			})
		}
	}
	httpServer := rest.MustNewServer(c.Http, rest.WithChain(chain.New(socketTicketMiddleware())))
	handler.RegisterHandlers(httpServer, ctx)
	// register service to nacos
	if c.NacosConfig.IsRegister {
		sc := []constant.ServerConfig{
			*constant.NewServerConfig(c.NacosConfig.Host, c.NacosConfig.Port),
		}
		cc := &constant.ClientConfig{
			NamespaceId:         c.NacosConfig.NamespaceId,
			Username:            c.NacosConfig.Username,
			Password:            c.NacosConfig.PassWord,
			TimeoutMs:           5000,
			NotLoadCacheAtStart: true,
		}
		m := map[string]string{
			"gRPC_port":                 strutil.After(c.RpcServerConf.ListenOn, ":"),
			"preserved.register.source": "go-zero",
		}
		opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, c.ListenOn, sc, cc, nacosx.WithMetadata(m))
		_ = nacosx.RegisterService(opts)
	}
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(httpServer)
	serviceGroup.Add(s)
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	fmt.Printf("Starting server at %s:%d...\n", c.Http.Host, c.Http.Port)
	serviceGroup.Start()
}
