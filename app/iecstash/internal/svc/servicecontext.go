package svc

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/app/iecstash/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/facade/iecstream/iecstream"
)

type ServiceContext struct {
	Config          config.Config
	IecStreamRpcCli iecstream.IecStreamRpcClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		IecStreamRpcCli: iecstream.NewIecStreamRpcClient(zrpc.MustNewClient(c.IecStreamRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
	}
}
