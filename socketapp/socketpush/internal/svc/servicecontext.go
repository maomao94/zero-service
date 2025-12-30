package svc

import (
	"zero-service/common/socketio"
	"zero-service/socketapp/socketpush/internal/config"
)

type ServiceContext struct {
	Config          config.Config
	SocketContainer *socketio.SocketContainer
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		SocketContainer: socketio.MustNewPubContainer(c.SocketGtwConf),
	}
}
