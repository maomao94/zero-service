package svc

import (
	"zero-service/app/trigger/internal/config"
	"zero-service/common/asynqx"
	"zero-service/common/dbx"
	"zero-service/model"

	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}

	// 解析数据库类型
	dbType := dbx.ParseDatabaseType(c.DB.DataSource)

	// 创建数据库连接
	dbConn := dbx.New(c.DB.DataSource)

	return &ServiceContext{
		Config:            c,
		Validate:          validator.New(),
		AsynqClient:       asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqInspector:    asynqx.NewAsynqInspector(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqServer:       asynqx.NewAsynqServer(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		Scheduler:         asynqx.NewScheduler(c.Redis.Host, c.Redis.Pass),
		Httpc:             httpc.NewService("httpc"),
		ConnMap:           collection.NewSafeMap(),
		PlanModel:         model.NewPlanModel(dbConn),
		PlanExecItemModel: model.NewPlanExecItemModelWithDBType(dbConn, dbType),
		PlanExecLogModel:  model.NewPlanExecLogModel(dbConn),
	}
}
