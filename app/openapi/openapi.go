package main

import (
	"flag"
	"fmt"

	"zero-service/app/openapi/internal/config"
	"zero-service/app/openapi/internal/server"
	"zero-service/app/openapi/internal/svc"
	"zero-service/app/openapi/openapi"
	_ "zero-service/common/carbonx"
	"zero-service/common/grpcx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/openapi.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	tool.PrintGoVersion()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		openapi.RegisterOpenApiRpcServer(grpcServer, server.NewOpenApiRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
