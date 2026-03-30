package antsx

import (
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

// Subscriber 订阅者
type Subscriber[T any] struct {
	ch   chan T
	once sync.Once
}

// EventEmitter 事件发射器，支持 topic 级别的发布/订阅
type EventEmitter[T any] struct {
	mu          sync.RWMutex
	subscribers map[string][]*Subscriber[T]
	closed      bool
}

// NewEventEmitter 创建事件发射器
func NewEventEmitter[T any]() *EventEmitter[T] {
	return &EventEmitter[T]{
		subscribers: make(map[string][]*Subscriber[T]),
	}
}

// Subscribe 订阅指定 topic，返回只读 channel 和取消函数
// bufSize 可选，默认为 16
func (e *EventEmitter[T]) Subscribe(topic string, bufSize ...int) (<-chan T, func()) {
	size := 16
	if len(bufSize) > 0 && bufSize[0] > 0 {
		size = bufSize[0]
	}

	sub := &Subscriber[T]{
		ch: make(chan T, size),
	}

	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		close(sub.ch)
		return sub.ch, func() {}
	}
	e.subscribers[topic] = append(e.subscribers[topic], sub)
	e.mu.Unlock()

	cancel := func() {
		sub.once.Do(func() {
			e.mu.Lock()
			subs := e.subscribers[topic]
			for i, s := range subs {
				if s == sub {
					logx.Debugf("unsubscribe topic: %s, index: %d", topic, i)
					// 使用 copy 方式删除，避免底层数组保留对已删除元素的引用
					if i == 0 {
						e.subscribers[topic] = subs[1:]
					} else if i == len(subs)-1 {
						e.subscribers[topic] = subs[:i]
					} else {
						newSubs := make([]*Subscriber[T], len(subs)-1)
						copy(newSubs, subs[:i])
						copy(newSubs[i:], subs[i+1:])
						e.subscribers[topic] = newSubs
					}
					break
				}
			}
			if len(e.subscribers[topic]) == 0 {
				delete(e.subscribers, topic)
			}
			e.mu.Unlock()
			close(sub.ch)
		})
	}

	return sub.ch, cancel
}

// Emit 向指定 topic 的所有订阅者非阻塞广播事件
func (e *EventEmitter[T]) Emit(topic string, value T) {
	e.mu.RLock()
	subs := make([]*Subscriber[T], len(e.subscribers[topic]))
	copy(subs, e.subscribers[topic])
	e.mu.RUnlock()
	if len(subs) == 0 {
		logx.Debugf("no subscriber for topic: %s", topic)
	}
	for _, sub := range subs {
		select {
		case sub.ch <- value:
		default:
			// 非阻塞，丢弃慢消费者的消息
		}
	}
}

// TopicCount 返回当前活跃的 topic 数量
func (e *EventEmitter[T]) TopicCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.subscribers)
}

// SubscriberCount 返回指定 topic 的订阅者数量
func (e *EventEmitter[T]) SubscriberCount(topic string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.subscribers[topic])
}

// Close 关闭所有订阅者 channel
func (e *EventEmitter[T]) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return
	}
	e.closed = true

	for topic, subs := range e.subscribers {
		// 先从 map 中删除，阻止新的 emit 操作访问到这些订阅者
		delete(e.subscribers, topic)
		// 再关闭 channel
		for _, sub := range subs {
			sub.once.Do(func() {
				close(sub.ch)
			})
		}
	}
}
