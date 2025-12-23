package main

import (
	"flag"
	"fmt"
	"zero-service/app/ieccaller/cron"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/app/ieccaller/internal/iec"
	"zero-service/app/ieccaller/internal/server"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/app/ieccaller/kafka"
	interceptor "zero-service/common/Interceptor/rpcserver"
	_ "zero-service/common/carbonx"
	"zero-service/common/iec104/iec104client"
	"zero-service/common/nacosx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-queue/kq"
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
			"broadcastTopic":            c.KafkaConfig.BroadcastTopic,
			"broadcastGroupId":          c.KafkaConfig.BroadcastGroupId,
			"isPush":                    convertor.ToString(c.KafkaConfig.IsPush),
		}
		opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, c.ListenOn, sc, cc, nacosx.WithMetadata(m))
		_ = nacosx.RegisterService(opts)
	}
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup.Add(s)

	for _, cf := range c.IecServerConfig {
		serviceGroup.Add(client.MustNewIecServerClient(cf, iec.NewClientCall(ctx, cf.Host, cf.Port, cf.MetaData, cf.TaskConcurrency), ctx.ClientManager))
	}

	// cron
	serviceGroup.Add(cron.NewCronService(ctx))

	// kafka 广播队列
	if len(c.KafkaConfig.Brokers) > 0 {
		kqConf := kq.KqConf{
			ServiceConf: service.ServiceConf{
				Name: "broadcast-" + c.KafkaConfig.BroadcastGroupId,
			},
			Brokers:       c.KafkaConfig.Brokers,
			Group:         c.KafkaConfig.BroadcastGroupId,
			Topic:         c.KafkaConfig.BroadcastTopic,
			Offset:        "last",
			Conns:         3,
			Consumers:     3,
			Processors:    18,
			MinBytes:      10240,
			MaxBytes:      10485760,
			ForceCommit:   true,
			CommitInOrder: false,
		}
		serviceGroup.Add(kq.MustNewQueue(kqConf, kafka.NewBroadcast(ctx)))
	}

	fmt.Printf("DeployMode: %s\n", c.DeployMode)
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
