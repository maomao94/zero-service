package main

import (
	"flag"
	"fmt"
	"zero-service/app/ieccaller/cron"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/app/ieccaller/internal/iec"
	"zero-service/app/ieccaller/internal/server"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/iec104/iec104client"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/ieccaller.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		ieccaller.RegisterIecCallerServer(grpcServer, server.NewIecCallerServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(s)

	// client conn
	serviceGroup.Add(iec104client.MustNewIecServerClient(c.IecServerConfig, c.IecCoaConfig, iec.NewClientCall(ctx), ctx.ClientManager))

	// cron
	serviceGroup.Add(cron.NewCronService(ctx))
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
