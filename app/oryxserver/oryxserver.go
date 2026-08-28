package main

import (
	"flag"
	"fmt"

	"zero-service/common/asynqx"
	"zero-service/common/grpcx"
	"zero-service/common/nacosx"
	"zero-service/common/tool"

	"zero-service/app/oryxserver/internal/config"
	"zero-service/app/oryxserver/internal/cron"
	"zero-service/app/oryxserver/internal/server"
	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/internal/task"
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
			"broadcastInstanceId":       ctx.NodeID,
			"relayNodeId":               ctx.NodeID,
		}
		opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, c.ListenOn, sc, cc, nacosx.WithMetadata(m))
		_ = nacosx.RegisterService(opts)
	}
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	// 对齐 trigger 模式：用 serviceGroup 统一管理所有子服务的启动/关闭
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	// 1. gRPC server
	serviceGroup.Add(s)

	// 2. 分布式 relay Asynq worker（独立 DB + 独立队列，与 trigger 完全隔离）
	relayMux := task.Register(ctx)
	relayTaskServer := asynqx.NewTaskServer(ctx.AsynqServer, relayMux)
	serviceGroup.Add(relayTaskServer)

	// 3. 节点上报：每秒写 Redis（SADD + HSET + EXPIRE）
	serviceGroup.Add(cron.NewNodeReporter(ctx))

	// 4. 孤儿 relay 扫描：每 30s 扫描 Sorted Set 索引，发现无 lease 的 relay → 补偿
	serviceGroup.Add(cron.NewRegistryScanner(ctx))

	// 5. WrapUp：停止全部 FFmpeg 进程，让其拥有完整 GracePeriod 预算
	waitRelayStop := proc.AddWrapUpListener(func() {
		ctx.RelayRegistry.StopAll()
		ctx.FFmpegManager.StopAll()
	})
	defer waitRelayStop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
