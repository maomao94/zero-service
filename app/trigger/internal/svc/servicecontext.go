package svc

import (
	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/rest/httpc"
	"zero-service/app/trigger/internal/config"
	"zero-service/common/asynqx"
)

type ServiceContext struct {
	Config      config.Config
	Validate    *validator.Validate
	AsynqClient *asynq.Client
	AsynqServer *asynq.Server
	Scheduler   *asynq.Scheduler
	Httpc       httpc.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:      c,
		Validate:    validator.New(),
		AsynqClient: asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass),
		AsynqServer: asynqx.NewAsynqServer(c.Redis.Host, c.Redis.Pass),
		Scheduler:   asynqx.NewScheduler(c.Redis.Host, c.Redis.Pass),
		Httpc:       httpc.NewService("httpc"),
	}
}
