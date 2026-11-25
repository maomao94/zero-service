package main

import (
	"flag"
	"fmt"
	"zero-service/common/grpcx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/config"
	"zero-service/app/gis/internal/server"
	"zero-service/app/gis/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/gis.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	// Print GEOS/H3/SQLite versions
	printDependencyVersions()

	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		gis.RegisterGisServer(grpcServer, server.NewGisServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// register service to nacos
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
