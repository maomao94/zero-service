package svc

import (
	"zero-service/app/oryxgtw/internal/config"
	"zero-service/app/oryxgtw/internal/middleware"
	"zero-service/app/oryxserver/oryxserver"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	OryxHookAuth rest.Middleware
	// OryxServerClient oryxserver 客户端（record 生命周期落库）
	OryxServerClient oryxserver.OryxServerClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:           c,
		OryxHookAuth:     middleware.NewOryxHookAuthMiddleware(c),
		OryxServerClient: oryxserver.NewOryxServerClient(zrpc.MustNewClient(c.OryxServerConf).Conn()),
	}
}
