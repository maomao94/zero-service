package main

import (
	"flag"
	"fmt"
	"zero-service/common/nacosx"

	"zero-service/app/alarm/alarm"
	"zero-service/app/alarm/internal/config"
	"zero-service/app/alarm/internal/server"
	"zero-service/app/alarm/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/alarm.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		alarm.RegisterAlarmServer(grpcServer, server.NewAlarmServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	nacosx.SetUpLogger(nacosx.LoggerConfig{
		AppendToStdout: true,
		Level:          "error",
		LogDir:         "/tmp/nacos/log",
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
