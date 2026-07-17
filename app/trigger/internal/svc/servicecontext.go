package svc

import (
	"math"
	"time"
	"zero-service/app/trigger/internal/config"
	"zero-service/app/trigger/model/gormmodel"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/asynqx"
	"zero-service/common/gormx"
	"zero-service/common/netx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"

	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mathx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpc"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

const (
	expiryDeviation = 0.05
	// connExpiryMinutes 连接缓存过期时间（分钟）
	connExpiryMinutes = 30
)

type ServiceContext struct {
	Config         config.Config
	Validate       *validator.Validate
	AsynqClient    *asynq.Client
	AsynqInspector *asynq.Inspector
	AsynqServer    *asynq.Server
	Scheduler      *asynq.Scheduler
	Httpc          httpc.Service
	NetClient      *netx.Client
	ConnMap        *collection.Cache // 使用带过期清理的缓存
	DB             *gormx.DB
	UnstableExpiry mathx.Unstable
	Redis          *redis.Redis
	StreamEventCli streamevent.StreamEventClient
	IdUtil         *tool.IdUtil
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	redisDb := redis.MustNewRedis(c.Redis.RedisConf)
	// 创建数据库连接
	db := gormx.MustOpenWithConf(c.DB)

	if c.Mode == service.DevMode || c.Mode == service.TestMode {
		db.MustAutoMigrate(
			&gormmodel.Plan{},
			&gormmodel.PlanBatch{},
			&gormmodel.PlanExecItem{},
			&gormmodel.PlanExecLog{})
	}

	// 使用带自动过期清理的缓存，30分钟不访问自动移除过期连接
	connCache, err := collection.NewCache(time.Minute*connExpiryMinutes, collection.WithName("conn-cache"))
	if err != nil {
		panic(err)
	}

	httpcSvc := netx.NewHTTPCService("trigger-httpc")
	return &ServiceContext{
		Config:         c,
		Validate:       validator.New(),
		AsynqClient:    asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqInspector: asynqx.NewAsynqInspector(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		AsynqServer:    asynqx.NewAsynqServer(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		Scheduler:      asynqx.NewScheduler(c.Redis.Host, c.Redis.Pass, c.RedisDB),
		Httpc:          httpcSvc,
		NetClient:      netx.NewClient(netx.WithEngine(netx.NewHTTPEngine(httpcSvc))),
		ConnMap:        connCache,
		DB:             db,
		Redis:          redisDb,
		UnstableExpiry: mathx.NewUnstable(expiryDeviation),
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
