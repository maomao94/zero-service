package main

import (
	"flag"
	"fmt"

	"zero-service/app/live/internal/config"
	"zero-service/app/live/internal/server"
	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	_ "zero-service/common/carbonx"
	"zero-service/common/grpcx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/live.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	proc.SetTimeToForceQuit(c.GracePeriod)

	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		live.RegisterLiveRpcServer(grpcServer, server.NewLiveRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

	// 可选：注册服务到 nacos
	c.NacosConfig = c.NacosConfig.WithMetadata(map[string]string{"deployMode": c.DeployMode})
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	defer s.Stop()
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
