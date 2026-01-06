package handler

import (
	"context"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

type MqttStreamHandler struct {
	clientID       string
	streamEventCli streamevent.StreamEventClient
	socketPushCli  socketpush.SocketPushClient
	taskRunner     *threading.TaskRunner
}

func NewMqttStreamHandler(clientID string, streamEventCli streamevent.StreamEventClient, socketPushCli socketpush.SocketPushClient) *MqttStreamHandler {
	return &MqttStreamHandler{
		clientID:       clientID,
		streamEventCli: streamEventCli,
		socketPushCli:  socketPushCli,
		taskRunner:     threading.NewTaskRunner(16),
	}
}

func (h *MqttStreamHandler) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	if h.streamEventCli != nil {
		h.taskRunner.Schedule(func() {
			msgId, _ := tool.SimpleUUID()
			sendTime := carbon.Now().ToDateTimeMicroString()
			startTime := timex.Now()
			duration := timex.Since(startTime)
			_, err := h.streamEventCli.ReceiveMQTTMessage(ctx, &streamevent.ReceiveMQTTMessageReq{
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
			logx.WithContext(ctx).WithDuration(duration).Infof("push mqtt ReceiveMQTTMessage, msgId: %s, topic: %s, topicTemplate: %s, time: %s - %s", msgId, topic, topicTemplate, sendTime, invokeflg)
		})
	}
	if h.socketPushCli != nil {
		h.taskRunner.Schedule(func() {
			reqId, _ := tool.SimpleUUID()
			sendTime := carbon.Now().ToDateTimeMicroString()
			startTime := timex.Now()
			duration := timex.Since(startTime)
			_, err := h.socketPushCli.BroadcastRoom(ctx, &socketpush.BroadcastRoomReq{
				ReqId:   reqId,
				Room:    topicTemplate,
				Event:   "mqtt",
				Payload: string(payload),
			})
			var invokeflg = "success"
			if err != nil {
				invokeflg = "fail"
			}
			logx.WithContext(ctx).WithDuration(duration).Infof("push mqtt BroadcastRoom, reqId: %s, room: %s, event: %s, time: %s - %s", reqId, topicTemplate, "mqtt", sendTime, invokeflg)
		})
	}
	return nil
}
