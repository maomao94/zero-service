package main

import (
	"flag"
	"fmt"
	interceptor "zero-service/common/Interceptor/rpcserver"

	"zero-service/app/ispserver/internal/config"
	"zero-service/app/ispserver/internal/server"
	"zero-service/app/ispserver/internal/svc"
	"zero-service/app/ispserver/ispserver"

	_ "zero-service/common/carbonx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/ispserver.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		ispserver.RegisterIspServerServer(grpcServer, server.NewIspServerServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(s)
	serviceGroup.Add(ctx.IspServer)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	fmt.Printf("ISP TCP server listening on %s...\n", c.IspConf.ListenAddr)
	serviceGroup.Start()
}
