package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/iec104/iec104client"
)

type ServiceContext struct {
	Config          config.Config
	ClientManager   *iec104client.ClientManager
	KafkaASDUPusher *kq.Pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config:          c,
		ClientManager:   iec104client.NewClientManager(),
		KafkaASDUPusher: kq.NewPusher(c.KafkaASDUConfig.Brokers, c.KafkaASDUConfig.Topic),
	}
	return svcCtx
}
