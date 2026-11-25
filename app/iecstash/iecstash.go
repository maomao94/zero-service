package main

import (
	"flag"
	"fmt"
	"zero-service/app/iecstash/kafka"
	"zero-service/common/grpcx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"

	"zero-service/app/iecstash/iecstash"
	"zero-service/app/iecstash/internal/config"
	"zero-service/app/iecstash/internal/server"
	"zero-service/app/iecstash/internal/svc"

	_ "zero-service/common/carbonx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/iecstash.yaml", "the config file")

// GOOS=linux GOARCH=arm64 go build -x -v -ldflags="-s -w" -o app/iecstash iecstash.go
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	proc.SetTimeToForceQuit(c.GracePeriod)

	// Print Go version
	tool.PrintGoVersion()
	zrpc.DontLogClientContentForMethod(streamevent.StreamEvent_PushChunkAsdu_FullMethodName)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		iecstash.RegisterIecStashServer(grpcServer, server.NewIecStashServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// register service to nacos
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)

	// kafka
	if len(c.KafkaASDUConfig.Brokers) > 0 && c.KafkaASDUConfig.Topic != "" {
		fullConf := c.KafkaASDUConfig.ToKqConf(c.ServiceConf)
		serviceGroup.Add(kq.MustNewQueue(fullConf, kafka.NewAsdu(ctx)))
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
