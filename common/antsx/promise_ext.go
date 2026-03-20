package antsx

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PromiseAll 并发等待所有 Promise 完成，任一失败快速返回
func PromiseAll[T any](ctx context.Context, promises ...*Promise[T]) ([]T, error) {
	results := make([]T, len(promises))
	var (
		errOnce sync.Once
		retErr  error
		wg      sync.WaitGroup
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(len(promises))
	for i, p := range promises {
		go func(idx int, pr *Promise[T]) {
			defer wg.Done()
			val, err := pr.Await(ctx)
			if err != nil {
				errOnce.Do(func() {
					retErr = err
					cancel() // 快速失败，取消其他等待
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

// PromiseRace 竞争，返回第一个完成的 Promise 结果
func PromiseRace[T any](ctx context.Context, promises ...*Promise[T]) (T, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type raceResult struct {
		val T
		err error
	}

	ch := make(chan raceResult, len(promises))
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

// Map 纯映射转换（映射函数不返回 error）
func Map[T, U any](ctx context.Context, p *Promise[T], fn func(T) U) *Promise[U] {
	newPromise := NewPromise[U](p.id)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				newPromise.Reject(fmt.Errorf("antsx: Map panicked: %v", r))
			}
		}()

		val, err := p.Await(ctx)
		if err != nil {
			newPromise.Reject(err)
			return
		}
		newPromise.Resolve(fn(val))
	}()

	return newPromise
}

// FlatMap 扁平化映射，映射函数返回新的 Promise
func FlatMap[T, U any](ctx context.Context, p *Promise[T], fn func(T) *Promise[U]) *Promise[U] {
	newPromise := NewPromise[U](p.id)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				newPromise.Reject(fmt.Errorf("antsx: FlatMap panicked: %v", r))
			}
		}()

		val, err := p.Await(ctx)
		if err != nil {
			newPromise.Reject(err)
			return
		}

		innerPromise := fn(val)
		innerVal, innerErr := innerPromise.Await(ctx)
		if innerErr != nil {
			newPromise.Reject(innerErr)
			return
		}
		newPromise.Resolve(innerVal)
	}()

	return newPromise
}

// AwaitWithTimeout 带超时的 Await
func (p *Promise[T]) AwaitWithTimeout(timeout time.Duration) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.Await(ctx)
}
