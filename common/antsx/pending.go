package antsx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
)

// RegistryOption 配置 PendingRegistry 的函数式选项。
type RegistryOption func(*registryConfig)

type registryConfig struct {
	defaultTTL     time.Duration
	timingInterval time.Duration
	numSlots       int
}

// WithDefaultTTL 设置 PendingRegistry 的默认超时时间，未指定时默认 30s。
// Register 时未指定 ttl 参数将使用此值。
func WithDefaultTTL(d time.Duration) RegistryOption {
	return func(c *registryConfig) {
		c.defaultTTL = d
	}
}

// WithTimingWheel 自定义 TimingWheel 的刻度间隔和槽数。
// interval: 时间轮刻度间隔，建议设为 timeout/numSlots。
// numSlots: 时间轮槽数，建议设为 20-60。
func WithTimingWheel(interval time.Duration, numSlots int) RegistryOption {
	return func(c *registryConfig) {
		c.timingInterval = interval
		c.numSlots = numSlots
	}
}

// pendingEntry 是 PendingRegistry 内部的单条待匹配记录。
type pendingEntry[T any] struct {
	promise *Promise[T]
	removed bool
}

// PendingRegistry 是基于关联 ID 的异步请求-响应匹配注册表。
// 典型场景：MQ correlationId 匹配、TCP sendNo 匹配、RPC 请求-响应关联。
//
// 使用 go-zero TimingWheel 管理超时，所有定时器共享一个 ticker，
// 相比每个请求一个 time.Timer 大幅降低内存开销。
//
// 协程安全：所有方法均可在多个 goroutine 中并发调用。
type PendingRegistry[T any] struct {
	mu      sync.Mutex
	pending map[string]*pendingEntry[T]
	closed  bool
	cfg     registryConfig

	tw *collection.TimingWheel
}

// NewPendingRegistry 创建关联 ID 注册表。
// 默认配置：TTL 30s，时间轮 100ms 刻度 × 60 槽。
// 创建失败（极端情况）会 panic。
func NewPendingRegistry[T any](opts ...RegistryOption) *PendingRegistry[T] {
	cfg := registryConfig{
		defaultTTL:     30 * time.Second,
		timingInterval: time.Millisecond * 100,
		numSlots:       60,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	r := &PendingRegistry[T]{
		pending: make(map[string]*pendingEntry[T]),
		cfg:     cfg,
	}

	tw, err := collection.NewTimingWheel(cfg.timingInterval, cfg.numSlots, func(key, value any) {
		id, ok := key.(string)
		if !ok {
			return
		}
		r.handleTimeout(id)
	})
	if err != nil {
		panic("antsx: failed to create timing wheel: " + err.Error())
	}
	r.tw = tw

	return r
}

// handleTimeout 处理 TimingWheel 超时回调，以 ErrPendingExpired 拒绝过期条目。
//
// 锁安全性：由 TimingWheel 内部 goroutine 回调触发，先获取 r.mu 检查并删除 entry，
// Unlock 后调用 promise.Reject。Promise 内部使用 sync.Once 保证幂等，即使与
// Resolve/Reject 并发执行也安全——先从 map 中 delete 的一方获得操作权。
func (r *PendingRegistry[T]) handleTimeout(id string) {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	entry.removed = true
	delete(r.pending, id)
	r.mu.Unlock()

	entry.promise.Reject(ErrPendingExpired)
}

// Register 注册一个待匹配的请求，返回 Promise 用于等待响应。
// ttl 可选，未指定则使用 defaultTTL；超时后自动 Reject(ErrPendingExpired)。
// 重复 ID 返回 ErrDuplicateID，注册表已关闭返回 ErrRegistryClosed。
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

	promise := NewPromise[T]()
	entry := &pendingEntry[T]{
		promise: promise,
		removed: false,
	}
	r.pending[id] = entry
	r.mu.Unlock()

	// 锁安全性：Unlock 后调用 tw.SetTimer。TimingWheel 内部有独立的锁机制，
	// 不依赖 r.mu。此处 entry 已写入 pending map，即使 SetTimer 回调在设置完成前
	// 触发 handleTimeout，handleTimeout 也能正确找到并处理该 entry。
	// SetTimer 失败时需重新获取 r.mu 清理 entry，不存在锁重入问题。
	if err := r.tw.SetTimer(id, nil, effectiveTTL); err != nil {
		r.mu.Lock()
		delete(r.pending, id)
		r.mu.Unlock()
		return nil, fmt.Errorf("antsx: failed to set timer: %w", err)
	}

	return promise, nil
}

// Resolve 通过关联 ID 成功解决一个待匹配请求。
// 返回 false 表示 ID 不存在（已被解决、拒绝或过期）。
//
// 锁安全性：Lock 期间从 pending map 中删除 entry，Unlock 后调用 tw.RemoveTimer 和
// promise.Resolve。TimingWheel 和 Promise 各有独立同步机制，不持有 r.mu 时调用
// 可避免与 handleTimeout 回调产生锁嵌套。entry.removed 标志防止并发 handleTimeout 重复处理。
func (r *PendingRegistry[T]) Resolve(id string, val T) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	entry.removed = true
	delete(r.pending, id)
	r.mu.Unlock()

	r.tw.RemoveTimer(id)
	entry.promise.Resolve(val)
	return true
}

// Reject 通过关联 ID 拒绝一个待匹配请求。
// 返回 false 表示 ID 不存在。
//
// 锁安全性：与 Resolve 一致，Lock 内删除 entry，Unlock 后调用 tw.RemoveTimer 和
// promise.Reject，不持有 r.mu 时执行外部调用以避免锁嵌套。
func (r *PendingRegistry[T]) Reject(id string, err error) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	entry.removed = true
	delete(r.pending, id)
	r.mu.Unlock()

	r.tw.RemoveTimer(id)
	entry.promise.Reject(err)
	return true
}

// Has 检查指定 ID 是否在待匹配状态。
func (r *PendingRegistry[T]) Has(id string) bool {
	r.mu.Lock()
	_, ok := r.pending[id]
	r.mu.Unlock()
	return ok
}

// Len 返回当前待匹配请求数量。
func (r *PendingRegistry[T]) Len() int {
	r.mu.Lock()
	n := len(r.pending)
	r.mu.Unlock()
	return n
}

// Close 关闭注册表，所有待匹配请求以 ErrRegistryClosed 拒绝。
// 幂等调用安全。关闭后 Register 返回 ErrRegistryClosed。
//
// 锁安全性：Lock 期间设置 closed 标志并快照 pending map，Unlock 后调用 tw.Stop
// 和遍历快照逐个 promise.Reject。快照保证遍历期间不受并发 handleTimeout 修改影响；
// tw.Stop 在 Unlock 后调用，TimingWheel 有独立锁机制；promise.Reject 幂等安全。
func (r *PendingRegistry[T]) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true

	snapshot := make(map[string]*pendingEntry[T], len(r.pending))
	for k, v := range r.pending {
		snapshot[k] = v
	}
	r.pending = make(map[string]*pendingEntry[T])
	r.mu.Unlock()

	r.tw.Stop()

	for _, e := range snapshot {
		e.promise.Reject(ErrRegistryClosed)
	}
}

// RequestReply 是注册-发送-等待响应的便捷封装。
// sendFn 负责实际发送请求（如发 MQ 消息、写 TCP 帧），
// 如果 sendFn 失败，自动清理已注册的条目。
// ctx 用于控制等待超时。
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
