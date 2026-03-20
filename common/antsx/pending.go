package antsx

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RegistryOption 配置 PendingRegistry 的函数式选项
type RegistryOption func(*registryConfig)

type registryConfig struct {
	defaultTTL time.Duration
}

// WithDefaultTTL 设置默认超时时间，默认 30s
func WithDefaultTTL(d time.Duration) RegistryOption {
	return func(c *registryConfig) {
		c.defaultTTL = d
	}
}

type entry[T any] struct {
	promise *Promise[T]
	timer   *time.Timer
}

// PendingRegistry 关联 ID 注册表，用于异步请求-响应匹配
// 典型场景：MQ correlationId 匹配、TCP sendNo 匹配
type PendingRegistry[T any] struct {
	mu      sync.Mutex
	pending map[string]*entry[T]
	closed  bool
	cfg     registryConfig
}

// NewPendingRegistry 创建关联 ID 注册表
func NewPendingRegistry[T any](opts ...RegistryOption) *PendingRegistry[T] {
	cfg := registryConfig{
		defaultTTL: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &PendingRegistry[T]{
		pending: make(map[string]*entry[T]),
		cfg:     cfg,
	}
}

// Register 注册一个待匹配的请求，返回 Promise 用于等待响应
// ttl 可选，未指定则使用 defaultTTL；超时后自动 Reject(ErrPendingExpired)
func (r *PendingRegistry[T]) Register(id string, ttl ...time.Duration) (*Promise[T], error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRegistryClosed
	}
	if _, ok := r.pending[id]; ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrDuplicateID, id)
	}

	effectiveTTL := r.cfg.defaultTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		effectiveTTL = ttl[0]
	}

	promise := NewPromise[T](id)

	timer := time.AfterFunc(effectiveTTL, func() {
		r.mu.Lock()
		if _, ok := r.pending[id]; ok {
			delete(r.pending, id)
			r.mu.Unlock()
			promise.Reject(ErrPendingExpired)
		} else {
			r.mu.Unlock()
		}
	})

	r.pending[id] = &entry[T]{
		promise: promise,
		timer:   timer,
	}
	r.mu.Unlock()

	return promise, nil
}

// Resolve 通过关联 ID 解决一个待匹配请求
// 返回 false 表示 ID 不存在（已被解决、拒绝或过期）
func (r *PendingRegistry[T]) Resolve(id string, val T) bool {
	r.mu.Lock()
	e, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	delete(r.pending, id)
	e.timer.Stop()
	r.mu.Unlock()

	e.promise.Resolve(val)
	return true
}

// Reject 通过关联 ID 拒绝一个待匹配请求
// 返回 false 表示 ID 不存在
func (r *PendingRegistry[T]) Reject(id string, err error) bool {
	r.mu.Lock()
	e, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	delete(r.pending, id)
	e.timer.Stop()
	r.mu.Unlock()

	e.promise.Reject(err)
	return true
}

// Has 检查指定 ID 是否在待匹配状态
func (r *PendingRegistry[T]) Has(id string) bool {
	r.mu.Lock()
	_, ok := r.pending[id]
	r.mu.Unlock()
	return ok
}

// Len 返回当前待匹配请求数量
func (r *PendingRegistry[T]) Len() int {
	r.mu.Lock()
	n := len(r.pending)
	r.mu.Unlock()
	return n
}

// Close 关闭注册表，所有待匹配请求以 ErrRegistryClosed 拒绝
func (r *PendingRegistry[T]) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true

	// 快照并清空
	snapshot := make(map[string]*entry[T], len(r.pending))
	for k, v := range r.pending {
		snapshot[k] = v
	}
	r.pending = nil
	r.mu.Unlock()

	// 解锁后逐个 reject，避免长时间持锁
	for _, e := range snapshot {
		e.timer.Stop()
		e.promise.Reject(ErrRegistryClosed)
	}
}

// RequestReply 便捷封装：注册 -> 发送 -> 等待响应
// sendFn 负责实际发送请求（如发 MQ 消息、写 TCP 帧）
// 如果 sendFn 失败，自动清理已注册的 entry
func RequestReply[T any](ctx context.Context, reg *PendingRegistry[T], id string, sendFn func() error) (T, error) {
	promise, err := reg.Register(id)
	if err != nil {
		var zero T
		return zero, err
	}

	if err := sendFn(); err != nil {
		reg.Reject(id, err)
		var zero T
		return zero, err
	}

	return promise.Await(ctx)
}
