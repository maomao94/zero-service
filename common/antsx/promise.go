package antsx

import (
	"context"
	"fmt"
	"sync"
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

// Await 等待结果，支持多次调用（并发安全）
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	// 先检查缓存
	p.mu.Lock()
	if p.done {
		val, err := p.val, p.err
		p.mu.Unlock()
		return val, err
	}
	p.mu.Unlock()

	// 用 channel 作为信号，不使用接收到的值
	// Resolve/Reject 在发送前已将结果写入 p.val/p.err，
	// 这样多个并发 Await 都能正确读取缓存值
	select {
	case <-p.result:
		p.mu.Lock()
		val, err := p.val, p.err
		p.mu.Unlock()
		return val, err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Then 链式调用
func Then[T, U any](ctx context.Context, p *Promise[T], fn func(T) (U, error)) *Promise[U] {
	newPromise := NewPromise[U](p.id)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				newPromise.Reject(fmt.Errorf("antsx: Then panicked: %v", r))
			}
		}()

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
		p.mu.Unlock()

		p.result <- result[T]{val: val}
		close(p.result)
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
