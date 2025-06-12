package main

import (
	"flag"
	"fmt"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/ieccaller/cron"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/app/ieccaller/internal/iec"
	"zero-service/app/ieccaller/internal/server"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/app/ieccaller/kafka"
	interceptor "zero-service/common/Interceptor/rpcserver"
	"zero-service/common/nacosx"
	"zero-service/iec104/iec104client"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	_ "zero-service/common/carbonx"
)

var configFile = flag.String("f", "etc/ieccaller.yaml", "the config file")

// GOARCH=amd64 GOOS=linux GOOS=linux go build -o app
// GOARCH=arm GOOS=linux go build -o app
// go build -o app
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

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
			LogDir:              "/tmp/nacos/log",
			CacheDir:            "/tmp/nacos/cache",
			LogLevel:            "debug",
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

	for _, cf := range c.IecServerConfig {
		serviceGroup.Add(iec104client.MustNewIecServerClient(cf, iec.NewClientCall(ctx, cf.Host, cf.Port, cf.MetaData), ctx.ClientManager))
	}

	// cron
	serviceGroup.Add(cron.NewCronService(ctx))

	if c.DeployMode == "cluster" {
		// kafka 广播队列
		kqConf := kq.KqConf{
			ServiceConf: service.ServiceConf{
				Name: "broadcast-" + c.KafkaConfig.BroadcastGroupId,
			},
			Brokers:       c.KafkaConfig.Brokers,
			Group:         c.KafkaConfig.BroadcastGroupId,
			Topic:         c.KafkaConfig.BroadcastTopic,
			Offset:        "last",
			Conns:         1,
			Consumers:     3,
			Processors:    6,
			MinBytes:      10240,
			MaxBytes:      10485760,
			ForceCommit:   true,
			CommitInOrder: false,
		}
		serviceGroup.Add(kq.MustNewQueue(kqConf, kafka.NewBroadcast(ctx)))
	}
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
