package mqttx

import (
	"context"
	"errors"
	"slices"
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
	// topic: MQTT 消息实际主题（如 device/123/data）
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
// 维护主题到处理器列表的映射；reply handler 独立存储，key = topic template。
type handlerManager struct {
	mu            sync.RWMutex
	handlers      map[string][]ConsumeHandler // key: topic template → ordered handler list
	replyHandlers map[string]ConsumeHandler   // key: topic template → single reply handler
}

func newHandlerManager() *handlerManager {
	return &handlerManager{
		handlers:      make(map[string][]ConsumeHandler),
		replyHandlers: make(map[string]ConsumeHandler),
	}
}

// addHandler 添加普通处理器到指定订阅模板（保留注册顺序）。
func (m *handlerManager) addHandler(topicTemplate string, h ConsumeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[topicTemplate] = append(m.handlers[topicTemplate], h)
}

// addReplyHandler 注册 reply handler 到指定订阅模板。
// reply/普通 handler 的区别仅由注册路径区分，不依赖接口类型。
func (m *handlerManager) addReplyHandler(topicTemplate string, h ConsumeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replyHandlers[topicTemplate] = h
}

// getReplyHandler returns the reply handler for a topic template, or nil.
func (m *handlerManager) getReplyHandler(topicTemplate string) ConsumeHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.replyHandlers[topicTemplate]
}

// closeReplyHandlers closes any reply handler that implements Close().
func (m *handlerManager) closeReplyHandlers() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, h := range m.replyHandlers {
		if closer, ok := h.(interface{ Close() }); ok {
			closer.Close()
		}
	}
}

// getHandlers 获取主题模板对应的所有普通处理器（保留注册顺序）。
func (m *handlerManager) getHandlers(topicTemplate string) []ConsumeHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.handlers[topicTemplate])
}

// getAllTopicTemplates 获取所有已注册处理器的订阅模板（handler + replyHandler，去重，顺序不保证）。
func (m *handlerManager) getAllTopicTemplates() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	seen := make(map[string]struct{}, len(m.handlers)+len(m.replyHandlers))
	for topicTemplate := range m.handlers {
		seen[topicTemplate] = struct{}{}
	}
	for topicTemplate := range m.replyHandlers {
		seen[topicTemplate] = struct{}{}
	}
	topicTemplates := make([]string, 0, len(seen))
	for topicTemplate := range seen {
		topicTemplates = append(topicTemplates, topicTemplate)
	}
	return topicTemplates
}

// messageDispatcher 消息分发器。
// 根据触发回调的订阅模板调用对应处理器；不负责合并订阅 topic。
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
			logx.WithContext(ctx).Info("[mqtt] no handler registered")
		},
	}
}

// dispatch 根据触发回调的订阅模板分发消息到对应的处理器。
//
// reply handler 先运行，无论是否匹配 pending，之后都会运行同 topic 的普通 handler。
// 两者独立——reply 负责解析 tid、匹配 pending 请求；普通 handler 负责业务处理（通知、落库等）。
// 仅当 topic 上既无 reply handler 也无普通 handler 时才触发 onNoHandler。
func (d *messageDispatcher) dispatch(ctx context.Context, payload []byte, topic, topicTemplate string) {
	startTime := timex.Now()
	defer func() {
		d.metrics.Add(stat.Task{Duration: timex.Since(startTime)})
	}()

	replyHandler := d.manager.getReplyHandler(topicTemplate)
	handlers := d.manager.getHandlers(topicTemplate)
	if replyHandler == nil && len(handlers) == 0 {
		if d.onNoHandler != nil {
			d.onNoHandler(ctx, payload, topic, topicTemplate)
		}
		return
	}

	if replyHandler != nil {
		err := replyHandler.Consume(ctx, payload, topic, topicTemplate)
		if err != nil && !errors.Is(err, ErrReplyNotMatched) {
			logx.WithContext(ctx).Errorf("[mqtt] reply handler error: %v", err)
		}
	}

	for _, handler := range handlers {
		if err := handler.Consume(ctx, payload, topic, topicTemplate); err != nil {
			logx.WithContext(ctx).Errorf("[mqtt] handler error: %v", err)
		}
	}
}

// SetNoHandlerHandler 设置无处理器时的回调
func (d *messageDispatcher) SetNoHandlerHandler(fn func(ctx context.Context, payload []byte, topic, topicTemplate string)) {
	if fn == nil {
		return
	}
	d.onNoHandler = fn
}
