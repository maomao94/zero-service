package handler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	"zero-service/common/mqttx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

type TopicLogConfig struct {
	LogPayload     atomic.Bool
	MinLogInterval int64
	LastLogUnix    int64
}

func NewTopicLogConfig() *TopicLogConfig {
	return &TopicLogConfig{
		MinLogInterval: int64(5 * time.Second),
		LastLogUnix:    0,
	}
}

func (c *TopicLogConfig) ShouldLog() bool {
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&c.LastLogUnix)
	if now-last < c.MinLogInterval {
		return false
	}
	return atomic.CompareAndSwapInt64(&c.LastLogUnix, last, now)
}

func (c *TopicLogConfig) ShouldLogPayload() bool {
	return c.LogPayload.Load()
}

func (c *TopicLogConfig) SetLogPayload(enabled bool) {
	c.LogPayload.Store(enabled)
}

type TopicLogManagerOption func(*TopicLogManager)

type TopicLogManager struct {
	configs           sync.Map
	defaultLogPayload atomic.Bool
}

func NewTopicLogManager(opts ...TopicLogManagerOption) *TopicLogManager {
	manager := &TopicLogManager{}
	for _, opt := range opts {
		opt(manager)
	}
	return manager
}

func WithDefaultLogPayload(enabled bool) TopicLogManagerOption {
	return func(m *TopicLogManager) {
		m.defaultLogPayload.Store(enabled)
	}
}

func (m *TopicLogManager) SetDefaultLogPayload(enabled bool) {
	m.defaultLogPayload.Store(enabled)
}

func (m *TopicLogManager) GetConfig(topic string) *TopicLogConfig {
	if v, ok := m.configs.Load(topic); ok {
		return v.(*TopicLogConfig)
	}
	config := NewTopicLogConfig()
	config.SetLogPayload(m.defaultLogPayload.Load())
	actual, _ := m.configs.LoadOrStore(topic, config)
	return actual.(*TopicLogConfig)
}

func (m *TopicLogManager) ShouldLog(topic string) bool {
	config := m.GetConfig(topic)
	return config.ShouldLog()
}

func (m *TopicLogManager) ShouldLogPayload(topic string) bool {
	config := m.GetConfig(topic)
	return config.ShouldLogPayload()
}

func (m *TopicLogManager) SetLogPayload(topic string, enabled bool) {
	config := m.GetConfig(topic)
	config.SetLogPayload(enabled)
}

type MqttStreamHandler struct {
	clientID       string
	streamEventCli streamevent.StreamEventClient
	socketPushCli  socketpush.SocketPushClient
	taskRunner     *threading.TaskRunner
	eventMapping   []mqttx.EventMapping
	defaultEvent   string
	logManager     *TopicLogManager
}

func NewMqttStreamHandler(clientID string, streamEventCli streamevent.StreamEventClient, socketPushCli socketpush.SocketPushClient, eventMapping []mqttx.EventMapping, defaultEvent string) *MqttStreamHandler {
	return &MqttStreamHandler{
		clientID:       clientID,
		streamEventCli: streamEventCli,
		socketPushCli:  socketPushCli,
		taskRunner:     threading.NewTaskRunner(16),
		eventMapping:   eventMapping,
		defaultEvent:   defaultEvent,
		logManager:     NewTopicLogManager(WithDefaultLogPayload(true)),
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
	shouldLog := h.logManager.ShouldLog(topicTemplate)
	shouldLogPayload := h.logManager.ShouldLogPayload(topicTemplate)
	if shouldLog {
		if shouldLogPayload {
			logx.WithContext(ctx).Infof("receive mqtt message, topic: %s, topicTemplate: %s, payload: %s", topic, topicTemplate, string(payload))
		} else {
			logx.WithContext(ctx).Infof("receive mqtt message, topic: %s, topicTemplate: %s", topic, topicTemplate)
		}
	}
	if h.streamEventCli != nil {
		h.taskRunner.Schedule(func() {
			msgId, _ := tool.SimpleUUID()
			sendTime := carbon.Now().ToDateTimeMicroString()
			startTime := timex.Now()
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
			duration := timex.Since(startTime)
			invokeflg := "success"
			if err != nil {
				invokeflg = "fail"
			}
			logger := logx.WithContext(ctx).WithDuration(duration)
			logger.Infof("push mqtt to grpc, msgId: %s, topic: %s, topicTemplate: %s, time: %s - %s", msgId, topic, topicTemplate, sendTime, invokeflg)
		})
	}
	if h.socketPushCli != nil {
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
			duration := timex.Since(startTime)
			invokeflg := "success"
			if err != nil {
				invokeflg = "fail"
			}
			logger := logx.WithContext(ctx).WithDuration(duration)
			logger.Infof("push mqtt to socket, reqId: %s, topic: %s, topicTemplate: %s, time: %s - %s", reqId, topic, topicTemplate, sendTime, invokeflg)
		})
	}
	return nil
}

// lock 优化
//type TopicLogConfig struct {
//	mutex          sync.Mutex
//	LogPayload     bool
//	MinLogInterval time.Duration
//	LastLogTime    time.Time
//}
//
//func (c *TopicLogConfig) ShouldLog() bool {
//	c.mutex.Lock()
//	defer c.mutex.Unlock()
//
//	now := time.Now()
//
//	if now.Sub(c.LastLogTime) < c.MinLogInterval {
//		return false
//	}
//
//	c.LastLogTime = now
//	return true
//}
//
//type TopicLogManager struct {
//	configs map[string]*TopicLogConfig
//	mutex   sync.RWMutex
//}
//
//func NewTopicLogManager() *TopicLogManager {
//	return &TopicLogManager{
//		configs: make(map[string]*TopicLogConfig),
//	}
//}
//
//func (m *TopicLogManager) GetConfig(topic string) *TopicLogConfig {
//	m.mutex.RLock()
//	config, ok := m.configs[topic]
//	m.mutex.RUnlock()
//	if ok {
//		return config
//	}
//	m.mutex.Lock()
//	defer m.mutex.Unlock()
//	config, ok = m.configs[topic]
//	if ok {
//		return config
//	}
//	config = &TopicLogConfig{
//		LogPayload:     false,
//		MinLogInterval: 5 * time.Second,
//		LastLogTime:    time.Now(),
//	}
//	m.configs[topic] = config
//	return config
//}
//
//func (m *TopicLogManager) ShouldLog(topic string) bool {
//	config := m.GetConfig(topic)
//	return config.ShouldLog()
//}
//
//func (m *TopicLogManager) ShouldLogPayload(topic string) bool {
//	config := m.GetConfig(topic)
//	return config.LogPayload
//}
