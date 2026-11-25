package main

import (
	"flag"
	"fmt"
	"zero-service/common/grpcx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/config"
	"zero-service/app/djicloud/internal/server"
	"zero-service/app/djicloud/internal/svc"

	_ "zero-service/common/carbonx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/djicloud.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		djicloud.RegisterDjiCloudServer(grpcServer, server.NewDjiCloudServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// register service to nacos
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	group := service.NewServiceGroup()
	defer group.Stop()
	group.Add(s)
	group.Add(svc.NewDeviceOnlineRefreshCron(ctx.DB))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	group.Start()
}
