package antsx

import (
	"context"
	"errors"
	"sync"
	"time"

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

// invokeState 封装 Invoke 系列函数的共享状态。
type invokeState[T any] struct {
	ctx     context.Context
	cancel  context.CancelFunc
	results []T
	mu      sync.Mutex
	errs    []error
	wg      sync.WaitGroup
}

func newInvokeState[T any](ctx context.Context, n int) *invokeState[T] {
	ctx, cancel := context.WithCancel(ctx)
	return &invokeState[T]{
		ctx:     ctx,
		cancel:  cancel,
		results: make([]T, n),
	}
}

func (s *invokeState[T]) addErr(err error) {
	s.mu.Lock()
	s.errs = append(s.errs, err)
	s.mu.Unlock()
	s.cancel()
}

func (s *invokeState[T]) wait() ([]T, error) {
	s.wg.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.errs) > 0 {
		return nil, errors.Join(s.errs...)
	}
	return s.results, nil
}

// runTaskWithRecovery 执行单个任务，包含 panic 恢复和 wg 管理。
func (s *invokeState[T]) runTaskWithRecovery(idx int, t Task[T]) {
	defer func() {
		if r := recover(); r != nil {
			s.addErr(taskPanicErr(t.Name, r))
		}
		s.wg.Done()
	}()
	val, err := runTask(s.ctx, t)
	if err != nil {
		s.addErr(err)
		return
	}
	s.results[idx] = val
}

func (s *invokeState[T]) goTask(idx int, t Task[T]) {
	s.wg.Add(1)
	threading.GoSafe(func() {
		s.runTaskWithRecovery(idx, t)
	})
}

func (s *invokeState[T]) runTaskSync(idx int, t Task[T]) {
	s.wg.Add(1)
	s.runTaskWithRecovery(idx, t)
}

func (s *invokeState[T]) submitTask(r *Reactor, idx int, t Task[T]) error {
	s.wg.Add(1)
	err := r.Go(s.ctx, func(ctx context.Context) {
		s.runTaskWithRecovery(idx, t)
	})
	if err != nil {
		s.wg.Done()
		return err
	}
	return nil
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

	if len(tasks) == 1 {
		val, err := invokeSingle(ctx, tasks[0])
		if err != nil {
			return nil, err
		}
		return []T{val}, nil
	}

	s := newInvokeState[T](ctx, len(tasks))
	defer s.cancel()

	// 其余任务并行执行
	for i := 1; i < len(tasks); i++ {
		s.goTask(i, tasks[i])
	}

	// 第一个任务在当前 goroutine 同步执行，放在最后启动
	// 这样其他任务的错误可以先触发 cancel，让第一个任务检测到 ctx.Done()
	s.runTaskSync(0, tasks[0])

	return s.wait()
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

	if len(tasks) == 1 {
		val, err := invokeSingle(ctx, tasks[0])
		if err != nil {
			return nil, err
		}
		return []T{val}, nil
	}

	s := newInvokeState[T](ctx, len(tasks))
	defer s.cancel()

	for i, task := range tasks {
		err := s.submitTask(r, i, task)
		if err != nil {
			s.addErr(err)
			break
		}
	}

	return s.wait()
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

	results[0] = SettledResult[T]{Name: tasks[0].Name}
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
func invokeSingle[T any](ctx context.Context, t Task[T]) (val T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = taskPanicErr(t.Name, r)
		}
	}()
	val, err = runTask(ctx, t)
	return
}
