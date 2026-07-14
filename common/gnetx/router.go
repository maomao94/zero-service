package gnetx

import (
	"context"
	"reflect"
	"strconv"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

// Router is a message-ID-based dispatcher implementing Handler.
// Registered handlers receive a Conn, satisfied by both ServerConn and ClientConn,
// enabling the same handler registration to work on both sides.
//
// Messages must implement Identifiable (MessageID() int). Unmatched messages
// fall back to a fallback handler if configured. Per-handler async offload is
// enabled via Router.Async(id).
type Router struct {
	mu       sync.RWMutex
	handlers map[int]*routerEntry
	fallback *routerEntry
	types    map[int]func() any
	pool     *workerPool
}

type routerEntry struct {
	handle func(ctx context.Context, conn Conn, msg any) (any, error)
	async  bool
}

func NewRouter(pool *workerPool) *Router {
	return &Router{
		handlers: make(map[int]*routerEntry),
		types:    make(map[int]func() any),
		pool:     pool,
	}
}

func (r *Router) Handle(ctx context.Context, conn Conn, msg any) (any, error) {
	entry, err := r.lookupEntry(msg)
	if err != nil {
		return nil, err
	}
	if r.pool != nil && entry.async {
		_ = r.pool.Submit(func() {
			reply, hErr := entry.handle(ctx, conn, msg)
			if hErr != nil {
				logx.Errorf("[gnetx] router async handler error: %v", hErr)
				return
			}
			if reply != nil {
				if err := conn.Send(ctx, reply); err != nil {
					logx.Errorf("[gnetx] router async reply error: %v", err)
				}
			}
		})
		return nil, nil
	}
	return entry.handle(ctx, conn, msg)
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

func (r *Router) Register(id int, h func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[id] = &routerEntry{handle: h}
}

func (r *Router) RegisterFunc(id int, fn func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.Register(id, fn)
}

func HandleTyped[T any](r *Router, id int, fn func(ctx context.Context, conn Conn, msg T) (any, error)) {
	r.Register(id, func(ctx context.Context, conn Conn, msg any) (any, error) {
		typed, ok := msg.(T)
		if !ok {
			return nil, errTypeMismatch(id, msg)
		}
		return fn(ctx, conn, typed)
	})
}

// HandleTypedAsync 注册 typed handler 并标记为异步执行，等价于 HandleTyped + Async。
func HandleTypedAsync[T any](r *Router, id int, fn func(ctx context.Context, conn Conn, msg T) (any, error)) {
	HandleTyped(r, id, fn)
	r.Async(id)
}

func (r *Router) Async(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.handlers[id]; ok {
		entry.async = true
	}
}

func (r *Router) Fallback(h func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = &routerEntry{handle: h}
}

func (r *Router) FallbackFunc(fn func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.Fallback(fn)
}

func (r *Router) FallbackFuncAsync(fn func(ctx context.Context, conn Conn, msg any) (any, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = &routerEntry{handle: fn, async: true}
}

func (r *Router) RegisterType(id int, factory func() any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[id] = factory
}

func (r *Router) Factory(id int) func() any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.types[id]
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
