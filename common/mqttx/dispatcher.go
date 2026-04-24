package mqttx

import (
	"context"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
)

// ConsumeHandler 消息消费接口
// 实现此接口来处理 MQTT 消息
type ConsumeHandler interface {
	// Consume 消费消息
	// ctx: 上下文
	// payload: 消息内容
	// topic: 完整主题（如 device/123/data）
	// topicTemplate: 主题模板（如 device/+/data）
	Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error
}

// ConsumeHandlerFunc 函数适配器
// 允许使用普通函数作为 ConsumeHandler
type ConsumeHandlerFunc func(ctx context.Context, payload []byte, topic string, topicTemplate string) error

func (f ConsumeHandlerFunc) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	return f(ctx, payload, topic, topicTemplate)
}

// handlerManager 处理器管理器
// 维护主题到处理器列表的映射
type handlerManager struct {
	mu       sync.RWMutex
	handlers map[string][]ConsumeHandler
}

func newHandlerManager() *handlerManager {
	return &handlerManager{
		handlers: make(map[string][]ConsumeHandler),
	}
}

// addHandler 添加处理器到指定主题
func (m *handlerManager) addHandler(topic string, handler ConsumeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[topic] = append(m.handlers[topic], handler)
}

// getHandlers 获取主题对应的所有处理器
func (m *handlerManager) getHandlers(topic string) []ConsumeHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handlers[topic]
}

// getAllTopics 获取所有已注册处理器的主题
func (m *handlerManager) getAllTopics() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	topics := make([]string, 0, len(m.handlers))
	for topic := range m.handlers {
		topics = append(topics, topic)
	}
	return topics
}

// messageDispatcher 消息调度器
// 负责将消息分发给对应的处理器
type messageDispatcher struct {
	manager     *handlerManager
	metrics     *stat.Metrics
	onNoHandler func(ctx context.Context, payload []byte, topic, topicTemplate string)
}

func newMessageDispatcher(m *handlerManager, metrics *stat.Metrics) *messageDispatcher {
	return &messageDispatcher{
		manager: m,
		metrics: metrics,
		onNoHandler: func(ctx context.Context, payload []byte, topic, topicTemplate string) {
			logx.WithContext(ctx).Infof("[mqttx] no handler registered for topic=%s template=%s", topic, topicTemplate)
		},
	}
}

// dispatch 调度消息到对应的处理器
func (d *messageDispatcher) dispatch(ctx context.Context, payload []byte, topic, topicTemplate string) {
	startTime := timex.Now()
	defer func() {
		d.metrics.Add(stat.Task{Duration: timex.Since(startTime)})
	}()

	handlers := d.manager.getHandlers(topicTemplate)
	if len(handlers) == 0 {
		d.onNoHandler(ctx, payload, topic, topicTemplate)
		return
	}

	// 依次调用所有处理器
	for _, h := range handlers {
		if err := h.Consume(ctx, payload, topic, topicTemplate); err != nil {
			logx.WithContext(ctx).Errorf("[mqttx] handler error for %s: %v", topicTemplate, err)
		}
	}
}

// SetNoHandlerHandler 设置无处理器时的回调
func (d *messageDispatcher) SetNoHandlerHandler(fn func(ctx context.Context, payload []byte, topic, topicTemplate string)) {
	d.onNoHandler = fn
}
