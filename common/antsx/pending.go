package antsx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
)

// RegistryOption 配置 PendingRegistry 的函数式选项
type RegistryOption func(*registryConfig)

type registryConfig struct {
	defaultTTL     time.Duration
	timingInterval time.Duration
	numSlots       int
}

// WithDefaultTTL 设置默认超时时间，默认 30s
func WithDefaultTTL(d time.Duration) RegistryOption {
	return func(c *registryConfig) {
		c.defaultTTL = d
	}
}

// WithTimingWheel 设置 TimingWheel 参数
// interval: 时间轮刻度间隔，建议设为 timeout/numSlots
// numSlots: 时间轮槽数，建议设为 20-60
func WithTimingWheel(interval time.Duration, numSlots int) RegistryOption {
	return func(c *registryConfig) {
		c.timingInterval = interval
		c.numSlots = numSlots
	}
}

type pendingEntry[T any] struct {
	promise *Promise[T]
	removed bool // 是否已被主动移除（resolve/reject）
}

// PendingRegistry 关联 ID 注册表，用于异步请求-响应匹配
// 典型场景：MQ correlationId 匹配、TCP sendNo 匹配
// 使用 go-zero TimingWheel 实现，所有定时器共享一个 ticker，避免内存泄漏
type PendingRegistry[T any] struct {
	mu      sync.Mutex
	pending map[string]*pendingEntry[T]
	closed  bool
	cfg     registryConfig

	tw *collection.TimingWheel
}

// NewPendingRegistry 创建关联 ID 注册表
func NewPendingRegistry[T any](opts ...RegistryOption) *PendingRegistry[T] {
	cfg := registryConfig{
		defaultTTL:     30 * time.Second,
		timingInterval: time.Millisecond * 100, // 默认 100ms 刻度
		numSlots:       60,                     // 默认 60 槽，约 6 秒一轮
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	r := &PendingRegistry[T]{
		pending: make(map[string]*pendingEntry[T]),
		cfg:     cfg,
	}

	// 创建 TimingWheel，timeout 回调处理过期
	tw, err := collection.NewTimingWheel(cfg.timingInterval, cfg.numSlots, func(key, value any) {
		id, ok := key.(string)
		if !ok {
			return
		}
		r.handleTimeout(id)
	})
	if err != nil {
		// 如果创建失败，使用降级方案（虽然不太可能发生）
		panic("failed to create timing wheel: " + err.Error())
	}
	r.tw = tw

	return r
}

// handleTimeout 处理定时器超时
func (r *PendingRegistry[T]) handleTimeout(id string) {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		// 已被 resolve/reject 或已被移除
		r.mu.Unlock()
		return
	}
	// 标记为已移除，防止再次被调用
	entry.removed = true
	delete(r.pending, id)
	r.mu.Unlock()

	entry.promise.Reject(ErrPendingExpired)
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
	entry := &pendingEntry[T]{
		promise: promise,
		removed: false,
	}
	r.pending[id] = entry
	r.mu.Unlock()

	// 使用 TimingWheel 设置定时器
	if err := r.tw.SetTimer(id, nil, effectiveTTL); err != nil {
		r.mu.Lock()
		delete(r.pending, id)
		r.mu.Unlock()
		return nil, fmt.Errorf("failed to set timer: %w", err)
	}

	return promise, nil
}

// Resolve 通过关联 ID 解决一个待匹配请求
// 返回 false 表示 ID 不存在（已被解决、拒绝或过期）
func (r *PendingRegistry[T]) Resolve(id string, val T) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	// 标记为已移除，防止 handleTimeout 再次调用
	entry.removed = true
	delete(r.pending, id)
	r.mu.Unlock()

	// 移除 TimingWheel 中的定时器
	r.tw.RemoveTimer(id)

	entry.promise.Resolve(val)
	return true
}

// Reject 通过关联 ID 拒绝一个待匹配请求
// 返回 false 表示 ID 不存在
func (r *PendingRegistry[T]) Reject(id string, err error) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	// 标记为已移除，防止 handleTimeout 再次调用
	entry.removed = true
	delete(r.pending, id)
	r.mu.Unlock()

	// 移除 TimingWheel 中的定时器
	r.tw.RemoveTimer(id)

	entry.promise.Reject(err)
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
	snapshot := make(map[string]*pendingEntry[T], len(r.pending))
	for k, v := range r.pending {
		snapshot[k] = v
	}
	r.pending = make(map[string]*pendingEntry[T])
	r.mu.Unlock()

	// 停止 TimingWheel
	r.tw.Stop()

	// 逐个 reject，避免长时间持锁
	for _, e := range snapshot {
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
