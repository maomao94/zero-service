package main

import (
	"flag"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/logx"
	interceptor "zero-service/common/Interceptor/rpcserver"

	"zero-service/zeroalarm/internal/config"
	"zero-service/zeroalarm/internal/server"
	"zero-service/zeroalarm/internal/svc"
	"zero-service/zeroalarm/zeroalarm"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/nacos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/zeroalarm.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		zeroalarm.RegisterZeroalarmServer(grpcServer, server.NewZeroalarmServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// register service to nacos
	sc := []constant.ServerConfig{
		*constant.NewServerConfig("10.10.1.103", 8848),
	}
	cc := &constant.ClientConfig{
		NamespaceId:         "public",
		Username:            "nacos",
		Password:            "Zxhc@1234ns",
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		//RotateTime:          "1h",
		//MaxAge:              3,
		LogLevel: "debug",
	}
	opts := nacos.NewNacosConfig("nacos.alarm", c.ListenOn, sc, cc)
	_ = nacos.RegisterService(opts)
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	defer s.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
