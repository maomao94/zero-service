package main

import (
	"flag"
	"zero-service/app/ieccaller/cron"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/app/ieccaller/internal/iec"
	"zero-service/app/ieccaller/internal/server"
	"zero-service/app/ieccaller/internal/svc"
	_ "zero-service/common/carbonx"
	"zero-service/common/grpcx"
	"zero-service/common/iec104/client"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/ieccaller.yaml", "the config file")

// GOARCH=amd64 GOOS=linux GOOS=linux go build -o app
// GOARCH=arm GOOS=linux go build -o app
// go build -o app
// GOOS=linux GOARCH=arm64 go build -x -v -ldflags="-s -w" -o app/iecaller ieccaller.go
// docker build -t {name}:{tag} .
// docker buildx build --pull=false --platform linux/arm64 -t {name}:{tag} .
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	proc.SetTimeToForceQuit(c.GracePeriod)
	ctx := svc.NewServiceContext(c)

	// Print Go version
	tool.PrintGoVersion()
	zrpc.DontLogClientContentForMethod(streamevent.StreamEvent_PushChunkAsdu_FullMethodName)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		ieccaller.RegisterIecCallerServer(grpcServer, server.NewIecCallerServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// register service to nacos
	c.NacosConfig = c.NacosConfig.WithMetadata(map[string]string{
		"deployMode":          c.DeployMode,
		"broadcastTopic":      ctx.BroadcastTopic(),
		"broadcastAckTopic":   ctx.BroadcastAckTopic(),
		"broadcastInstanceId": ctx.BroadcastInstanceId(),
	})
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)

	for _, cf := range c.IecServerConfig {
		iecCli := client.MustNewClient(cf.ClientConfig, client.WithASDUHandler(iec.NewClientCall(ctx, cf)))
		serviceGroup.Add(iecCli)
		ctx.ClientManager.PublishRegister(iecCli)
	}

	// cron
	serviceGroup.Add(cron.NewCronService(ctx))

	logx.Infof("Deploy mode: %s\n", c.DeployMode)
	logx.Infof("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
