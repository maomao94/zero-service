package antsx

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
)

// RegistryOption 配置 PendingRegistry 的函数式选项。
type RegistryOption func(*registryConfig)

type registryConfig struct {
	defaultTTL    time.Duration
	statsInterval time.Duration
}

// RegistryStats 包含 PendingRegistry 的运行时统计。
type RegistryStats struct {
	Registered uint64
	Resolved   uint64
	Rejected   uint64
	Expired    uint64
	Pending    int

	IntervalRegistered uint64
	IntervalResolved   uint64
	IntervalRejected   uint64
	IntervalExpired    uint64
	IntervalDuration   time.Duration
}

// WithDefaultTTL 设置 PendingRegistry 的默认超时时间，未指定时默认 30s。
// Register 时未指定 ttl 参数将使用此值。
// 时间轮参数会自动根据 TTL 推导，无需手动配置。
func WithDefaultTTL(d time.Duration) RegistryOption {
	return func(c *registryConfig) {
		c.defaultTTL = d
	}
}

// WithStatsLoop 启用统计日志循环，按指定间隔打印 PendingRegistry 运行时统计。
func WithStatsLoop(interval time.Duration) RegistryOption {
	return func(c *registryConfig) {
		c.statsInterval = interval
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

	registered atomic.Uint64
	resolved   atomic.Uint64
	rejected   atomic.Uint64
	expired    atomic.Uint64

	deltaRegistered atomic.Uint64
	deltaResolved   atomic.Uint64
	deltaRejected   atomic.Uint64
	deltaExpired    atomic.Uint64
}

func autoTimingWheel(ttl time.Duration) (interval time.Duration, numSlots int) {
	const slots = 300
	interval = ttl / slots
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	if interval > time.Second {
		interval = time.Second
	}
	return interval, slots
}

// NewPendingRegistry 创建关联 ID 注册表。
// 默认配置：TTL 30s，时间轮参数自动推导。
// 创建失败（极端情况）会 panic。
func NewPendingRegistry[T any](opts ...RegistryOption) *PendingRegistry[T] {
	cfg := registryConfig{
		defaultTTL: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	r := &PendingRegistry[T]{
		pending: make(map[string]*pendingEntry[T]),
		cfg:     cfg,
	}

	interval, numSlots := autoTimingWheel(cfg.defaultTTL)
	tw, err := collection.NewTimingWheel(interval, numSlots, func(key, value any) {
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

func (r *PendingRegistry[T]) handleTimeout(id string) {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	entry.removed = true
	delete(r.pending, id)
	r.expired.Add(1)
	r.deltaExpired.Add(1)
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
	r.registered.Add(1)
	r.deltaRegistered.Add(1)
	r.mu.Unlock()

	if err := r.tw.SetTimer(id, nil, effectiveTTL); err != nil {
		r.mu.Lock()
		delete(r.pending, id)
		r.registered.Add(^uint64(0))
		r.deltaRegistered.Add(^uint64(0))
		r.mu.Unlock()
		return nil, fmt.Errorf("antsx: failed to set timer: %w", err)
	}

	return promise, nil
}

// Resolve 通过关联 ID 成功解决一个待匹配请求。
// 返回 false 表示 ID 不存在（已被解决、拒绝或过期）。
func (r *PendingRegistry[T]) Resolve(id string, val T) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	entry.removed = true
	delete(r.pending, id)
	r.resolved.Add(1)
	r.deltaResolved.Add(1)
	r.mu.Unlock()

	r.tw.RemoveTimer(id)
	entry.promise.Resolve(val)
	return true
}

// Reject 通过关联 ID 拒绝一个待匹配请求。
// 返回 false 表示 ID 不存在。
func (r *PendingRegistry[T]) Reject(id string, err error) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	entry.removed = true
	delete(r.pending, id)
	r.rejected.Add(1)
	r.deltaRejected.Add(1)
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

	rejected := uint64(0)
	for _, e := range snapshot {
		e.promise.Reject(ErrRegistryClosed)
		rejected++
	}
	r.rejected.Add(rejected)
	r.deltaRejected.Add(rejected)
}

// Stats 返回当前运行时统计快照。
func (r *PendingRegistry[T]) Stats() RegistryStats {
	r.mu.Lock()
	pending := len(r.pending)
	r.mu.Unlock()

	return RegistryStats{
		Registered: r.registered.Load(),
		Resolved:   r.resolved.Load(),
		Rejected:   r.rejected.Load(),
		Expired:    r.expired.Load(),
		Pending:    pending,
	}
}

// StartStatsLoop 启动后台统计日志循环，按指定间隔打印区间增量和累计总计。
// 返回 stop 函数，调用后停止循环。
func (r *PendingRegistry[T]) StartStatsLoop(ctx context.Context, interval time.Duration, logFn func(RegistryStats)) func() {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		lastTime := time.Now()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				elapsed := now.Sub(lastTime)
				lastTime = now

				ir := r.deltaRegistered.Swap(0)
				ires := r.deltaResolved.Swap(0)
				irej := r.deltaRejected.Swap(0)
				iexp := r.deltaExpired.Swap(0)

				total := ir + ires + irej + iexp
				if total == 0 {
					continue
				}

				logFn(RegistryStats{
					Registered: r.registered.Load(),
					Resolved:   r.resolved.Load(),
					Rejected:   r.rejected.Load(),
					Expired:    r.expired.Load(),
					Pending:    r.Len(),

					IntervalRegistered: ir,
					IntervalResolved:   ires,
					IntervalRejected:   irej,
					IntervalExpired:    iexp,
					IntervalDuration:   elapsed,
				})
			case <-ctx.Done():
				return
			}
		}
	}()
	return cancel
}

// RequestReply 是注册-发送-等待响应的便捷封装。
// sendFn 负责实际发送请求（如发 MQ 消息、写 TCP 帧），
// 如果 sendFn 失败，自动清理已注册的条目。
// ctx 用于控制等待超时。
// ttl 可选，未指定则使用 PendingRegistry 的 defaultTTL；指定时覆盖本次请求的 TTL。
func RequestReply[T any](ctx context.Context, reg *PendingRegistry[T], id string, sendFn func() error, ttl ...time.Duration) (T, error) {
	promise, err := reg.Register(id, ttl...)
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
