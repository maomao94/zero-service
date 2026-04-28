package handler

import (
	"context"
	"time"

	"zero-service/app/bridgemqtt/internal/config"
	"zero-service/common/mqttx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

// MqttStreamHandler MQTT 消息流处理器
type MqttStreamHandler struct {
	clientID       string
	streamEventCli streamevent.StreamEventClient
	socketPushCli  socketpush.SocketPushClient
	taskRunner     *threading.TaskRunner
	eventMapping   []config.EventMapping
	defaultEvent   string
	logManager     *mqttx.TopicLogManager
}

// NewMqttStreamHandler 创建 MQTT 流处理器
func NewMqttStreamHandler(clientID string, streamEventCli streamevent.StreamEventClient, socketPushCli socketpush.SocketPushClient, eventMapping []config.EventMapping, defaultEvent string, logCfg mqttx.TopicLogConfig) *MqttStreamHandler {
	logManager := mqttx.NewTopicLogManager()
	logManager.LoadFromConfig(logCfg)

	return &MqttStreamHandler{
		clientID:       clientID,
		streamEventCli: streamEventCli,
		socketPushCli:  socketPushCli,
		taskRunner:     threading.NewTaskRunner(16),
		eventMapping:   eventMapping,
		defaultEvent:   defaultEvent,
		logManager:     logManager,
	}
}

func (h *MqttStreamHandler) matchEvent(topicTemplate string) string {
	for _, mapping := range h.eventMapping {
		if mapping.TopicTemplate == topicTemplate {
			return mapping.Event
		}
	}
	return h.defaultEvent
}

// Consume 处理 MQTT 消息
func (h *MqttStreamHandler) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	h.logMessage(ctx, topic, topicTemplate, payload)

	if h.streamEventCli != nil {
		h.pushToStreamEvent(ctx, topic, topicTemplate, payload)
	}
	if h.socketPushCli != nil {
		h.pushToSocket(ctx, topic, topicTemplate, payload)
	}
	return nil
}

func (h *MqttStreamHandler) logMessage(ctx context.Context, topic, topicTemplate string, payload []byte) {
	if !h.logManager.ShouldLog(topicTemplate) {
		return
	}

	if h.logManager.ShouldLogPayload(topicTemplate) {
		logx.WithContext(ctx).Infof("receive mqtt message, topic: %s, topicTemplate: %s, payload: %s", topic, topicTemplate, string(payload))
	} else {
		logx.WithContext(ctx).Infof("receive mqtt message, topic: %s, topicTemplate: %s", topic, topicTemplate)
	}
}

func (h *MqttStreamHandler) pushToStreamEvent(ctx context.Context, topic, topicTemplate string, payload []byte) {
	h.taskRunner.Schedule(func() {
		msgId, _ := tool.SimpleUUID()
		sendTime := carbon.Now().ToDateTimeMicroString()
		startTime := timex.Now()

		_, err := h.streamEventCli.ReceiveMQTTMessage(ctx, &streamevent.ReceiveMQTTMessageReq{
			Messages: []*streamevent.MqttMessage{{
				SessionId:     h.clientID,
				MsgId:         msgId,
				Topic:         topic,
				Payload:       payload,
				SendTime:      sendTime,
				TopicTemplate: topicTemplate,
			}},
		})

		h.logPushResult(ctx, "grpc", msgId, topic, topicTemplate, sendTime, startTime, err)
	})
}

func (h *MqttStreamHandler) pushToSocket(ctx context.Context, topic, topicTemplate string, payload []byte) {
	h.taskRunner.Schedule(func() {
		reqId, _ := tool.SimpleUUID()
		sendTime := carbon.Now().ToDateTimeMicroString()
		startTime := timex.Now()
		event := h.matchEvent(topicTemplate)

		_, err := h.socketPushCli.BroadcastRoom(ctx, &socketpush.BroadcastRoomReq{
			ReqId:   reqId,
			Room:    topicTemplate,
			Event:   event,
			Payload: string(payload),
		})

		h.logPushResult(ctx, "socket", reqId, topic, topicTemplate, sendTime, startTime, err)
	})
}

func (h *MqttStreamHandler) logPushResult(ctx context.Context, pushType, reqId, topic, topicTemplate, sendTime string, startTime time.Duration, err error) {
	result := "success"
	if err != nil {
		result = "fail"
	}
	logx.WithContext(ctx).WithDuration(startTime).Infof(
		"push mqtt to %s, reqId: %s, topic: %s, topicTemplate: %s, time: %s - %s",
		pushType, reqId, topic, topicTemplate, sendTime, result,
	)
}
