package antsx

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

// RegistryOption 配置 ReplyPool 的函数式选项。
type RegistryOption func(*registryConfig)

type registryConfig struct {
	defaultTTL time.Duration
	name       string
}

const defaultStatsInterval = time.Minute

// WithDefaultTTL 设置 ReplyPool 的默认超时时间，未指定时默认 30s。
// Register 时未指定 ttl 参数将使用此值。
// 时间轮参数会自动根据 TTL 推导，无需手动配置。
func WithDefaultTTL(d time.Duration) RegistryOption {
	return func(c *registryConfig) {
		c.defaultTTL = d
	}
}

// WithName 设置 ReplyPool 的名称，用于统计日志中区分多个池子。
func WithName(name string) RegistryOption {
	return func(c *registryConfig) {
		c.name = name
	}
}

// pendingEntry 是 ReplyPool 内部的单条待匹配记录。
type pendingEntry[T any] struct {
	promise *Promise[T]
}

// ReplyPool 是基于关联 ID 的异步请求-响应匹配注册表。
// 典型场景：MQ correlationId 匹配、TCP sendNo 匹配、RPC 请求-响应关联。
//
// 使用 go-zero TimingWheel 管理超时，所有定时器共享一个 ticker，
// 相比每个请求一个 time.Timer 大幅降低内存开销。
//
// 协程安全：所有方法均可在多个 goroutine 中并发调用。
type ReplyPool[T any] struct {
	mu      sync.Mutex
	pending map[string]*pendingEntry[T]
	closed  bool
	cfg     registryConfig

	tw *collection.TimingWheel

	deltaRegistered atomic.Uint64
	deltaResolved   atomic.Uint64
	deltaRejected   atomic.Uint64
	deltaExpired    atomic.Uint64

	statsCtx    context.Context
	statsCancel context.CancelFunc
	statsWG     sync.WaitGroup
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

// NewReplyPool 创建关联 ID 注册表。
// 默认配置：TTL 30s，时间轮参数自动推导。
// 创建失败（极端情况）会 panic。
func NewReplyPool[T any](opts ...RegistryOption) *ReplyPool[T] {
	cfg := registryConfig{
		defaultTTL: 30 * time.Second,
		name:       "reply-pool",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	r := &ReplyPool[T]{
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

	r.statsCtx, r.statsCancel = context.WithCancel(context.Background())
	r.statsWG.Add(1)
	go r.statLoop()

	return r
}

func (r *ReplyPool[T]) handleTimeout(id string) {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	delete(r.pending, id)
	r.deltaExpired.Add(1)
	r.mu.Unlock()

	logx.Debugw(fmt.Sprintf("[%s] entry %s expired by timing wheel", r.cfg.name, id),
		logx.Field("tid", id),
	)
	entry.promise.Reject(ErrReplyExpired)
}

// startInternalStatsLoop 启动内置统计循环，通过 logx.Statf 输出。
// 构造时 WithStatsLoop 配置 interval 后自动调用，用户无需手动启动。
type pendingStat struct {
	registered uint64
	resolved   uint64
	rejected   uint64
	expired    uint64
}

func (r *ReplyPool[T]) reset() pendingStat {
	return pendingStat{
		registered: r.deltaRegistered.Swap(0),
		resolved:   r.deltaResolved.Swap(0),
		rejected:   r.deltaRejected.Swap(0),
		expired:    r.deltaExpired.Swap(0),
	}
}

func (r *ReplyPool[T]) statLoop() {
	defer r.statsWG.Done()
	ticker := time.NewTicker(defaultStatsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s := r.reset()
			logx.Statf("%s[1m] - registered: %d, resolved: %d, rejected: %d, expired: %d, pending: %d",
				r.cfg.name, s.registered, s.resolved, s.rejected, s.expired, r.Len())
		case <-r.statsCtx.Done():
			return
		}
	}
}

// Register 注册一个待匹配的请求，返回 Promise 用于等待响应。
// ttl 可选，未指定则使用 defaultTTL；超时后自动 Reject(ErrReplyExpired)。
// 重复 ID 返回 ErrDuplicateID，注册表已关闭返回 ErrReplyClosed。
func (r *ReplyPool[T]) Register(id string, ttl ...time.Duration) (*Promise[T], error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrReplyClosed
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
	}
	r.pending[id] = entry
	r.deltaRegistered.Add(1)
	r.mu.Unlock()

	if err := r.tw.SetTimer(id, nil, effectiveTTL); err != nil {
		r.mu.Lock()
		delete(r.pending, id)
		r.deltaRegistered.Add(^uint64(0))
		r.mu.Unlock()
		return nil, fmt.Errorf("antsx: failed to set timer: %w", err)
	}

	return promise, nil
}

// Resolve 通过关联 ID 成功解决一个待匹配请求。
// 返回 false 表示 ID 不存在（已被解决、拒绝或过期）。
func (r *ReplyPool[T]) Resolve(id string, val T) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	delete(r.pending, id)
	r.deltaResolved.Add(1)
	r.mu.Unlock()

	_ = r.tw.RemoveTimer(id)
	logx.Debugw(fmt.Sprintf("[%s] entry %s resolved", r.cfg.name, id),
		logx.Field("tid", id),
	)
	entry.promise.Resolve(val)
	return true
}

// Reject 通过关联 ID 拒绝一个待匹配请求。
// 返回 false 表示 ID 不存在。
func (r *ReplyPool[T]) Reject(id string, err error) bool {
	r.mu.Lock()
	entry, ok := r.pending[id]
	if !ok {
		r.mu.Unlock()
		return false
	}
	delete(r.pending, id)
	r.deltaRejected.Add(1)
	r.mu.Unlock()

	_ = r.tw.RemoveTimer(id)
	logx.Debugw(fmt.Sprintf("[%s] entry %s rejected: %v", r.cfg.name, id, err),
		logx.Field("tid", id),
	)
	entry.promise.Reject(err)
	return true
}

// Has 检查指定 ID 是否在待匹配状态。
func (r *ReplyPool[T]) Has(id string) bool {
	r.mu.Lock()
	_, ok := r.pending[id]
	r.mu.Unlock()
	return ok
}

// Len 返回当前待匹配请求数量。
func (r *ReplyPool[T]) Len() int {
	r.mu.Lock()
	n := len(r.pending)
	r.mu.Unlock()
	return n
}

// Close 关闭注册表，所有待匹配请求以 ErrReplyClosed 拒绝。
// 幂等调用安全。关闭后 Register 返回 ErrReplyClosed。
func (r *ReplyPool[T]) Close() {
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

	r.statsCancel()
	r.statsWG.Wait()
	r.tw.Stop()

	rejected := uint64(0)
	for _, e := range snapshot {
		e.promise.Reject(ErrReplyClosed)
		rejected++
	}
	r.deltaRejected.Add(rejected)
}

// RequestReply 是注册-发送-等待响应的便捷封装。
// sendFn 负责实际发送请求（如发 MQ 消息、写 TCP 帧），
// 如果 sendFn 失败，自动清理已注册的条目。
// ctx 用于控制等待超时。
// ttl 可选，未指定则使用 ReplyPool 的 defaultTTL；指定时覆盖本次请求的 TTL。
func RequestReply[T any](ctx context.Context, reg *ReplyPool[T], id string, sendFn func() error, ttl ...time.Duration) (T, error) {
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
