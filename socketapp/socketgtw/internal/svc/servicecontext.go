package svc

import (
	"zero-service/common/socketiox"
	"zero-service/socketapp/socketgtw/internal/config"
)

type ServiceContext struct {
	Config       config.Config
	SocketServer *socketiox.Server
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		SocketServer: socketiox.MustServer(socketiox.WithContextKeys(c.SocketMetaData)),
	}
}
