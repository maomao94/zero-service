package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/iecmock/kafka"

	"zero-service/app/iecmock/iecmock"
	"zero-service/app/iecmock/internal/config"
	"zero-service/app/iecmock/internal/server"
	"zero-service/app/iecmock/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	_ "zero-service/common/carbonx"
)

var configFile = flag.String("f", "etc/iecmock.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		iecmock.RegisterIecMockRpcServer(grpcServer, server.NewIecMockRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)

	// kafka
	c.KafkaASDUConfig.ServiceConf = c.ServiceConf
	serviceGroup.Add(kq.MustNewQueue(c.KafkaASDUConfig, kafka.NewAsdu(ctx)))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
