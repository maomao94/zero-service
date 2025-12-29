package handler

import (
	"context"
	"time"
	"zero-service/common/socketio"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/gateway/socketgtw/socketgtw"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

type MqttStreamHandler struct {
	clientID        string
	cli             streamevent.StreamEventClient
	socketContainer *socketio.SocketContainer
}

func NewMqttStreamHandler(clientID string, cli streamevent.StreamEventClient, container *socketio.SocketContainer) *MqttStreamHandler {
	return &MqttStreamHandler{
		clientID:        clientID,
		cli:             cli,
		socketContainer: container,
	}
}

func (h *MqttStreamHandler) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	threading.GoSafe(func() {
		msgId, _ := tool.SimpleUUID()
		sendTime := carbon.Now().ToDateTimeMicroString()
		startTime := timex.Now()
		duration := timex.Since(startTime)
		mqttCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_, err := h.cli.ReceiveMQTTMessage(mqttCtx, &streamevent.ReceiveMQTTMessageReq{
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
		logx.WithContext(ctx).WithDuration(duration).Infof("consume mqtt message, msgId: %s, topic: %s, topicTemplate: %s, time: %s - %s", msgId, topic, topicTemplate, sendTime, invokeflg)
	})
	threading.GoSafe(func() {
		reqId, _ := tool.SimpleUUID()
		for key, cli := range h.socketContainer.GetClients() {
			sendTime := carbon.Now().ToDateTimeMicroString()
			startTime := timex.Now()
			duration := timex.Since(startTime)
			socktCTx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			_, err := cli.BroadcastGlobal(socktCTx, &socketgtw.BroadcastGlobalReq{
				ReqId:   reqId,
				Event:   topicTemplate,
				Payload: payload,
			})
			var invokeflg = "success"
			if err != nil {
				invokeflg = "fail"
			}
			logx.WithContext(ctx).WithDuration(duration).Infof("[mqtt] broadcast socketio global, node: %s, reqId: %s, topicTemplate: %s, time: %s - %s", key, reqId, topicTemplate, sendTime, invokeflg)
		}
	})
	return nil
}
