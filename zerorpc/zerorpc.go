package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/common/tool"
	"zero-service/zerorpc/internal/task"

	"zero-service/zerorpc/internal/config"
	"zero-service/zerorpc/internal/server"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	_ "zero-service/common/carbonx"
)

var configFile = flag.String("f", "etc/zerorpc.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		zerorpc.RegisterZerorpcServer(grpcServer, server.NewZerorpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(s)
	mux := task.NewCronJob(ctx).Register()
	taskServer := svc.NewTaskServer(ctx.AsynqServer, mux)
	serviceGroup.Add(taskServer)
	scheduler := svc.NewSchedulerServer(ctx.Scheduler)
	scheduler.Register()
	serviceGroup.Add(scheduler)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
