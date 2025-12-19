package main

import (
	"flag"
	"fmt"
	"zero-service/app/iecagent/iecagent"
	"zero-service/app/iecagent/internal/config"
	"zero-service/app/iecagent/internal/iec"
	"zero-service/app/iecagent/internal/server"
	"zero-service/app/iecagent/internal/svc"
	interceptor "zero-service/common/Interceptor/rpcserver"
	iec104server2 "zero-service/common/iec104/iec104server"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/iecagent.yaml", "the config file")

// GOOS=windows GOARCH=amd64 go build -o app.exe
// GOARCH=amd64 GOOS=linux GOOS=linux go build -o app
// GOARCH=arm GOOS=linux go build -o app
// go build -o app
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

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
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)
	iecServer := iec104server2.NewIecServer(c.IecSetting.Host, c.IecSetting.Port, c.IecSetting.LogMode, iec104server2.NewServerHandler(iec.NewIecHandler(ctx)))
	serviceGroup.Add(iecServer)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
