package main

import (
	"flag"
	"fmt"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/logx"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/common/nacosx"

	"zero-service/facade/iecstream/iecstream"
	"zero-service/facade/iecstream/internal/config"
	"zero-service/facade/iecstream/internal/server"
	"zero-service/facade/iecstream/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	_ "zero-service/common/carbonx"
)

var configFile = flag.String("f", "etc/iecstream.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.RpcServerConf.Middlewares.StatConf.IgnoreContentMethods = []string{iecstream.IecStreamRpc_PushChunkAsdu_FullMethodName}
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		iecstream.RegisterIecStreamRpcServer(grpcServer, server.NewIecStreamRpcServer(ctx))
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
			LogDir:              "/tmp/nacos/log",
			CacheDir:            "/tmp/nacos/cache",
			LogLevel:            "debug",
		}
		m := map[string]string{
			"gRPC_port":                 strutil.After(c.RpcServerConf.ListenOn, ":"),
			"preserved.register.source": "go-zero",
		}
		opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, c.ListenOn, sc, cc, nacosx.WithMetadata(m))
		_ = nacosx.RegisterService(opts)
	}
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
