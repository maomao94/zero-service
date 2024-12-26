package svc

import (
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/rest/httpc"
	"zero-service/app/trigger/internal/config"
	"zero-service/common/asynqx"
)

type ServiceContext struct {
	Config      config.Config
	AsynqClient *asynq.Client
	AsynqServer *asynq.Server
	Httpc       httpc.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:      c,
		AsynqClient: asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass),
		AsynqServer: asynqx.NewAsynqServer(c.Redis.Host, c.Redis.Pass),
		Httpc:       httpc.NewService("httpc"),
	}
}
