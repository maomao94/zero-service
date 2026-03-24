package svc

import (
	"zero-service/aiapp/mcpserver/internal/config"
	"zero-service/app/bridgemodbus/bridgemodbus"
	interceptor "zero-service/common/Interceptor/rpcclient"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config          config.Config
	BridgeModbusCli bridgemodbus.BridgeModbusClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		BridgeModbusCli: bridgemodbus.NewBridgeModbusClient(
			zrpc.MustNewClient(c.BridgeModbusRpcConf,
				zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			).Conn()),
	}
}
