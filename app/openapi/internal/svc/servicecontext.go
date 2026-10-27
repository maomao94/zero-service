package svc

import (
	"errors"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/openapi/internal/config"
	"zero-service/app/openapi/internal/dispatch"
	"zero-service/app/podengine/podengine"
	"zero-service/common/grpcx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	Registry *dispatch.Registry
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.Mock && c.Mode != service.DevMode && c.Mode != service.TestMode {
		logx.Must(errors.New("openapi Mock 仅允许在 dev/test 模式启用"))
	}

	if c.Mock {
		logx.Info("openapi running in mock mode")
		return &ServiceContext{
			Config:   c,
			Registry: dispatch.NewMockRegistry(),
		}
	}

	client := podengine.NewPodEngineClient(zrpc.MustNewClient(c.PodEngineConf,
		zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn())
	modbusClient := bridgemodbus.NewBridgeModbusClient(zrpc.MustNewClient(c.BridgeModbusConf,
		zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn())
	return &ServiceContext{
		Config:   c,
		Registry: dispatch.NewRegistry(client, modbusClient),
	}
}
