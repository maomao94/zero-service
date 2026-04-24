package antsx

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Promise 是一个泛型异步结果容器，表示一个未来可用的值。
//
// Promise 通过 Resolve 或 Reject 设定最终结果（仅第一次有效，基于 sync.Once），
// 调用方通过 Await 阻塞等待结果，支持 context 超时/取消控制。
//
// 协程安全：所有方法均可在多个 goroutine 中并发调用。
type Promise[T any] struct {
	done chan struct{}
	once sync.Once
	val  T
	err  error
}

// NewPromise 创建一个处于 pending 状态的 Promise。
// 调用方需通过 Resolve 或 Reject 将其标记为已完成。
func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{done: make(chan struct{})}
}

// Resolve 以成功值 val 完成 Promise。
// 仅第一次调用生效，后续调用（包括 Reject）静默忽略。
func (p *Promise[T]) Resolve(val T) {
	p.once.Do(func() {
		p.val = val
		close(p.done)
	})
}

// Reject 以错误 err 完成 Promise。
// 仅第一次调用生效，后续调用（包括 Resolve）静默忽略。
func (p *Promise[T]) Reject(err error) {
	p.once.Do(func() {
		p.err = err
		close(p.done)
	})
}

// Await 阻塞等待 Promise 完成并返回结果。
// 如果 ctx 先于 Promise 完成被取消或超时，则返回 ctx.Err()。
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	select {
	case <-p.done:
		return p.val, p.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Done 返回一个在 Promise 完成时关闭的 channel，用于 select 监听。
func (p *Promise[T]) Done() <-chan struct{} {
	return p.done
}

// Get 非阻塞地检查 Promise 当前状态。
// 如果 Promise 已完成，返回 (val, err, true)；否则返回零值和 ok=false。
func (p *Promise[T]) Get() (val T, err error, ok bool) {
	select {
	case <-p.done:
		return p.val, p.err, true
	default:
		return val, nil, false
	}
}

// Catch 注册一个错误回调，当 Promise 以错误完成时在独立 goroutine 中调用 fn。
// ctx 用于控制回调 goroutine 的生命周期：如果 ctx 先于 Promise 取消，goroutine 自动退出，
// 防止因 Promise 永远不完成而导致 goroutine 泄漏。
func (p *Promise[T]) Catch(ctx context.Context, fn func(error)) {
	go func() {
		select {
		case <-p.done:
			if p.err != nil {
				fn(p.err)
			}
		case <-ctx.Done():
		}
	}()
}

// AwaitWithTimeout 是 Await 的便捷封装，使用指定的 timeout 创建临时 context。
func (p *Promise[T]) AwaitWithTimeout(timeout time.Duration) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.Await(ctx)
}

// FireAndForget 在后台 goroutine 中等待 Promise 完成但忽略结果。
// 典型用途：确保 Promise 关联的资源被正确清理。
// 内部包含 panic 恢复，不会因为 Await 后的操作崩溃而泄漏。
func (p *Promise[T]) FireAndForget(ctx context.Context) {
	go func() {
		defer func() { recover() }()
		_, _ = p.Await(ctx)
	}()
}

// PromiseAll 等待所有 Promise 完成并按顺序返回结果。
// 如果任何一个 Promise 失败（或 ctx 取消），立即取消剩余等待并返回第一个错误。
// 类似 JavaScript 的 Promise.all()。
func PromiseAll[T any](ctx context.Context, promises ...*Promise[T]) ([]T, error) {
	if len(promises) == 0 {
		return []T{}, nil
	}

	results := make([]T, len(promises))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		retErr  error
	)

	wg.Add(len(promises))
	for i, p := range promises {
		go func(idx int, pr *Promise[T]) {
			defer wg.Done()
			val, err := pr.Await(ctx)
			if err != nil {
				errOnce.Do(func() {
					retErr = err
					cancel()
				})
				return
			}
			results[idx] = val
		}(i, p)
	}

	wg.Wait()
	if retErr != nil {
		return nil, retErr
	}
	return results, nil
}

// PromiseRace 等待第一个完成的 Promise 并返回其结果。
// 第一个 Promise 完成后，通过 cancel 通知其他等待 goroutine 退出。
// 如果 promises 为空，返回 context.Canceled。
// 类似 JavaScript 的 Promise.race()。
func PromiseRace[T any](ctx context.Context, promises ...*Promise[T]) (T, error) {
	if len(promises) == 0 {
		var zero T
		return zero, context.Canceled
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type raceResult struct {
		val T
		err error
	}

	ch := make(chan raceResult, 1)
	for _, p := range promises {
		go func(pr *Promise[T]) {
			val, err := pr.Await(ctx)
			select {
			case ch <- raceResult{val: val, err: err}:
			default:
			}
		}(p)
	}

	select {
	case r := <-ch:
		return r.val, r.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Then 对 Promise 的成功值执行转换函数 fn，返回一个新的 Promise。
// 如果源 Promise 失败或 fn 返回错误，新 Promise 将以该错误 Reject。
// fn 中的 panic 会被捕获并转换为 error。
func Then[T, U any](ctx context.Context, p *Promise[T], fn func(T) (U, error)) *Promise[U] {
	next := NewPromise[U]()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				next.Reject(newPanicErr(r))
			}
		}()
		val, err := p.Await(ctx)
		if err != nil {
			next.Reject(err)
			return
		}
		newVal, err := fn(val)
		if err != nil {
			next.Reject(err)
			return
		}
		next.Resolve(newVal)
	}()
	return next
}

// Map 对 Promise 的成功值执行纯转换函数 fn（无 error 返回），返回新 Promise。
// fn 中的 panic 会被捕获并转换为 error。
func Map[T, U any](ctx context.Context, p *Promise[T], fn func(T) U) *Promise[U] {
	next := NewPromise[U]()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				next.Reject(newPanicErr(r))
			}
		}()
		val, err := p.Await(ctx)
		if err != nil {
			next.Reject(err)
			return
		}
		next.Resolve(fn(val))
	}()
	return next
}

// FlatMap 对 Promise 的成功值执行返回另一个 Promise 的函数 fn，实现链式异步操作。
// 等价于 Then + 自动展开内层 Promise。fn 中的 panic 会被捕获并转换为 error。
func FlatMap[T, U any](ctx context.Context, p *Promise[T], fn func(T) *Promise[U]) *Promise[U] {
	next := NewPromise[U]()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				next.Reject(newPanicErr(r))
			}
		}()
		val, err := p.Await(ctx)
		if err != nil {
			next.Reject(err)
			return
		}
		inner := fn(val)
		innerVal, innerErr := inner.Await(ctx)
		if innerErr != nil {
			next.Reject(innerErr)
			return
		}
		next.Resolve(innerVal)
	}()
	return next
}

// Go 在新 goroutine 中执行 fn 并返回代表其结果的 Promise。
// fn 中的 panic 会被捕获并转换为 error。
//
// 用法:
//
//	p := antsx.Go(ctx, func(ctx context.Context) (int, error) {
//	    return computeExpensiveResult(ctx)
//	})
//	val, err := p.Await(ctx)
func Go[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) *Promise[T] {
	p := NewPromise[T]()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.Reject(newPanicErr(r))
			}
		}()
		val, err := fn(ctx)
		if err != nil {
			p.Reject(err)
		} else {
			p.Resolve(val)
		}
	}()
	return p
}

// taskPanicErr 将 task 中捕获的 panic 包装为带任务名的错误。
func taskPanicErr(name string, r any) error {
	return fmt.Errorf("antsx: task %q panicked: %w", name, newPanicErr(r))
}
