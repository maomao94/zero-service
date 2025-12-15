package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/common/iec104/iec104client"
	"zero-service/common/iec104/types"
	"zero-service/common/mqttx"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config               config.Config
	ClientManager        *iec104client.ClientManager
	KafkaASDUPusher      *kq.Pusher
	KafkaBroadcastPusher *kq.Pusher
	MqttClient           *mqttx.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	svcCtx := &ServiceContext{
		Config:               c,
		ClientManager:        iec104client.NewClientManager(),
		KafkaASDUPusher:      kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.Topic),
		KafkaBroadcastPusher: kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.BroadcastTopic),
	}
	// 初始化MQTT客户端
	if len(c.MqttConfig.Broker) > 0 {
		svcCtx.MqttClient = mqttx.MustNewClient(c.MqttConfig.MqttConfig)
	}

	return svcCtx
}

func (svc ServiceContext) PushASDU(ctx context.Context, data *types.MsgBody) error {
	key, _ := data.GetKey()
	data.Time = carbon.Now().ToDateTimeMicroString()
	byteData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error %v", err)
	}

	// Kafka推送
	if svc.Config.KafkaConfig.IsPush {
		if svc.KafkaASDUPusher == nil {
			logx.WithContext(ctx).Errorf("kafka asdu pusher is nil, msgId: %s", data.MsgId)
			return fmt.Errorf("kafka asdu pusher is nil")
		}
		pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err = svc.KafkaASDUPusher.PushWithKey(pushCtx, key, string(byteData))
		if err != nil {
			logx.WithContext(ctx).Errorf("failed to push asdu to kafka: %v", err)
			return err
		}
	}

	// MQTT推送
	if svc.Config.MqttConfig.IsPush {
		if svc.MqttClient == nil {
			logx.WithContext(ctx).Errorf("mqtt client is nil, msgId: %s", data.MsgId)
			return fmt.Errorf("mqtt client is nil")
		}
		pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := svc.MqttClient.Publish(pushCtx, svc.Config.MqttConfig.Topic, byteData); err != nil {
			logx.WithContext(ctx).Errorf("failed to push asdu to mqtt: %v", err)
			return err
		}
	}

	return nil
}

func (svc ServiceContext) PushPbBroadcast(ctx context.Context, method string, in any) error {
	if svc.IsBroadcast() {
		pbData, err := json.Marshal(in)
		if err != nil {
			return err
		}
		data := &types.BroadcastBody{
			Method: method,
			Body:   string(pbData),
		}
		err = svc.PushBroadcast(ctx, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (svc ServiceContext) PushBroadcast(ctx context.Context, data *types.BroadcastBody) error {
	if !svc.IsBroadcast() {
		return nil
	}

	data.BroadcastGroupId = svc.Config.KafkaConfig.BroadcastGroupId
	byteData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error %v", err)
	}

	// Kafka推送
	if svc.KafkaBroadcastPusher != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := svc.KafkaBroadcastPusher.Push(ctx, string(byteData)); err != nil {
			logx.WithContext(ctx).Errorf("failed to push broadcast to kafka: %v", err)
			return err
		}
	}
	return nil
}

func (svc ServiceContext) IsBroadcast() bool {
	return svc.Config.DeployMode == "cluster"
}

// Close 关闭所有资源
func (svc ServiceContext) Close() {
	if svc.KafkaASDUPusher != nil {
		logx.Infof("closing kafka asdu pusher")
		if err := svc.KafkaASDUPusher.Close(); err != nil {
			logx.Errorf("failed to close kafka asdu pusher: %v", err)
		}
	}
	if svc.KafkaBroadcastPusher != nil {
		logx.Infof("closing kafka broadcast pusher")
		if err := svc.KafkaBroadcastPusher.Close(); err != nil {
			logx.Errorf("failed to close kafka broadcast pusher: %v", err)
		}
	}
	if svc.MqttClient != nil {
		logx.Infof("closing mqtt client")
		svc.MqttClient.Close()
	}
	logx.Infof("service context closed")
}
