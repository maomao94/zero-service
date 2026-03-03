package handler

import (
	"context"
	"zero-service/common/mqttx"
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
	eventMapping   []mqttx.EventMapping
	defaultEvent   string
}

func NewMqttStreamHandler(clientID string, streamEventCli streamevent.StreamEventClient, socketPushCli socketpush.SocketPushClient, eventMapping []mqttx.EventMapping, defaultEvent string) *MqttStreamHandler {
	return &MqttStreamHandler{
		clientID:       clientID,
		streamEventCli: streamEventCli,
		socketPushCli:  socketPushCli,
		taskRunner:     threading.NewTaskRunner(16),
		eventMapping:   eventMapping,
		defaultEvent:   defaultEvent,
	}
}

func (h *MqttStreamHandler) matchEvent(topicTemplate string) string {
	for _, mapping := range h.eventMapping {
		if topicTemplate == mapping.Match {
			return mapping.Event
		}
	}
	return h.defaultEvent
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
			event := h.matchEvent(topicTemplate)
			_, err := h.socketPushCli.BroadcastRoom(ctx, &socketpush.BroadcastRoomReq{
				ReqId:   reqId,
				Room:    topicTemplate,
				Event:   event,
				Payload: string(payload),
			})
			var invokeflg = "success"
			if err != nil {
				invokeflg = "fail"
			}
			logx.WithContext(ctx).WithDuration(duration).Infof("push mqtt BroadcastRoom, reqId: %s, room: %s, event: %s, time: %s - %s", reqId, topicTemplate, event, sendTime, invokeflg)
		})
	}
	return nil
}
