package main

import (
	"flag"
	"fmt"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/config"
	"zero-service/app/file/internal/server"
	"zero-service/app/file/internal/svc"
	_ "zero-service/common/carbonx"
	"zero-service/common/ffmpegx"
	"zero-service/common/grpcx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/file.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	// Print ffmpeg/ffprobe versions
	ffmpegx.PrintVersions()

	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		file.RegisterFileRpcServer(grpcServer, server.NewFileRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// register service to nacos
	c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)
	s.AddOptions(grpc.ChainUnaryInterceptor(grpcx.ServerFinalizeInterceptor))
	s.AddUnaryInterceptors(grpcx.LoggerInterceptor)
	s.AddOptions(grpc.ChainStreamInterceptor(grpcx.ServerStreamFinalizeInterceptor))
	s.AddStreamInterceptors(grpcx.StreamLoggerInterceptor)
	defer s.Stop()
	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
