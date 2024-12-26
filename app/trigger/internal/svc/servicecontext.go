package svc

import (
	"github.com/hibiken/asynq"
	"zero-service/app/trigger/internal/config"
	"zero-service/common/asynqx"
)

type ServiceContext struct {
	Config      config.Config
	AsynqClient *asynq.Client
	AsynqServer *asynq.Server
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:      c,
		AsynqClient: asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass),
	}
}
