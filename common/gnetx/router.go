package gnetx

import (
	"context"
	"reflect"
	"strconv"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

// Router 是按 messageID 路由的 Handler 容器（opt-in）。
// 消息需实现 Identifiable（MessageID() int），Router 据此查找注册的 handler。
// 未命中 id 时走 fallback（若配置）；fallback 也未配置则返回 (nil, ErrNoHandler)。
//
// Router 本身实现 Handler，可直接传给 Server/Client 作为统一 handler。
// 纯推送协议可直接用 HandlerFunc 在单入口里 type-switch 分发，不用 Router。
//
// 并发安全：注册与查找可跨 goroutine。
//
// per-handler 异步：用 Router.Async(id) 标记单个 handler 后，
// Router.Handle 在查找该 handler 时检测到异步标记，自动 offload 到 worker pool 执行。
type Router struct {
	mu       sync.RWMutex
	handlers map[int]Handler
	fallback Handler
	types    map[int]func() any // 可选：按 id 注册消息工厂，供 decoder 实例化
	pool     *workerPool        // gnet worker pool，用于 per-handler 异步 offload
}

// NewRouter 创建空 Router，pool 用于 per-handler 异步 offload（Router.Async 标记的 handler）。
// pool 传入 nil 时，Router.Async 标记的 handler 仍会同步执行（无 worker pool 不可 offload）。
func NewRouter(pool *workerPool) *Router {
	return &Router{
		handlers: make(map[int]Handler),
		types:    make(map[int]func() any),
		pool:     pool,
	}
}

// Handle 实现 Handler。按 msg.(Identifiable).MessageID() 查找 handler。
// msg 未实现 Identifiable 时走 fallback；fallback 未配置返回 ErrNoHandler。
//
// 若查到的 handler 实现了 AsyncHandler 且 Router 持有 worker pool，
// 则 offload 到 pool 执行，回包走 AsyncWrite，方法立即返回 (nil, nil)。
func (r *Router) Handle(ctx context.Context, sess *Session, msg any) (any, error) {
	h, err := r.lookup(msg)
	if err != nil {
		return nil, err
	}
	// per-handler 异步
	if r.pool != nil && isAsync(h) {
		_ = r.pool.Submit(func() {
			reply, hErr := h.Handle(ctx, sess, msg)
			if hErr != nil {
				logx.Errorf("[gnetx] router async handler error: %v", hErr)
				return
			}
			if reply != nil {
				if err := sess.Send(reply); err != nil {
					logx.Errorf("[gnetx] router async reply error: %v", err)
				}
			}
		})
		return nil, nil
	}
	return h.Handle(ctx, sess, msg)
}

// lookup 根据消息找到对应 handler。msg 未实现 Identifiable 走 fallback。
func (r *Router) lookup(msg any) (Handler, error) {
	id, ok := messageIDOf(msg)
	if !ok {
		// 无 id 可路由，走 fallback。
		r.mu.RLock()
		fb := r.fallback
		r.mu.RUnlock()
		if fb != nil {
			return fb, nil
		}
		return nil, ErrNoHandler
	}
	r.mu.RLock()
	h, found := r.handlers[id]
	fb := r.fallback
	r.mu.RUnlock()
	if !found {
		if fb != nil {
			return fb, nil
		}
		logx.Infof("[gnetx] no handler for message id=%d", id)
		return nil, ErrNoHandler
	}
	return h, nil
}

// Register 注册一个 handler 处理指定 messageID。重复注册同一 id 会覆盖。
func (r *Router) Register(id int, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[id] = h
}

// RegisterFunc 是 Register 的便捷版本，直接传函数。
func (r *Router) RegisterFunc(id int, fn HandlerFunc) {
	r.Register(id, fn)
}

// HandleTyped 注册一个类型安全的 handler：fn 只接收具体类型 T。
// 运行时若 msg 不是 T 类型，返回 typeMismatchErr（不静默丢弃，便于排查协议/注册不一致）。
// 这是类型安全的注册方式，避免用户在 handler 里手写 type-assert。
func HandleTyped[T any](r *Router, id int, fn func(ctx context.Context, sess *Session, msg T) (any, error)) {
	r.Register(id, HandlerFunc(func(ctx context.Context, sess *Session, msg any) (any, error) {
		typed, ok := msg.(T)
		if !ok {
			return nil, errTypeMismatch(id, msg)
		}
		return fn(ctx, sess, typed)
	}))
}

// Async 把已注册的某个 id 的 handler 标记为异步执行（offload 到 gnet worker pool）。
// 等价于重新用 Async 包裹注册。
func (r *Router) Async(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.handlers[id]; ok {
		r.handlers[id] = Async(h)
	}
}

// Fallback 设置未命中 id 时的兜底 handler。
func (r *Router) Fallback(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = h
}

// FallbackFunc 是 Fallback 的便捷版本。
func (r *Router) FallbackFunc(fn HandlerFunc) {
	r.Fallback(fn)
}

// RegisterType 为指定 id 注册一个消息工厂，供自定义 decoder 按 wire id 实例化具体类型。
// 这是修复 netmc 需用户手写反射扫描痛点的 first-class 能力。
func (r *Router) RegisterType(id int, factory func() any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[id] = factory
}

// Factory 返回指定 id 的消息工厂；未注册返回 nil。
func (r *Router) Factory(id int) func() any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.types[id]
}

// messageIDOf 尝试从 msg 提取 messageID。返回 (id, true) 或 (0, false)。
func messageIDOf(msg any) (int, bool) {
	if id, ok := msg.(Identifiable); ok {
		return id.MessageID(), true
	}
	return 0, false
}

// errTypeMismatch 构造类型不匹配错误。
func errTypeMismatch(id int, msg any) error {
	return &typeMismatchErr{id: id, got: typeName(msg)}
}

// typeMismatchErr 表示 HandleTyped 的运行时类型断言失败。
type typeMismatchErr struct {
	id  int
	got string
}

func (e *typeMismatchErr) Error() string {
	return "gnetx: handler for message id=" + itoa(e.id) + " got unexpected type " + e.got
}

// messageIDOf 和 typeName 的辅助放在 router.go 内避免新增文件。

// typeName 返回 msg 的可读类型名（用于错误信息）。
func typeName(msg any) string {
	if msg == nil {
		return "nil"
	}
	t := reflect.TypeOf(msg)
	if t.Name() == "" {
		return t.String()
	}
	return t.String()
}

// itoa 把 int 转字符串（strconv.Itoa 的薄封装，避免在多处直接 import strconv）。
func itoa(i int) string {
	return strconv.Itoa(i)
}
