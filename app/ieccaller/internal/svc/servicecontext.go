package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-queue/kq"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/iec104/iec104client"
	"zero-service/iec104/types"
)

type ServiceContext struct {
	Config               config.Config
	ClientManager        *iec104client.ClientManager
	KafkaASDUPusher      *kq.Pusher
	KafkaBroadcastPusher *kq.Pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config:               c,
		ClientManager:        iec104client.NewClientManager(),
		KafkaASDUPusher:      kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.AsduTopic),
		KafkaBroadcastPusher: kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.BroadcastTopic),
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
	if svc.Config.KafkaConfig.IsPush {
		return svc.KafkaASDUPusher.PushWithKey(context.Background(), key, string(byteData))
	}
	return nil
}

func (svc ServiceContext) PushPbBroadcast(method string, in any) error {
	if svc.IsBroadcast() {
		pbData, err := json.Marshal(in)
		if err != nil {
			return err
		}
		data := &types.BroadcastBody{
			Method: method,
			Body:   string(pbData),
		}
		err = svc.PushBroadcast(data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (svc ServiceContext) PushBroadcast(data *types.BroadcastBody) error {
	if svc.IsBroadcast() {
		byteData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("json marshal error %v", err)
		}
		return svc.KafkaBroadcastPusher.Push(context.Background(), string(byteData))
	}
	return nil
}

func (svc ServiceContext) IsBroadcast() bool {
	return svc.Config.DeployMode == "cluster"
}
