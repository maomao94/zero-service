package main

import (
	"flag"
	"fmt"
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/config"
	"zero-service/app/bridgemodbus/internal/server"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/grpcx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/bridgemodbus.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)
	defer ctx.Manager.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		bridgemodbus.RegisterBridgeModbusServer(grpcServer, server.NewBridgeModbusServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

	// register service to nacos
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	defer s.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
