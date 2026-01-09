package svc

import (
	"zero-service/app/trigger/internal/config"
	"zero-service/common/asynqx"
	"zero-service/common/dbx"
	"zero-service/model"

	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/rest/httpc"
)

type ServiceContext struct {
	Config            config.Config
	Validate          *validator.Validate
	AsynqClient       *asynq.Client
	AsynqInspector    *asynq.Inspector
	AsynqServer       *asynq.Server
	Scheduler         *asynq.Scheduler
	Httpc             httpc.Service
	ConnMap           *collection.SafeMap
	PlanModel         model.PlanModel
	PlanExecItemModel model.PlanExecItemModel
	PlanExecLogModel  model.PlanExecLogModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:            c,
		Validate:          validator.New(),
		AsynqClient:       asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqInspector:    asynqx.NewAsynqInspector(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqServer:       asynqx.NewAsynqServer(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		Scheduler:         asynqx.NewScheduler(c.Redis.Host, c.Redis.Pass),
		Httpc:             httpc.NewService("httpc"),
		ConnMap:           collection.NewSafeMap(),
		PlanModel:         model.NewPlanModel(dbx.New(c.DB.DataSource)),
		PlanExecItemModel: model.NewPlanExecItemModel(dbx.New(c.DB.DataSource)),
		PlanExecLogModel:  model.NewPlanExecLogModel(dbx.New(c.DB.DataSource)),
	}
}
