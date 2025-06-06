package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/trigger/internal/task"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/common/asynqx"

	"zero-service/app/trigger/internal/config"
	"zero-service/app/trigger/internal/server"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	_ "zero-service/common/carbonx"
)

var configFile = flag.String("f", "etc/trigger.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		trigger.RegisterTriggerRpcServer(grpcServer, server.NewTriggerRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)
	mux := task.NewCronJob(ctx).Register()
	taskServer := asynqx.NewTaskServer(ctx.AsynqServer, mux)
	serviceGroup.Add(taskServer)
	scheduler := asynqx.NewSchedulerServer(ctx.Scheduler)
	//scheduler.RegisterTest()
	serviceGroup.Add(scheduler)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
