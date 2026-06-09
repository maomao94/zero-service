package main

import (
	"flag"
	"fmt"

	"zero-service/app/bridgekafka/bridgekafka"
	"zero-service/app/bridgekafka/internal/config"
	"zero-service/app/bridgekafka/internal/handler"
	"zero-service/app/bridgekafka/internal/server"
	"zero-service/app/bridgekafka/internal/svc"
	interceptor "zero-service/common/Interceptor/rpcserver"
	_ "zero-service/common/carbonx"
	"zero-service/common/nacosx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/bridgekafka.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()

	ctx := svc.NewServiceContext(c)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	for i := range c.KafkaConsumeConfig {
		kc := c.KafkaConsumeConfig[i]
		if len(kc.Brokers) == 0 || kc.Topic == "" {
			continue
		}
		fullConf := kc.ToKqConf(c.ServiceConf)
		h := handler.NewKafkaStreamHandler(kc.Topic, kc.Group, ctx.StreamEventCli)
		serviceGroup.Add(kq.MustNewQueue(fullConf, h))
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		bridgekafka.RegisterBridgeKafkaServer(grpcServer, server.NewBridgeKafkaServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

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

	serviceGroup.Add(s)
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	serviceGroup.Start()
}
