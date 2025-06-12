package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/gateway"
	"zero-service/gtw/internal/config"
	"zero-service/gtw/internal/handler"
	"zero-service/gtw/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	_ "zero-service/common/nacosx"
)

var configFile = flag.String("f", "etc/gtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// grpc-gateway
	server := gateway.MustNewServer(c.GatewayConf)
	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server.Server, ctx)
	logx.AddGlobalFields(logx.Field("app", c.Name))
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
