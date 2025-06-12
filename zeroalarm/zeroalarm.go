package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/zeroalarm/internal/config"
	"zero-service/zeroalarm/internal/server"
	"zero-service/zeroalarm/internal/svc"
	"zero-service/zeroalarm/zeroalarm"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	_ "zero-service/common/carbonx"
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
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	defer s.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
