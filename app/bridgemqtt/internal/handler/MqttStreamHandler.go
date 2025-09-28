package handler

import (
	"context"
	"zero-service/facade/mqttstream/mqttstream"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/random"
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
	if err != nil {
		return err
	}
	return nil
}
