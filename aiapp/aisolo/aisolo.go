package main

import (
	"flag"
	"fmt"
	"os"
	interceptor "zero-service/common/Interceptor/rpcserver"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/config"
	"zero-service/aiapp/aisolo/internal/server"
	"zero-service/aiapp/aisolo/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/aisolo.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	if err := c.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}
	if apiKey := os.Getenv("AISOLO_MODEL_API_KEY"); apiKey != "" {
		c.Model.APIKey = apiKey
	}
	if c.Model.APIKey == "" {
		fmt.Println("model api key is empty, set Model.APIKey or AISOLO_MODEL_API_KEY")
		return
	}
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		aisolo.RegisterAiSoloServer(grpcServer, server.NewAiSoloServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
	s.AddStreamInterceptors(interceptor.StreamLoggerInterceptor)
	defer s.Stop()

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
