package handler

import (
	"context"
	"zero-service/facade/mqttstream/mqttstream"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/random"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/timex"
)

type MqttStreamHandler struct {
	clientID string
	cli      mqttstream.MqttStreamClient
}

func NewMqttStreamHandler(clientID string, cli mqttstream.MqttStreamClient) *MqttStreamHandler {
	return &MqttStreamHandler{
		clientID: clientID,
		cli:      cli,
	}
}

func (h *MqttStreamHandler) Consume(ctx context.Context, topic string, payload []byte) error {
	msgId, _ := random.UUIdV4()
	time := carbon.Now().ToDateTimeMicroString()
	startTime := timex.Now()
	duration := timex.Since(startTime)
	_, err := h.cli.ReceiveMessage(ctx, &mqttstream.ReceiveMessageReq{
		Messages: []*mqttstream.MqttMessage{
			{
				SessionId: h.clientID,
				MsgId:     msgId,
				Topic:     topic,
				Payload:   payload,
				SendTime:  time,
			},
		},
	})
	logx.WithContext(ctx).WithDuration(duration).Infof("push mqtt message, topic: %s, time: %s", topic, time)
	if err != nil {
		return err
	}
	return nil
}
