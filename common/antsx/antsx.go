package antsx

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"sync"

	"github.com/panjf2000/ants/v2"
)

// Promise 是泛型响应式对象
type Promise[T any] struct {
	id        string
	result    chan result[T]
	once      sync.Once
	mu        sync.Mutex
	err       error
	catchFunc func(error)
}

type result[T any] struct {
	val T
	err error
}

func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	select {
	case r := <-p.result:
		return r.val, r.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

func Then[T, U any](ctx context.Context, p *Promise[T], fn func(T) (U, error)) *Promise[U] {
	newPromise := NewPromise[U](p.id)

	go func() {
		val, err := p.Await(ctx)
		if err != nil {
			newPromise.Reject(err)
			return
		}

		newVal, err := fn(val)
		if err != nil {
			newPromise.Reject(err)
			return
		}

		newPromise.Resolve(newVal)
	}()

	return newPromise
}

func (p *Promise[T]) Catch(fn func(error)) *Promise[T] {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.catchFunc = fn
	if p.err != nil {
		go fn(p.err)
	}
	return p
}

func (p *Promise[T]) Resolve(val T) {
	p.once.Do(func() {
		p.result <- result[T]{val: val}
	})
}

func (p *Promise[T]) Reject(err error) {
	p.once.Do(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.err = err
		p.result <- result[T]{err: err}
		if p.catchFunc != nil {
			go p.catchFunc(err)
		}
	})
}

func (p *Promise[T]) FireAndForget() {
	go func() {
		_, _ = p.Await(context.Background())
	}()
}

func NewPromise[T any](id string) *Promise[T] {
	return &Promise[T]{
		id:     id,
		result: make(chan result[T], 1),
	}
}

// Reactor 非泛型，统一管理池和注册表
type Reactor struct {
	pool     *ants.Pool
	registry *sync.Map
}

func NewReactor(size int) (*Reactor, error) {
	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}
	return &Reactor{
		pool:     pool,
		registry: &sync.Map{},
	}, nil
}

// Submit 泛型方法，支持任意类型任务
func Submit[T any](ctx context.Context, r *Reactor, id string, task func(ctx context.Context) (T, error)) (*Promise[T], error) {
	if _, loaded := r.registry.LoadOrStore(id, struct{}{}); loaded {
		return nil, fmt.Errorf("promise id %s already exists", id)
	}

	promise := NewPromise[T](id)

	err := r.pool.Submit(func() {
		defer r.registry.Delete(id)

		// 这里传递 ctx 给 task
		val, err := task(ctx)
		if err != nil {
			promise.Reject(err)
		} else {
			promise.Resolve(val)
		}
	})

	if err != nil {
		r.registry.Delete(id)
		return nil, err
	}

	return promise, nil
}

// Post 任务
func Post[T any](ctx context.Context, r *Reactor, task func(ctx context.Context) (T, error)) error {

	err := r.pool.Submit(func() {
		// 这里传递 ctx 给 task
		_, err := task(ctx)
		if err != nil {
			logx.WithContext(ctx).Errorf("task error: %v", err)
		}
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *Reactor) Release() {
	r.pool.Release()
}

func (r *Reactor) ActiveCount() int {
	return r.pool.Running()
}
