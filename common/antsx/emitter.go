package antsx

import (
	"context"
	"sync"
)

// emitterSub 是 EventEmitter 内部的单个订阅者，持有一个 StreamWriter 用于向订阅者推送数据。
// mu 保护 send 和 stop 之间的并发安全，防止 close(items) 与 items<-item 的数据竞争。
type emitterSub[T any] struct {
	sw     *StreamWriter[T]
	mu     sync.Mutex
	closed bool
	once   sync.Once
	done   chan struct{} // ctx 取消监听 goroutine 的退出信号
}

// send 向订阅者推送一个值。如果订阅者已取消则静默忽略。
// 通过 mu 锁与 stop 互斥，避免 close(items) 与 items<-item 的竞争。
//
// 锁安全性：持有 s.mu 时调用 sw.Send()，sw.Send 内部执行 channel 写入。
// 当 channel 缓冲区满时 Send 会阻塞，此时若其他 goroutine 调用 stop()
// 需等待 s.mu 释放。这不构成死锁，仅产生有界等待——消费端消费一个元素后
// channel 可写，send 返回并释放 s.mu，stop 即可获取锁继续执行。
func (s *emitterSub[T]) send(val T) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.sw.Send(val, nil)
	s.mu.Unlock()
}

// stop 关闭订阅者，通过 sync.Once 保证幂等。
// mu 锁确保 close(items) 执行时不会有并发的 send 在写入 items。
func (s *emitterSub[T]) stop() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.sw.Close()
		s.mu.Unlock()
		close(s.done)
	})
}

// EventEmitter 是基于 topic 的泛型发布/订阅事件分发器。
// 订阅者通过 Subscribe 获得 StreamReader 来接收事件，发布者通过 Emit 发送事件。
//
// 协程安全：所有方法均可在多个 goroutine 中并发调用。
//
// 用法:
//
//	emitter := antsx.NewEventEmitter[string]()
//	sr, cancel := emitter.Subscribe(ctx, "topic-1")
//	defer cancel()
//	go emitter.Emit("topic-1", "hello")
//	val, _ := sr.Recv() // "hello"
type EventEmitter[T any] struct {
	mu          sync.RWMutex
	subscribers map[string][]*emitterSub[T]
	closed      bool
}

// NewEventEmitter 创建一个新的 EventEmitter 实例。
func NewEventEmitter[T any]() *EventEmitter[T] {
	return &EventEmitter[T]{
		subscribers: make(map[string][]*emitterSub[T]),
	}
}

// Subscribe 订阅指定 topic，返回用于接收事件的 StreamReader 和取消订阅的函数。
// ctx 用于控制订阅生命周期：ctx 取消时自动执行 cancel 并关闭 StreamReader。
// bufSize 可选，指定内部 channel 缓冲大小，默认 16。
// 调用 cancel 函数后，StreamReader 将收到 io.EOF。
// 如果 EventEmitter 已关闭，返回的 StreamReader 会立即 EOF。
func (e *EventEmitter[T]) Subscribe(ctx context.Context, topic string, bufSize ...int) (*StreamReader[T], func()) {
	size := 16
	if len(bufSize) > 0 && bufSize[0] > 0 {
		size = bufSize[0]
	}

	sr, sw := Pipe[T](size)
	sub := &emitterSub[T]{
		sw:   sw,
		done: make(chan struct{}),
	}

	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		sub.stop()
		return sr, func() {}
	}
	e.subscribers[topic] = append(e.subscribers[topic], sub)
	e.mu.Unlock()

	var cancelOnce sync.Once
	// 锁安全性：cancel 内部先 Lock 修改 subscribers 列表，Unlock 后再调用 sub.stop。
	// sub.stop 内部获取 emitterSub.mu，与 EventEmitter.mu 不同层级，避免持锁嵌套。
	// 若持有 e.mu 时调用 sub.stop，而 send 持有 emitterSub.mu 等待 e.mu（不会发生，
	// 因 send 不获取 e.mu），但此处仍主动规避以保持统一的锁分离策略。
	cancel := func() {
		cancelOnce.Do(func() {
			e.mu.Lock()
			subs := e.subscribers[topic]
			for i, s := range subs {
				if s == sub {
					e.subscribers[topic] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
			if len(e.subscribers[topic]) == 0 {
				delete(e.subscribers, topic)
			}
			e.mu.Unlock()
			sub.stop()
		})
	}

	// 后台 goroutine 监听 ctx 取消，自动执行 cancel
	// cancel 的 cancelOnce 保证与手动调用 cancel 互斥，不会重复执行
	if ctx.Done() != nil {
		go func() {
			select {
			case <-ctx.Done():
				cancel()
			case <-sub.done:
			}
		}()
	}

	return sr, cancel
}

// Emit 向指定 topic 的所有订阅者广播一个值。
// 如果没有订阅者或 EventEmitter 已关闭，则静默忽略。
//
// 锁安全性：RLock 期间仅做快照拷贝，RUnlock 后遍历快照调用 sub.send。
// send 内部使用 emitterSub.mu，与 EventEmitter.mu 属于不同锁层级，不存在锁嵌套。
// 快照拷贝保证遍历期间不受并发 Subscribe/cancel 修改 subscribers 的影响。
func (e *EventEmitter[T]) Emit(topic string, value T) {
	e.mu.RLock()
	subs := make([]*emitterSub[T], len(e.subscribers[topic]))
	copy(subs, e.subscribers[topic])
	e.mu.RUnlock()
	for _, sub := range subs {
		sub.send(value)
	}
}

// TopicCount 返回当前有活跃订阅者的 topic 数量。
func (e *EventEmitter[T]) TopicCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.subscribers)
}

// SubscriberCount 返回指定 topic 的当前订阅者数量。
func (e *EventEmitter[T]) SubscriberCount(topic string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.subscribers[topic])
}

// Close 关闭 EventEmitter，停止所有订阅者的 StreamWriter。
// 幂等调用安全，多次调用不会 panic。关闭后 Subscribe 返回的 StreamReader 会立即 EOF。
//
// 锁安全性：Lock 期间设置 closed 标志并快照 subscribers，Unlock 后遍历快照调用 sub.stop。
// 与 Emit、cancel 保持一致的锁分离策略：不持有 e.mu 时调用 sub.stop，避免锁嵌套。
func (e *EventEmitter[T]) Close() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.closed = true
	snapshot := e.subscribers
	e.subscribers = make(map[string][]*emitterSub[T])
	e.mu.Unlock()

	for _, subs := range snapshot {
		for _, sub := range subs {
			sub.stop()
		}
	}
}
