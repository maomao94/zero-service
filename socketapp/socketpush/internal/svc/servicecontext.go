package svc

import (
	"zero-service/common/socketiox"
	"zero-service/socketapp/socketpush/internal/config"
)

type ServiceContext struct {
	Config          config.Config
	SocketContainer *socketiox.SocketContainer
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		SocketContainer: socketiox.MustNewPubContainer(c.SocketGtwConf),
	}
}

// Close 释放服务发现等后台资源。
func (s *ServiceContext) Close() error {
	if s.SocketContainer != nil {
		s.SocketContainer.Close()
	}
	return nil
}
