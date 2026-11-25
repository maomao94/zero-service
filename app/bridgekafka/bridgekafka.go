package main

import (
	"flag"
	"fmt"

	"zero-service/app/bridgekafka/bridgekafka"
	"zero-service/app/bridgekafka/internal/config"
	"zero-service/app/bridgekafka/internal/handler"
	"zero-service/app/bridgekafka/internal/server"
	"zero-service/app/bridgekafka/internal/svc"
	_ "zero-service/common/carbonx"
	"zero-service/common/grpcx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/bridgekafka.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	for i := range c.KafkaConsumeConfig {
		kc := c.KafkaConsumeConfig[i]
		if len(kc.Brokers) == 0 || kc.Topic == "" {
			continue
		}
		fullConf := kc.ToKqConf(c.ServiceConf)
		h := handler.NewKafkaStreamHandler(kc.Topic, kc.Group, ctx.StreamEventCli)
		serviceGroup.Add(kq.MustNewQueue(fullConf, h))
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		bridgekafka.RegisterBridgeKafkaServer(grpcServer, server.NewBridgeKafkaServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

	// register service to nacos
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)

	serviceGroup.Add(s)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
