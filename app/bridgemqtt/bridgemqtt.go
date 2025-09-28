package main

import (
	"flag"
	"fmt"
	"zero-service/app/bridgemqtt/internal/handler"

	"zero-service/app/bridgemqtt/bridgemqtt"
	"zero-service/app/bridgemqtt/internal/config"
	"zero-service/app/bridgemqtt/internal/server"
	"zero-service/app/bridgemqtt/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/bridgemqtt.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		bridgemqtt.RegisterBridgeMqttServer(grpcServer, server.NewBridgeMqttServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	ctx.MqttClient.AddHandler("testGo", handler.NewTestHandler(ctx))
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
