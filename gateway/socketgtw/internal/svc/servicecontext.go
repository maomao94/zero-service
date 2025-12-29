package svc

import (
	"zero-service/common/socketio"
	"zero-service/gateway/socketgtw/internal/config"
)

type ServiceContext struct {
	Config       config.Config
	SocketServer *socketio.Server
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		SocketServer: socketio.MustServer(),
	}
}
