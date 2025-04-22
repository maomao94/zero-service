package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"zero-service/app/xfusionmock/internal/config"
)

type ServiceContext struct {
	Config                  config.Config
	KafkaTestPusher         *kq.Pusher
	KafkaPointPusher        *kq.Pusher
	KafkaAlarmPusher        *kq.Pusher
	KafkaEventPusher        *kq.Pusher
	KafkaTerminalBindPusher *kq.Pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:                  c,
		KafkaTestPusher:         kq.NewPusher(c.KafkaTestConfig.Brokers, c.KafkaTestConfig.Topic),
		KafkaPointPusher:        kq.NewPusher(c.KafkaPointConfig.Brokers, c.KafkaPointConfig.Topic),
		KafkaAlarmPusher:        kq.NewPusher(c.KafkaAlarmConfig.Brokers, c.KafkaAlarmConfig.Topic),
		KafkaEventPusher:        kq.NewPusher(c.KafkaEventConfig.Brokers, c.KafkaEventConfig.Topic),
		KafkaTerminalBindPusher: kq.NewPusher(c.KafkaTerminalBind.Brokers, c.KafkaTerminalBind.Topic),
	}
}
