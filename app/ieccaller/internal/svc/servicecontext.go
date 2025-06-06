package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-module/carbon/v2"
	"github.com/zeromicro/go-queue/kq"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/iec104/iec104client"
	"zero-service/iec104/types"
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

func (svc ServiceContext) PushASDU(data *types.MsgBody) error {
	key, _ := data.GetKey()
	data.Time = carbon.Now().ToDateTimeMicroString()
	byteData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error %v", err)
	}
	if svc.Config.KafkaASDUConfig.IsPush {
		return svc.KafkaASDUPusher.PushWithKey(context.Background(), key, string(byteData))
	}
	return nil
}
