package antsx

import (
	"context"
	"sync"
	"time"
)

// Task 描述一个可并行执行的任务单元。
// T 为任务返回值的类型。
type Task[T any] struct {
	// Name 任务名称，用于 panic 错误信息中定位问题。
	Name string
	// Fn 任务执行函数，接收 ctx 以支持超时/取消控制。
	Fn func(ctx context.Context) (T, error)
	// Timeout 单任务超时时间。为 0 时使用 ctx 的全局超时。
	Timeout time.Duration
}

// Invoke 并行执行多个任务，返回按输入顺序排列的结果。
// 采用 fast-fail 策略：任何一个任务失败则立即通过 cancel 通知其他任务退出。
//
// 优化：单任务时在当前 goroutine 同步执行，多任务时第一个在当前 goroutine 执行，
// 其余在新 goroutine 中并行，避免不必要的 goroutine 开销（借鉴 eino 框架）。
//
// 所有 goroutine 都有 panic 恢复保护。
func Invoke[T any](ctx context.Context, tasks ...Task[T]) ([]T, error) {
	if len(tasks) == 0 {
		return []T{}, nil
	}

	results := make([]T, len(tasks))

	if len(tasks) == 1 {
		return invokeSingle(ctx, tasks[0], results)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	// 其余任务并行执行
	wg.Add(len(tasks) - 1)
	for i := 1; i < len(tasks); i++ {
		go func(idx int, t Task[T]) {
			defer func() {
				if r := recover(); r != nil {
					errOnce.Do(func() {
						firstErr = taskPanicErr(t.Name, r)
						cancel()
					})
				}
				wg.Done()
			}()
			val, err := runTask(ctx, t)
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			results[idx] = val
		}(i, tasks[i])
	}

	// 第一个任务在当前 goroutine 同步执行
	func() {
		t := tasks[0]
		defer func() {
			if r := recover(); r != nil {
				errOnce.Do(func() {
					firstErr = taskPanicErr(t.Name, r)
					cancel()
				})
			}
		}()
		val, err := runTask(ctx, t)
		if err != nil {
			errOnce.Do(func() {
				firstErr = err
				cancel()
			})
			return
		}
		results[0] = val
	}()

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

// InvokeCallback 执行所有任务后将结果传入 callback 进行聚合转换。
// 任意任务失败时不调用 callback，直接返回错误。
func InvokeCallback[T, U any](ctx context.Context, tasks []Task[T], callback func([]T) (U, error)) (U, error) {
	results, err := Invoke(ctx, tasks...)
	if err != nil {
		var zero U
		return zero, err
	}
	return callback(results)
}

// InvokeWithReactor 使用 Reactor 协程池并行执行多个任务。
// 与 Invoke 不同的是所有任务（包括第一个）都通过协程池调度，
// 适用于需要限制并发度的场景。
func InvokeWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) ([]T, error) {
	if len(tasks) == 0 {
		return []T{}, nil
	}

	results := make([]T, len(tasks))

	if len(tasks) == 1 {
		return invokeSingle(ctx, tasks[0], results)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	for i, task := range tasks {
		wg.Add(1)
		idx, t := i, task
		submitErr := r.Go(ctx, func(ctx context.Context) {
			defer func() {
				if p := recover(); p != nil {
					errOnce.Do(func() {
						firstErr = taskPanicErr(t.Name, p)
						cancel()
					})
				}
				wg.Done()
			}()
			val, err := runTask(ctx, t)
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

// SettledResult 表示单个任务的最终结果，无论成功或失败。
// 调用方通过检查 Err 是否为 nil 来判断任务状态。
type SettledResult[T any] struct {
	// Name 任务名称，与 Task.Name 对应。
	Name string
	// Val 任务成功时的返回值。失败时为零值。
	Val T
	// Err 任务失败时的错误。成功时为 nil。
	Err error
}

// Succeeded 返回任务是否执行成功。
func (r SettledResult[T]) Succeeded() bool { return r.Err == nil }

// InvokeAllSettled 并行执行所有任务，等待全部完成后返回每个任务的独立结果。
// 与 Invoke 不同，不使用 fast-fail 策略：每个任务独立运行，互不影响。
// 返回的 SettledResult 切片顺序与输入 tasks 一一对应。
//
// 适用于需要获取所有任务结果的场景，如批量查询、容错降级等。
func InvokeAllSettled[T any](ctx context.Context, tasks ...Task[T]) []SettledResult[T] {
	if len(tasks) == 0 {
		return []SettledResult[T]{}
	}

	results := make([]SettledResult[T], len(tasks))

	if len(tasks) == 1 {
		results[0] = settleTask(ctx, tasks[0])
		return results
	}

	var wg sync.WaitGroup
	wg.Add(len(tasks) - 1)
	for i := 1; i < len(tasks); i++ {
		go func(idx int, t Task[T]) {
			defer wg.Done()
			results[idx] = settleTask(ctx, t)
		}(i, tasks[i])
	}

	results[0] = settleTask(ctx, tasks[0])
	wg.Wait()
	return results
}

// InvokeAllSettledWithReactor 使用 Reactor 协程池并行执行所有任务，等待全部完成。
// 行为与 InvokeAllSettled 相同，但通过协程池限制并发度。
//
// 当协程池提交失败时，对应任务的 Err 为提交错误，后续任务不再提交。
func InvokeAllSettledWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) []SettledResult[T] {
	if len(tasks) == 0 {
		return []SettledResult[T]{}
	}

	results := make([]SettledResult[T], len(tasks))

	if len(tasks) == 1 {
		results[0] = settleTask(ctx, tasks[0])
		return results
	}

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		idx, t := i, task
		submitErr := r.Go(ctx, func(ctx context.Context) {
			defer wg.Done()
			results[idx] = settleTask(ctx, t)
		})
		if submitErr != nil {
			results[idx] = SettledResult[T]{Name: t.Name, Err: submitErr}
			wg.Done()
			for j := i + 1; j < len(tasks); j++ {
				results[j] = SettledResult[T]{Name: tasks[j].Name, Err: submitErr}
			}
			break
		}
	}

	wg.Wait()
	return results
}

// settleTask 执行单个任务并返回 SettledResult，包含 panic 恢复保护。
func settleTask[T any](ctx context.Context, t Task[T]) (sr SettledResult[T]) {
	sr.Name = t.Name
	defer func() {
		if r := recover(); r != nil {
			sr.Err = taskPanicErr(t.Name, r)
		}
	}()
	val, err := runTask(ctx, t)
	sr.Val = val
	sr.Err = err
	return
}

// runTask 执行单个 Task，如有 Timeout 则创建超时子 context。
func runTask[T any](ctx context.Context, t Task[T]) (T, error) {
	if t.Timeout > 0 {
		var taskCancel context.CancelFunc
		ctx, taskCancel = context.WithTimeout(ctx, t.Timeout)
		defer taskCancel()
	}
	return t.Fn(ctx)
}

// invokeSingle 单任务快捷执行路径，包含 panic 恢复。
func invokeSingle[T any](ctx context.Context, t Task[T], results []T) (ret []T, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			ret = nil
			retErr = taskPanicErr(t.Name, r)
		}
	}()
	val, err := runTask(ctx, t)
	if err != nil {
		return nil, err
	}
	results[0] = val
	return results, nil
}
