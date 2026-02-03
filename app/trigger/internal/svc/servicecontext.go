package svc

import (
	"math"
	"zero-service/app/trigger/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/asynqx"
	"zero-service/common/dbx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/doug-martin/goqu/v9"
	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/mathx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpc"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

const (
	expiryDeviation = 0.05
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
	PlanBatchModel    model.PlanBatchModel
	PlanExecItemModel model.PlanExecItemModel
	PlanExecLogModel  model.PlanExecLogModel
	Database          *goqu.Database
	UnstableExpiry    mathx.Unstable
	Redis             *redis.Redis
	StreamEventCli    streamevent.StreamEventClient
	IdUtil            *tool.IdUtil
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}
	redisDb := redis.MustNewRedis(c.Redis.RedisConf)
	// 解析数据库类型
	dbType := dbx.ParseDatabaseType(c.DB.DataSource)

	// 创建数据库连接
	dbConn := dbx.New(c.DB.DataSource)
	database := dbx.NewQoqu(c.DB.DataSource)

	return &ServiceContext{
		Config:            c,
		Validate:          validator.New(),
		AsynqClient:       asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqInspector:    asynqx.NewAsynqInspector(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqServer:       asynqx.NewAsynqServer(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		Scheduler:         asynqx.NewScheduler(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		Httpc:             httpc.NewService("httpc"),
		ConnMap:           collection.NewSafeMap(),
		PlanModel:         model.NewPlanModelWithDBType(dbConn, model.DatabaseType(dbType)),
		PlanBatchModel:    model.NewPlanBatchModelWithDBType(dbConn, model.DatabaseType(dbType)),
		PlanExecItemModel: model.NewPlanExecItemModelWithDBType(dbConn, model.DatabaseType(dbType)),
		PlanExecLogModel:  model.NewPlanExecLogModelWithDBType(dbConn, model.DatabaseType(dbType)),
		Database:          database,
		Redis:             redisDb,
		UnstableExpiry:    mathx.NewUnstable(expiryDeviation),
		StreamEventCli: streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			// 添加最大消息配置
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
				//grpc.MaxCallSendMsgSize(50 * 1024 * 1024),   // 发送最大50MB
				//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
			)),
		).Conn()),
		IdUtil: tool.NewIdUtil(redisDb),
	}
}
