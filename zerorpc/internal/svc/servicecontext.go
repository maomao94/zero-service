package svc

import (
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpc"
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/zeroalarm/zeroalarm"
	"zero-service/zerorpc/internal/config"
)

type ServiceContext struct {
	Config       config.Config
	AsynqClient  *asynq.Client
	AsynqServer  *asynq.Server
	Scheduler    *asynq.Scheduler
	Httpc        httpc.Service
	RedisClient  *redis.Redis
	ZeroAlarmCli zeroalarm.ZeroalarmClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisClient := redis.MustNewRedis(c.Redis.RedisConf)
	return &ServiceContext{
		Config:       c,
		AsynqClient:  newAsynqClient(c),
		AsynqServer:  newAsynqServer(c),
		Scheduler:    newScheduler(c),
		Httpc:        httpc.NewService("httpc"),
		RedisClient:  redisClient,
		ZeroAlarmCli: zeroalarm.NewZeroalarmClient(zrpc.MustNewClient(c.ZeroAlarmConf).Conn()),
	}
}
