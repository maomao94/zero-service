package svc

import (
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpc"
	"time"
	"zero-service/common/alarmx"
	"zero-service/zeroalarm/internal/config"
)

type ServiceContext struct {
	Config      config.Config
	Httpc       httpc.Service
	RedisClient *redis.Redis
	AlarmX      *alarmx.AlarmX
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisClient := redis.MustNewRedis(c.Redis.RedisConf)
	return &ServiceContext{
		Config:      c,
		Httpc:       httpc.NewService("httpc"),
		RedisClient: redisClient,
		AlarmX: alarmx.NewAlarmX(
			lark.NewClient(c.Alarmx.AppId, c.Alarmx.AppSecret,
				lark.WithReqTimeout(3*time.Second),
				lark.WithHttpClient(alarmx.NewAlarmxHttpClient(httpc.NewService("alarm")))),
			redisClient),
	}
}
