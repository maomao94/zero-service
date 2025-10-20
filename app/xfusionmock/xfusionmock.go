package main

import (
	"flag"
	"fmt"
	"zero-service/app/xfusionmock/cron"
	"zero-service/app/xfusionmock/internal/config"
	"zero-service/app/xfusionmock/internal/server"
	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/kafka"
	"zero-service/app/xfusionmock/xfusionmock"
	"zero-service/common/nacosx"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"

	_ "zero-service/common/carbonx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/xfusionmock.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		xfusionmock.RegisterXFusionMockRpcServer(grpcServer, server.NewXFusionMockRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	nacosx.SetUpLogger(nacosx.LoggerConfig{
		AppendToStdout: true,
		Level:          "error",
		LogDir:         "/tmp/nacos/log",
	})
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)

	// kafka
	serviceGroup.Add(kq.MustNewQueue(c.KafkaTestConfig, kafka.NewTest(ctx)))

	// cron
	serviceGroup.Add(cron.NewCronService(ctx))
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
