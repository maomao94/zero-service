package main

import (
	"flag"
	"fmt"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/common/nacosx"
	"zero-service/common/tool"

	"zero-service/socketapp/socketpush/internal/config"
	"zero-service/socketapp/socketpush/internal/server"
	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/socketpush.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	// Print Go version
	tool.PrintGoVersion()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		socketpush.RegisterSocketPushServer(grpcServer, server.NewSocketPushServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
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
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
