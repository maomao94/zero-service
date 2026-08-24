package main

import (
	"flag"
	"fmt"

	"zero-service/common/grpcx"
	"zero-service/common/nacosx"
	"zero-service/common/tool"

	"zero-service/app/oryxserver/internal/config"
	"zero-service/app/oryxserver/internal/server"
	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	_ "zero-service/common/carbonx"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/oryxserver.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	proc.SetTimeToForceQuit(c.GracePeriod)

	// Print Go version
	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		oryxserver.RegisterOryxServerServer(grpcServer, server.NewOryxServerServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// 将 RelayManager 清理接入 go-zero 优雅关闭：收到停止信号时停止全部 FFmpeg 进程。
	// waitRelayStop 阻塞直到监听器执行完，避免 main 先行退出导致 ffmpeg 残留为孤儿进程。
	waitRelayStop := proc.AddShutdownListener(ctx.RelayManager.StopAll)
	defer waitRelayStop()
	defer s.Stop()

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
			"deployMode":                c.DeployMode,
			"broadcastTopic":            ctx.BroadcastTopic(),
			"broadcastAckTopic":         ctx.BroadcastAckTopic(),
			"broadcastInstanceId":       ctx.BroadcastInstanceId(),
		}
		opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, c.ListenOn, sc, cc, nacosx.WithMetadata(m))
		_ = nacosx.RegisterService(opts)
	}
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
