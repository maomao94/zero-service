package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/iecagent/internal/iec"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/iec104"

	"zero-service/app/iecagent/iecagent"
	"zero-service/app/iecagent/internal/config"
	"zero-service/app/iecagent/internal/server"
	"zero-service/app/iecagent/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/iecagent.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		iecagent.RegisterIecAgentServer(grpcServer, server.NewIecAgentServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(s)
	iecServer := iec104.NewIecServer(c.IecSetting.Host, c.IecSetting.Port, c.IecSetting.LogMode, iec104.NewServerHandler(iec.NewIecHandler(ctx)))
	serviceGroup.Add(iecServer)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}