package antsx

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task 流程编排中的单个任务定义
type Task[T any] struct {
	Name    string                                // 任务名称，用于日志/调试
	Fn      func(ctx context.Context) (T, error)  // 任务执行函数
	Timeout time.Duration                         // 单任务超时，0 表示使用调用方 ctx
}

// Invoke 并行执行多个任务，返回有序结果
// 任一任务失败或 panic 立即取消其余任务（fast-fail）
// 整体超时由调用方 ctx 控制
func Invoke[T any](ctx context.Context, tasks ...Task[T]) ([]T, error) {
	if len(tasks) == 0 {
		return []T{}, nil
	}

	results := make([]T, len(tasks))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		firstErr error
	)

	wg.Add(len(tasks))
	for i, task := range tasks {
		go func(idx int, t Task[T]) {
			defer func() {
				if r := recover(); r != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("antsx: task %q panicked: %v", t.Name, r)
						cancel()
					})
				}
				wg.Done()
			}()

			taskCtx := ctx
			if t.Timeout > 0 {
				var taskCancel context.CancelFunc
				taskCtx, taskCancel = context.WithTimeout(ctx, t.Timeout)
				defer taskCancel()
			}

			val, err := t.Fn(taskCtx)
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			results[idx] = val
		}(i, task)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

// InvokeCallback 并行执行任务后，用 callback 变换聚合结果
// 如果 Invoke 阶段失败，callback 不会被调用
func InvokeCallback[T, U any](ctx context.Context, tasks []Task[T], callback func([]T) (U, error)) (U, error) {
	results, err := Invoke(ctx, tasks...)
	if err != nil {
		var zero U
		return zero, err
	}
	return callback(results)
}

// InvokeWithReactor 通过 Reactor 池执行并行任务
// 与 Invoke 行为一致，但任务提交到 ants 池而非裸 goroutine
func InvokeWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) ([]T, error) {
	if len(tasks) == 0 {
		return []T{}, nil
	}

	results := make([]T, len(tasks))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	wg.Add(len(tasks))
	for i, task := range tasks {
		idx, t := i, task
		submitErr := r.Go(func() {
			defer func() {
				if p := recover(); p != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("antsx: task %q panicked: %v", t.Name, p)
						cancel()
					})
				}
				wg.Done()
			}()

			taskCtx := ctx
			if t.Timeout > 0 {
				var taskCancel context.CancelFunc
				taskCtx, taskCancel = context.WithTimeout(ctx, t.Timeout)
				defer taskCancel()
			}

			val, err := t.Fn(taskCtx)
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			results[idx] = val
		})

		if submitErr != nil {
			wg.Done()
			errOnce.Do(func() {
				firstErr = submitErr
				cancel()
			})
			break
		}
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}
