package handler

import (
	"context"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/timex"
)

type MqttStreamHandler struct {
	clientID string
	cli      streamevent.StreamEventClient
}

func NewMqttStreamHandler(clientID string, cli streamevent.StreamEventClient) *MqttStreamHandler {
	return &MqttStreamHandler{
		clientID: clientID,
		cli:      cli,
	}
}

func (h *MqttStreamHandler) Consume(ctx context.Context, topic string, payload []byte) error {
	msgId, _ := tool.SimpleUUID()
	time := carbon.Now().ToDateTimeMicroString()
	startTime := timex.Now()
	duration := timex.Since(startTime)
	_, err := h.cli.ReceiveMQTTMessage(ctx, &streamevent.ReceiveMQTTMessageReq{
		Messages: []*streamevent.MqttMessage{
			{
				SessionId: h.clientID,
				MsgId:     msgId,
				Topic:     topic,
				Payload:   payload,
				SendTime:  time,
			},
		},
	})
	logx.WithContext(ctx).WithDuration(duration).Debugf("consume mqtt message, msgId: %s, topic: %s, time: %s", msgId, topic, time)
	if err != nil {
		return err
	}
	return nil
}
