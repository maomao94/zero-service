package antsx

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/zeromicro/go-zero/core/threading"
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
// 采用 fast-fail 策略：任何一个任务失败则通过 errgroup 自动取消其他任务。
//
// 所有 goroutine 都有 panic 恢复保护。
func Invoke[T any](ctx context.Context, tasks ...Task[T]) ([]T, error) {
	if len(tasks) == 0 {
		return []T{}, nil
	}

	if len(tasks) == 1 {
		val, err := invokeSingle(ctx, tasks[0])
		if err != nil {
			return nil, err
		}
		return []T{val}, nil
	}

	results := make([]T, len(tasks))
	g, groupCtx := errgroup.WithContext(ctx)

	for i, task := range tasks {
		idx, t := i, task
		g.Go(func() error {
			val, err := invokeSingle(groupCtx, t)
			if err != nil {
				return err
			}
			results[idx] = val
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
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
// 采用 fast-fail 策略：任何一个任务失败则取消其他任务。
// 注意：不使用 errgroup，因为 goroutine 由 ants 池而非 errgroup 调度，
// errgroup 的 g.Go 会阻塞在 pool.Submit 上，无法响应 ctx 取消。
// 适用于需要限制并发度的场景。
func InvokeWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) ([]T, error) {
	if len(tasks) == 0 {
		return []T{}, nil
	}

	if len(tasks) == 1 {
		val, err := invokeSingle(ctx, tasks[0])
		if err != nil {
			return nil, err
		}
		return []T{val}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]T, len(tasks))
	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	for i, task := range tasks {
		if ctx.Err() != nil {
			break
		}
		idx, t := i, task
		wg.Add(1)
		submitErr := r.Go(ctx, func(poolCtx context.Context) {
			defer wg.Done()
			val, err := invokeSingle(poolCtx, t)
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
	if ctx.Err() != nil {
		return nil, ctx.Err()
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

// initSettledResults 处理空切片检查和单任务优化，返回 (results, done)。
func initSettledResults[T any](ctx context.Context, tasks []Task[T]) ([]SettledResult[T], bool) {
	if len(tasks) == 0 {
		return []SettledResult[T]{}, true
	}

	results := make([]SettledResult[T], len(tasks))

	if len(tasks) == 1 {
		results[0] = settleTask(ctx, tasks[0])
		return results, true
	}

	return results, false
}

// InvokeAllSettled 并行执行所有任务，等待全部完成后返回每个任务的独立结果。
// 与 Invoke 不同，不使用 fast-fail 策略：每个任务独立运行，互不影响。
// 返回的 SettledResult 切片顺序与输入 tasks 一一对应。
//
// 适用于需要获取所有任务结果的场景，如批量查询、容错降级等。
func InvokeAllSettled[T any](ctx context.Context, tasks ...Task[T]) []SettledResult[T] {
	results, done := initSettledResults(ctx, tasks)
	if done {
		return results
	}

	var wg sync.WaitGroup
	wg.Add(len(tasks) - 1)
	for i := 1; i < len(tasks); i++ {
		idx, t := i, tasks[i]
		results[idx] = SettledResult[T]{Name: t.Name}
		threading.GoSafe(func() {
			defer wg.Done()
			results[idx] = settleTask(ctx, t)
		})
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
	results, done := initSettledResults(ctx, tasks)
	if done {
		return results
	}

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		idx, t := i, task
		results[idx] = SettledResult[T]{Name: t.Name}
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

// settleTask 执行单个任务并返回 SettledResult。
func settleTask[T any](ctx context.Context, t Task[T]) (sr SettledResult[T]) {
	sr.Name = t.Name
	sr.Val, sr.Err = invokeSingle(ctx, t)
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
func invokeSingle[T any](ctx context.Context, t Task[T]) (val T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = taskPanicErr(t.Name, r)
		}
	}()
	val, err = runTask(ctx, t)
	return
}
