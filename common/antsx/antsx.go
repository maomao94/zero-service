package antsx

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"sync"

	"github.com/panjf2000/ants/v2"
)

// ---------------------- Promise ----------------------

type result[T any] struct {
	val T
	err error
}

// Promise 泛型响应式对象
type Promise[T any] struct {
	id string

	result chan result[T]
	once   sync.Once
	mu     sync.Mutex

	val  T
	err  error
	done bool

	catchFunc func(error)
}

// NewPromise 创建 Promise
func NewPromise[T any](id string) *Promise[T] {
	return &Promise[T]{
		id:     id,
		result: make(chan result[T], 1),
	}
}

// Await 等待结果，支持多次调用
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	// 先检查缓存
	p.mu.Lock()
	if p.done {
		val, err := p.val, p.err
		p.mu.Unlock()
		return val, err
	}
	p.mu.Unlock()

	select {
	case r := <-p.result:
		p.mu.Lock()
		p.val, p.err = r.val, r.err
		p.done = true
		p.mu.Unlock()
		return r.val, r.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Then 链式调用
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

// Catch 注册错误回调
func (p *Promise[T]) Catch(fn func(error)) *Promise[T] {
	p.mu.Lock()
	alreadyRejected := p.done && p.err != nil
	err := p.err
	p.catchFunc = fn
	p.mu.Unlock()

	if alreadyRejected && err != nil {
		go fn(err)
	}

	return p
}

// Resolve 内部调用，保证只触发一次
func (p *Promise[T]) Resolve(val T) {
	p.once.Do(func() {
		p.mu.Lock()
		p.val = val
		p.err = nil
		p.done = true
		catchFn := p.catchFunc
		p.mu.Unlock()

		p.result <- result[T]{val: val}
		close(p.result)

		if catchFn != nil {
			// 成功不触发 catch
		}
	})
}

// Reject 内部调用，保证只触发一次
func (p *Promise[T]) Reject(err error) {
	p.once.Do(func() {
		p.mu.Lock()
		p.err = err
		p.done = true
		catchFn := p.catchFunc
		p.mu.Unlock()

		p.result <- result[T]{err: err}
		close(p.result)

		if catchFn != nil {
			go catchFn(err)
		}
	})
}

// FireAndForget 异步消费结果，不关心返回值
func (p *Promise[T]) FireAndForget(ctx context.Context) {
	go func() {
		_, _ = p.Await(ctx)
	}()
}

// ---------------------- Reactor ----------------------

type Reactor struct {
	pool     *ants.Pool
	registry *sync.Map
}

// NewReactor 创建 Reactor
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

// Submit 提交带 id 的任务，返回 Promise[T]
func Submit[T any](ctx context.Context, r *Reactor, id string, task func(ctx context.Context) (T, error)) (*Promise[T], error) {
	if _, loaded := r.registry.LoadOrStore(id, struct{}{}); loaded {
		return nil, fmt.Errorf("promise id %s already exists", id)
	}

	promise := NewPromise[T](id)

	err := r.pool.Submit(func() {
		defer r.registry.Delete(id)

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

// Post 提交 fire-and-forget 任务
func Post[T any](ctx context.Context, r *Reactor, task func(ctx context.Context) (T, error)) error {
	return r.pool.Submit(func() {
		_, err := task(ctx)
		if err != nil {
			logx.WithContext(ctx).Errorf("task error: %v", err)
		}
	})
}

// Release 释放池
func (r *Reactor) Release() {
	r.pool.Release()
}

// ActiveCount 当前运行任务数
func (r *Reactor) ActiveCount() int {
	return r.pool.Running()
}
