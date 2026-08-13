package crontask

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"zero-service/common/rrulex"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Handler 业务回调函数，调度器按时触发后调用此函数。
// 返回 nil 表示执行成功，调度器计算下次时间继续调度。
// 返回 error 表示失败，不更新 next_run，任务可被后续扫描重试。
type Handler func(ctx context.Context, task *TaskConfig) error

// Scheduler 通用周期性任务调度器，依赖 TaskStore 接口实现存储无关。
// 主循环使用自适应 sleep：有任务 10ms 快速连扫，无任务 interval 间隔等待。
type Scheduler struct {
	store                TaskStore
	handler              Handler
	interval             time.Duration
	lockExpire           time.Duration
	maxDelay             time.Duration
	stopCh               chan struct{}
	startOnce            sync.Once
	stopOnce             sync.Once
	workerGroup          sync.WaitGroup
	tracer               oteltrace.Tracer
	invalidTimePredicate InvalidTimePredicate
	guard                Guard
}

// NewScheduler 创建调度器，默认扫描间隔 2 秒，lease 过期时间 5 分钟。
func NewScheduler(store TaskStore, handler Handler, opts ...SchedulerOption) *Scheduler {
	o := &SchedulerOptions{
		Interval:   2 * time.Second,
		LockExpire: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &Scheduler{
		store:                store,
		handler:              handler,
		interval:             o.Interval,
		lockExpire:           o.LockExpire,
		maxDelay:             o.MaxDelay,
		stopCh:               make(chan struct{}),
		tracer:               otel.Tracer(trace.TraceName),
		invalidTimePredicate: o.InvalidTimePredicate,
		guard:                o.Guard,
	}
}

// Start 启动调度器主循环。
func (s *Scheduler) Start() {
	s.startOnce.Do(func() {
		logx.Info("[crontask] scheduler started")
		s.workerGroup.Add(1)
		go func() {
			defer s.workerGroup.Done()
			s.scanLoop()
		}()
	})
}

// Stop 停止调度器。
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.workerGroup.Wait()
		logx.Info("[crontask] scheduler stopped")
	})
}

