package main

import (
	"flag"
	"fmt"
	"zero-service/app/trigger/cron"
	"zero-service/app/trigger/internal/task"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/common/asynqx"
	"zero-service/common/tool"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"

	"zero-service/app/trigger/internal/config"
	"zero-service/app/trigger/internal/server"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	_ "zero-service/common/carbonx"
	"zero-service/common/nacosx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/trigger.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	proc.SetTimeToForceQuit(c.GracePeriod)

	// Print Go version
	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		trigger.RegisterTriggerRpcServer(grpcServer, server.NewTriggerRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// register service to nacos
	if c.NacosConfig.IsRegister {
		sc := []constant.ServerConfig{
			*constant.NewServerConfig(c.NacosConfig.Host, c.NacosConfig.Port),
		}
		cc := &constant.ClientConfig{
			NamespaceId:         c.NacosConfig.NamespaceId,
			Username:            c.NacosConfig.Username,
			Password:            c.NacosConfig.PassWord,
			TimeoutMs:           5000,
			NotLoadCacheAtStart: true,
		}
		m := map[string]string{
			"gRPC_port":                 strutil.After(c.RpcServerConf.ListenOn, ":"),
			"preserved.register.source": "go-zero",
		}
		opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, c.ListenOn, sc, cc, nacosx.WithMetadata(m))
		_ = nacosx.RegisterService(opts)
	}
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
	// cron
	serviceGroup.Add(cron.NewCronService(ctx))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
