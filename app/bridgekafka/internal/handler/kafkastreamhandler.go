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

type KafkaStreamHandler struct {
	topic          string
	group          string
	streamEventCli streamevent.StreamEventClient
	taskRunner     *threading.TaskRunner
}

func NewKafkaStreamHandler(topic, group string, streamEventCli streamevent.StreamEventClient) *KafkaStreamHandler {
	return &KafkaStreamHandler{
		topic:          topic,
		group:          group,
		streamEventCli: streamEventCli,
		taskRunner:     threading.NewTaskRunner(16),
	}
}

func (h *KafkaStreamHandler) Consume(ctx context.Context, key, value string) error {
	logx.WithContext(ctx).Infof("receive kafka message, topic: %s, key: %s", h.topic, key)

	if h.streamEventCli != nil {
		h.pushToStreamEvent(ctx, key, value)
	}
	return nil
}

func (h *KafkaStreamHandler) pushToStreamEvent(ctx context.Context, key, value string) {
	h.taskRunner.Schedule(func() {
		msgId, _ := tool.SimpleUUID()
		sendTime := carbon.Now().ToDateTimeMicroString()
		startTime := timex.Now()

		_, err := h.streamEventCli.ReceiveKafkaMessage(ctx, &streamevent.ReceiveKafkaMessageReq{
			Messages: []*streamevent.KafkaMessage{{
				MsgId:    msgId,
				Topic:    h.topic,
				Group:    h.group,
				Key:      key,
				Value:    []byte(value),
				SendTime: sendTime,
			}},
		})

		result := "success"
		if err != nil {
			result = "fail"
		}
		logx.WithContext(ctx).WithDuration(startTime).Infof(
			"push kafka to grpc, msgId: %s, topic: %s, key: %s, time: %s - %s",
			msgId, h.topic, key, sendTime, result,
		)
	})
}
