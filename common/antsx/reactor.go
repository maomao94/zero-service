package antsx

import (
	"context"
	"log"
	"runtime/debug"

	"github.com/panjf2000/ants/v2"
)

// Reactor 是基于 ants 协程池的任务调度器，用于控制并发 goroutine 数量。
// 在高并发场景下，通过 Reactor 提交任务可以避免创建过多 goroutine 导致资源耗尽。
//
// 协程安全：所有方法均可在多个 goroutine 中并发调用。
type Reactor struct {
	pool *ants.Pool
}

// NewReactor 创建一个指定容量的 Reactor。
// size 为最大并发 goroutine 数量；size <= 0 时 ants 将使用默认池大小。
func NewReactor(size int) (*Reactor, error) {
	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}
	return &Reactor{pool: pool}, nil
}

// Submit 将一个带返回值的异步任务提交到 Reactor 协程池执行，返回代表结果的 Promise。
// fn 中的 panic 会被捕获并转换为 Promise 的 error，不会泄漏到调用方。
// 如果协程池已满或已关闭，返回非 nil error。
func Submit[T any](ctx context.Context, r *Reactor, fn func(ctx context.Context) (T, error)) (*Promise[T], error) {
	p := NewPromise[T]()
	err := r.pool.Submit(func() {
		defer func() {
			if v := recover(); v != nil {
				p.Reject(newPanicErr(v))
			}
		}()
		val, fnErr := fn(ctx)
		if fnErr != nil {
			p.Reject(fnErr)
		} else {
			p.Resolve(val)
		}
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Post 将一个带 context 的任务提交到 Reactor 协程池执行。
// fn 中的 panic 会被内部恢复，不会导致进程崩溃。
// 如果协程池已满或已关闭，返回非 nil error。
func Post(ctx context.Context, r *Reactor, fn func(ctx context.Context) error) error {
	return r.pool.Submit(func() {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("antsx: Post panic recovered: %v\n%s", p, debug.Stack())
			}
		}()
		_ = fn(ctx)
	})
}

// Go 将一个函数提交到协程池执行。
// ctx 用于传递上下文（如超时/取消），fn 内部应检查 ctx.Done() 以响应取消。
// 内置 panic 恢复保护，fn 中的 panic 不会导致进程崩溃。
// 如果协程池已满或已关闭，返回非 nil error。
func (r *Reactor) Go(ctx context.Context, fn func(ctx context.Context)) error {
	return r.pool.Submit(func() {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("antsx: Reactor.Go panic recovered: %v\n%s", p, debug.Stack())
			}
		}()
		fn(ctx)
	})
}

// Release 释放协程池资源。释放后不应再提交新任务。
func (r *Reactor) Release() {
	r.pool.Release()
}

// ActiveCount 返回当前正在执行任务的 goroutine 数量。
func (r *Reactor) ActiveCount() int {
	return r.pool.Running()
}
