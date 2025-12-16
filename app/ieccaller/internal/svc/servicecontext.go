package svc

import (
	"context"
	"encoding/json"
	"errors"
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
		Config:        c,
		ClientManager: iec104client.NewClientManager(),
	}
	if len(c.KafkaConfig.Brokers) > 0 {
		svcCtx.KafkaASDUPusher = kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.Topic)
		svcCtx.KafkaBroadcastPusher = kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.BroadcastTopic)
	}
	if len(c.MqttConfig.Broker) > 0 {
		svcCtx.MqttClient = mqttx.MustNewClient(c.MqttConfig.MqttConfig)
	}

	return svcCtx
}

var placeholderRegex = regexp.MustCompile(`{([^}]+)}`)

// generateTopic 根据配置的topic规则和报文值生成最终的topic
func generateTopic(topicPattern string, data *types.MsgBody) (string, error) {
	if data == nil {
		return "", errors.New("msg body is nil")
	}

	replacements := map[string]string{
		"typeId":   fmt.Sprintf("%d", data.TypeId),
		"host":     data.Host,
		"port":     fmt.Sprintf("%d", data.Port),
		"coa":      fmt.Sprintf("%d", data.Coa),
		"dataType": fmt.Sprintf("%d", data.DataType),
		"asdu":     data.Asdu,
		"ioa":      strconv.Itoa(int(data.Body.GetIoa())),
	}
	topic := topicPattern
	for k, v := range replacements {
		if v == "" {
			return "", fmt.Errorf("topic field {%s} is empty", k)
		}
		topic = strings.ReplaceAll(topic, "{"+k+"}", v)
	}

	missingKeys := make([]string, 0)

	topic = placeholderRegex.ReplaceAllStringFunc(topic, func(placeholder string) string {
		key := placeholder[1 : len(placeholder)-1]

		if strings.HasPrefix(key, "metaData.") {
			metaKey := key[len("metaData."):]
			if data.MetaData == nil {
				missingKeys = append(missingKeys, key)
				return placeholder
			}
			if val, ok := data.MetaData[metaKey]; ok {
				return fmt.Sprintf("%v", val)
			}
			missingKeys = append(missingKeys, key)
			return placeholder
		}

		// Handle regular keys for backward compatibility
		if data.MetaData == nil {
			missingKeys = append(missingKeys, key)
			return placeholder
		}

		if val, ok := data.MetaData[key]; ok {
			return fmt.Sprintf("%v", val)
		}

		missingKeys = append(missingKeys, key)
		return placeholder
	})

	if len(missingKeys) > 0 {
		return "", fmt.Errorf("missing topic fields: %v", missingKeys)
	}

	if placeholderRegex.MatchString(topic) {
		return "", fmt.Errorf("unresolved placeholders in topic: %s", topic)
	}

	if strings.Contains(topic, "+") || strings.Contains(topic, "#") {
		return "", fmt.Errorf("invalid mqtt topic: %s", topic)
	}
	return topic, nil
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
			return nil
		}

		for _, topicPattern := range topics {
			pushTopicCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			topic, genErr := generateTopic(topicPattern, data)
			if genErr != nil {
				logx.WithContext(ctx).Errorf("failed to generate topic: %v", genErr)
				continue
			}
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