// scanLoop 主扫描循环：LockAndFetch → 异步执行 → 成功后更新下次时间。
// 有任务时 10ms 快速连扫，无任务时按 interval 间隔等待。
func (s *Scheduler) scanLoop() {
	for {
		if s.guard != nil && !s.guard() {
			timer := time.NewTimer(s.interval)
			select {
			case <-s.stopCh:
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}
			continue
		}

		claim, err := s.store.LockAndFetch(context.Background(), carbon.Now().StdTime(), s.lockExpire)
		if err == nil && claim != nil {
			logx.WithContext(taskLogContext(context.Background(), claim)).Info("[crontask] task claimed")
			s.workerGroup.Add(1)
			threading.GoSafe(func() {
				defer s.workerGroup.Done()
				s.executeTask(claim)
			})
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			logx.Errorf("[crontask] scan loop error: %v", err)
		}

		var sleepDuration time.Duration
		if err == nil && claim != nil {
			sleepDuration = 10 * time.Millisecond
		} else {
			sleepDuration = s.interval
		}

		timer := time.NewTimer(sleepDuration)
		select {
		case <-s.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

// executeTask 执行单个任务。handler 成功后计算下次调度时间并通过 lease CAS 完成。
// 若 MaxDelay > 0 且任务已延迟超过 MaxDelay，跳过执行直接计算下次时间。
func (s *Scheduler) executeTask(claim *TaskClaim) {
	task := claim.Task
	ctx := context.Background()
	ctx, span := s.tracer.Start(ctx, "crontask-execute",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	defer span.End()
	span.SetAttributes(
		attribute.String("crontask.code", task.TaskCode),
		attribute.String("crontask.name", task.TaskName),
		attribute.String("crontask.id", task.ID),
	)
	ctx = logx.ContextWithFields(ctx,
		logx.Field("task_code", task.TaskCode),
		logx.Field("task_id", task.ID),
		logx.Field("scheduled_run", task.ScheduledTime),
		logx.Field("locked_until", claim.LockedUntil),
	)

	stale := false
	maxDelay := s.maxDelay
	if task.MaxDelay > 0 {
		maxDelay = task.MaxDelay
	}
	if maxDelay > 0 && !task.ScheduledTime.IsZero() && time.Since(task.ScheduledTime) > maxDelay {
		logx.WithContext(ctx).Infof("[crontask] task skipped: delayed %v > max %v", time.Since(task.ScheduledTime), maxDelay)
		stale = true
	}

	lastRun := time.Time{}
	if !stale {
		startedAt := time.Now()
		logx.WithContext(ctx).Info("[crontask] handler started")
		if err := invokeHandler(s.handler, ctx, task); err != nil {
			logx.WithContext(ctx).WithDuration(time.Since(startedAt)).Errorf("[crontask] handler failed: %v", err)
			if errors.Is(err, ErrDeleteTask) {
				deleteErr := s.store.Delete(ctx, task.ID)
				if deleteErr != nil && !errors.Is(deleteErr, ErrNotFound) {
					logx.WithContext(ctx).Errorf("[crontask] task delete failed: %v", deleteErr)
				} else {
					logx.WithContext(ctx).Info("[crontask] task deleted")
				}
				return
			}
			return
		}
		lastRun = carbon.Now().StdTime()
		logx.WithContext(ctx).WithDuration(time.Since(startedAt)).Info("[crontask] handler succeeded")
	}

	nextRun, err := s.computeNextRun(task)
	if err != nil {
		logx.WithContext(ctx).Errorf("[crontask] compute next run failed: %v", err)
		return
	}
	completionCtx := logx.ContextWithFields(ctx, logx.Field("next_run", nextRun))
	logx.WithContext(completionCtx).Info("[crontask] next run computed")
	completion := Completion{NextRun: nextRun}
	if !stale {
		completion.LastRun = lastRun
		completion.LastScheduledRun = task.ScheduledTime
	}
	if err := s.store.Complete(completionCtx, task.ID, claim.LockedUntil, completion); err != nil {
		logx.WithContext(completionCtx).Errorf("[crontask] completion failed: %v", err)
	} else {
		logx.WithContext(completionCtx).Info("[crontask] completion committed")
	}
}

// RunNow 立即异步触发一次任务执行，成功时只记录 LastRun，不修改周期计划。
// 异步执行保留 ctx value，但不继承调用方的取消信号和截止时间。
// 返回当前 trace_id 供调用方追踪异步执行结果。
func (s *Scheduler) RunNow(ctx context.Context, taskCode string) (string, error) {
	task, err := s.store.GetByCode(ctx, taskCode)
	if err != nil {
		return "", err
	}
	task.ScheduledTime = carbon.Now().StartOfSecond().StdTime()
	runCtx := logx.ContextWithFields(context.WithoutCancel(ctx),
		logx.Field("task_code", task.TaskCode),
		logx.Field("task_id", task.ID),
		logx.Field("scheduled_run", task.ScheduledTime),
	)
	logx.WithContext(runCtx).Info("[crontask] run now queued")
	traceID := trace.TraceIDFromContext(ctx)
	threading.GoSafe(func() {
		startedAt := time.Now()
		logx.WithContext(runCtx).Info("[crontask] run now handler started")
		if err := invokeHandler(s.handler, runCtx, task); err != nil {
			logx.WithContext(runCtx).WithDuration(time.Since(startedAt)).Errorf("[crontask] run now handler failed: %v", err)
			if errors.Is(err, ErrDeleteTask) {
				deleteErr := s.store.Delete(runCtx, task.ID)
				if deleteErr != nil && !errors.Is(deleteErr, ErrNotFound) {
					logx.WithContext(runCtx).Errorf("[crontask] run now delete failed: %v", deleteErr)
				} else {
					logx.WithContext(runCtx).Info("[crontask] run now task deleted")
				}
				return
			}
			return
		}
		lastRun := carbon.Now().StdTime()
		logx.WithContext(runCtx).WithDuration(time.Since(startedAt)).Info("[crontask] run now handler succeeded")
		if err := s.store.UpdateLastRun(runCtx, task.ID, lastRun); err != nil {
			logx.WithContext(runCtx).Errorf("[crontask] run now completion failed: %v", err)
		} else {
			logx.WithContext(runCtx).Info("[crontask] run now completion committed")
		}
	})
	return traceID, nil
}

// PreviewNextRuns 返回严格晚于 after 的后续有效计划时间。
// 委托 rrulex.NextRuns：解析与平移只做一次，用单个迭代器顺序收集；
// 调度器的无效时间谓词逐候选介入，被判无效的候选直接跳过，跳过不推进游标。
// count 只统计最终接受的有效时间点；规则耗尽则提前返回已收集结果。
// 该方法只读取任务配置，不访问 Store，也不改变任务运行状态。
func (s *Scheduler) PreviewNextRuns(task *TaskConfig, after time.Time, count int) ([]time.Time, error) {
	if s == nil {
		return nil, errors.New("scheduler is nil")
	}
	if task == nil {
		return nil, errors.New("task is nil")
	}
	if count <= 0 {
		return []time.Time{}, nil
	}
	return s.nextRuns(task, after, count)
}

// invalidPredicate 将调度器的无效时间谓词绑定到任务，供 rrulex.NextRuns 逐候选过滤。
func (s *Scheduler) invalidPredicate(task *TaskConfig) func(time.Time) bool {
	if s.invalidTimePredicate == nil {
		return nil
	}
	return func(t time.Time) bool { return s.invalidTimePredicate(task, t) }
}

// nextRuns 将调度器的无效时间谓词绑定到任务，委托 rrulex.NextRuns 做单趟过滤收集。
func (s *Scheduler) nextRuns(task *TaskConfig, after time.Time, count int) ([]time.Time, error) {
	return rrulex.NextRuns(task.RRuleStr, after, false, count, s.invalidPredicate(task))
}

func taskLogContext(ctx context.Context, claim *TaskClaim) context.Context {
	return logx.ContextWithFields(ctx,
		logx.Field("task_code", claim.Task.TaskCode),
		logx.Field("task_id", claim.Task.ID),
		logx.Field("scheduled_run", claim.Task.ScheduledTime),
		logx.Field("locked_until", claim.LockedUntil),
	)
}

func invokeHandler(handler Handler, ctx context.Context, task *TaskConfig) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("handler panic (%T)\n%s", recovered, debug.Stack())
		}
	}()
	return handler(ctx, task)
}

// computeNextRun 基于 rrule 计算下一次调度时间，无效时间谓词在单趟迭代中推进跳过。
// 以 max(ScheduledTime, now) 为基准避免延迟后算出已过去的时间。
// 若已无更多触发计划（COUNT 耗尽、超出 Until、候选全部落在无效区间），返回零值表示无下次调度。
func (s *Scheduler) computeNextRun(cfg *TaskConfig) (time.Time, error) {
	if cfg.RRuleStr == "" {
		return time.Time{}, nil
	}
	now := carbon.Now().StdTime()
	base := now
	if cfg.ScheduledTime.After(now) {
		base = cfg.ScheduledTime
	}
	runs, err := rrulex.NextRuns(cfg.RRuleStr, base, false, 1, s.invalidPredicate(cfg))
	if err != nil || len(runs) == 0 {
		return time.Time{}, err
	}
	return runs[0], nil
}
