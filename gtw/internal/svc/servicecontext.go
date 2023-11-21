package svc

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/gtw/internal/config"
	"zero-service/zerorpc/zerorpc"
)

type ServiceContext struct {
	Config     config.Config
	ZeroRpcCli zerorpc.ZerorpcClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:     c,
		ZeroRpcCli: zerorpc.NewZerorpcClient(zrpc.MustNewClient(c.ZeroRpcConf).Conn()),
	}
}
