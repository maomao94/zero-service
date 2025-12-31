package handler

import (
	"context"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
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

func (h *MqttStreamHandler) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	threading.GoSafe(func() {
		msgId, _ := tool.SimpleUUID()
		sendTime := carbon.Now().ToDateTimeMicroString()
		startTime := timex.Now()
		duration := timex.Since(startTime)
		_, err := h.cli.ReceiveMQTTMessage(ctx, &streamevent.ReceiveMQTTMessageReq{
			Messages: []*streamevent.MqttMessage{
				{
					SessionId:     h.clientID,
					MsgId:         msgId,
					Topic:         topic,
					Payload:       payload,
					SendTime:      sendTime,
					TopicTemplate: topicTemplate,
				},
			},
		})
		var invokeflg = "success"
		if err != nil {
			invokeflg = "fail"
		}
		logx.WithContext(ctx).WithDuration(duration).Infof("push mqtt eventMessage, msgId: %s, topic: %s, topicTemplate: %s, time: %s - %s", msgId, topic, topicTemplate, sendTime, invokeflg)
	})
	return nil
}
