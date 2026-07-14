package gnetx

import (
	"context"
	"reflect"
	"strconv"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

// RouteResolver 将消息解析为对应的业务 Handler。
type RouteResolver interface {
	Resolve(msg any) (Handler, error)
}

// Router 是基于 messageID 的路由表，同时实现 Handler 和 RouteResolver。
// 只负责消息→handler 的匹配，不关心 sync/async 执行模型。
// 异步标记通过注册时传入 Async(handler) 完成，执行调度由 Server/Client 的 dispatch 统一负责。
//
// 消息必须实现 Identifiable（MessageID() int），未匹配的消息回退到 fallback handler（若已配置）。
type Router struct {
	mu       sync.RWMutex
	handlers map[int]*routerEntry
	fallback *routerEntry
}

type routerEntry struct {
	handler Handler
}

// NewRouter 创建一个空的路由表。
func NewRouter() *Router {
	return &Router{
		handlers: make(map[int]*routerEntry),
	}
}

// Resolve 实现 RouteResolver，根据消息 ID 查找对应的 Handler。
func (r *Router) Resolve(msg any) (Handler, error) {
	entry, err := r.lookupEntry(msg)
	if err != nil {
		return nil, err
	}
	return entry.handler, nil
}

// Handle 实现 Handler，解析消息并委托给匹配的 handler 执行。
func (r *Router) Handle(ctx context.Context, conn Conn, msg any) (any, error) {
	h, err := r.Resolve(msg)
	if err != nil {
		return nil, err
	}
	return h.Handle(ctx, conn, msg)
}

func (r *Router) lookupEntry(msg any) (*routerEntry, error) {
	id, ok := messageIDOf(msg)
	if !ok {
		r.mu.RLock()
		fb := r.fallback
		r.mu.RUnlock()
		if fb != nil {
			return fb, nil
		}
		return nil, ErrNoHandler
	}
	r.mu.RLock()
	entry, found := r.handlers[id]
	fb := r.fallback
	r.mu.RUnlock()
	if !found {
		if fb != nil {
			return fb, nil
		}
		logx.Infof("[gnetx] no handler for message id=%d", id)
		return nil, ErrNoHandler
	}
	return entry, nil
}

// Register 为指定 messageID 注册一个 Handler。
func (r *Router) Register(id int, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[id] = &routerEntry{handler: h}
}

// RegisterFunc 为指定 messageID 注册一个处理函数。
func (r *Router) RegisterFunc(id int, fn func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.Register(id, HandlerFunc(fn))
}

// HandleTyped 为指定 messageID 注册一个带类型检查的处理函数。
func HandleTyped[T any](r *Router, id int, fn func(ctx context.Context, conn Conn, msg T) (any, error)) {
	r.Register(id, HandlerFunc(func(ctx context.Context, conn Conn, msg any) (any, error) {
		typed, ok := msg.(T)
		if !ok {
			return nil, errTypeMismatch(id, msg)
		}
		return fn(ctx, conn, typed)
	}))
}

// HandleTypedAsync 注册一个带类型检查的异步处理函数，等价于 HandleTyped + Async。
func HandleTypedAsync[T any](r *Router, id int, fn func(ctx context.Context, conn Conn, msg T) (any, error)) {
	HandleTyped(r, id, fn)
	r.Async(id)
}

// Async 将指定 messageID 的 handler 包装为 Async(handler)。
// 异步执行由 Server/Client 的 dispatch 通过 isAsync() 判断后统一调度。
func (r *Router) Async(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.handlers[id]; ok {
		entry.handler = Async(entry.handler)
	}
}

// Fallback 设置未匹配消息的兜底 Handler。
func (r *Router) Fallback(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = &routerEntry{handler: h}
}

// FallbackFunc 设置未匹配消息的兜底处理函数。
func (r *Router) FallbackFunc(fn func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.Fallback(HandlerFunc(fn))
}

// FallbackFuncAsync 设置未匹配消息的异步兜底处理函数。
func (r *Router) FallbackFuncAsync(fn func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.Fallback(Async(HandlerFunc(fn)))
}

func messageIDOf(msg any) (int, bool) {
	if id, ok := msg.(Identifiable); ok {
		return id.MessageID(), true
	}
	return 0, false
}

type typeMismatchErr struct {
	id  int
	got string
}

func (e *typeMismatchErr) Error() string {
	return "gnetx: handler for message id=" + itoa(e.id) + " got unexpected type " + e.got
}

func errTypeMismatch(id int, msg any) error {
	return &typeMismatchErr{id: id, got: typeName(msg)}
}

func typeName(msg any) string {
	if msg == nil {
		return "nil"
	}
	return reflect.TypeOf(msg).String()
}

func itoa(i int) string { return strconv.Itoa(i) }
