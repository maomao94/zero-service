package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

// generateTopic 根据配置的topic规则和报文值生成最终的topic
func generateTopic(topicPattern string, data *types.MsgBody) string {
	// 替换固定字段占位符
	topic := strings.ReplaceAll(topicPattern, "{typeId}", fmt.Sprintf("%d", data.TypeId))
	topic = strings.ReplaceAll(topic, "{host}", data.Host)
	topic = strings.ReplaceAll(topic, "{port}", fmt.Sprintf("%d", data.Port))
	topic = strings.ReplaceAll(topic, "{coa}", fmt.Sprintf("%d", data.Coa))
	topic = strings.ReplaceAll(topic, "{dataType}", fmt.Sprintf("%d", data.DataType))
	topic = strings.ReplaceAll(topic, "{asdu}", data.Asdu)
	topic = strings.ReplaceAll(topic, "{ioa}", strconv.Itoa(int(data.Body.GetIoa())))

	// 替换元数据占位符
	if data.MetaData != nil {
		// 使用正则表达式匹配所有{key}格式的占位符
		placeholderRegex := regexp.MustCompile(`{([^}]+)}`)
		topic = placeholderRegex.ReplaceAllStringFunc(topic, func(placeholder string) string {
			// 提取占位符中的键名（去掉{}）
			key := placeholder[1 : len(placeholder)-1]
			// 从元数据中获取对应的值
			if val, ok := data.MetaData[key]; ok {
				// 将值转换为字符串
				return fmt.Sprintf("%v", val)
			}
			// 如果元数据中没有对应键，则保留原占位符
			return placeholder
		})
	}

	return topic
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

		topics := svc.Config.MqttConfig.Topic
		if len(topics) == 0 {
			topics = []string{"iec/asdu"}
		}

		for _, topicPattern := range topics {
			pushTopicCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			topic := generateTopic(topicPattern, data)
			logx.WithContext(ctx).Debugf("pushing asdu to mqtt topic: %s, msgId: %s", topic, data.MsgId)
			err = svc.MqttClient.Publish(pushTopicCtx, topic, byteData)
			if err != nil {
				logx.WithContext(pushTopicCtx).Errorf("failed to push asdu to mqtt topic: %s, msgId: %s", topic, data.MsgId)
				continue
			}
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
