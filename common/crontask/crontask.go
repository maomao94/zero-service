package crontask

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
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
	store             TaskStore
	handler           Handler
	interval          time.Duration
	lockExpire        time.Duration
	maxDelay          time.Duration
	stopCh            chan struct{}
	startOnce         sync.Once
	stopOnce          sync.Once
	workerGroup       sync.WaitGroup
	tracer            oteltrace.Tracer
	invalidTimeFilter InvalidTimeFilter
	guard             Guard
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
		store:             store,
		handler:           handler,
		interval:          o.Interval,
		lockExpire:        o.LockExpire,
		maxDelay:          o.MaxDelay,
		stopCh:            make(chan struct{}),
		tracer:            otel.Tracer(trace.TraceName),
		invalidTimeFilter: o.InvalidTimeFilter,
		guard:             o.Guard,
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
	)

	stale := false
	if s.maxDelay > 0 && !task.NextRun.IsZero() && time.Since(task.NextRun) > s.maxDelay {
		logx.WithContext(ctx).Infof("[crontask] task %s skipped: delayed %v > max %v", task.TaskCode, time.Since(task.NextRun), s.maxDelay)
		stale = true
	}

	lastRun := time.Time{}
	if !stale {
		if err := s.handler(ctx, task); err != nil {
			if errors.Is(err, ErrDeleteTask) {
				deleteErr := s.store.Delete(ctx, task.ID)
				if deleteErr != nil && !errors.Is(deleteErr, ErrNotFound) {
					logx.WithContext(ctx).Errorf("[crontask] task %s delete failed: %v", task.TaskCode, deleteErr)
				}
				return
			}
			logx.WithContext(ctx).Errorf("[crontask] task %s execute failed: %v", task.TaskCode, err)
			return
		}
		lastRun = carbon.Now().StdTime()
	}

	nextRun, err := computeNextRun(task)
	if err != nil {
		logx.WithContext(ctx).Errorf("[crontask] task %s compute next run failed: %v", task.TaskCode, err)
		return
	}
	if s.invalidTimeFilter != nil {
		nextRun = s.invalidTimeFilter(task, nextRun)
	}
	if err := s.store.Complete(ctx, task.ID, claim.LockedUntil, nextRun, lastRun); err != nil {
		logx.WithContext(ctx).Errorf("[crontask] task %s complete failed: %v", task.TaskCode, err)
	}
}

// RunNow 立即异步触发一次任务执行，成功时只记录 LastRun，不修改周期计划。
// 异步执行保留 ctx value，但不继承调用方的取消信号和截止时间。
func (s *Scheduler) RunNow(ctx context.Context, taskCode string) error {
	task, err := s.store.GetByCode(ctx, taskCode)
	if err != nil {
		return err
	}
	task.NextRun = carbon.Now().StartOfSecond().StdTime()
	runCtx := context.WithoutCancel(ctx)
	threading.GoSafe(func() {
		if err := s.handler(runCtx, task); err != nil {
			if errors.Is(err, ErrDeleteTask) {
				deleteErr := s.store.Delete(runCtx, task.ID)
				if deleteErr != nil && !errors.Is(deleteErr, ErrNotFound) {
					logx.WithContext(runCtx).Errorf("[crontask] task %s run now delete failed: %v", task.TaskCode, deleteErr)
				}
				return
			}
			logx.WithContext(runCtx).Errorf("[crontask] task %s run now failed: %v", task.TaskCode, err)
			return
		}
		if err := s.store.UpdateLastRun(runCtx, task.ID, carbon.Now().StdTime()); err != nil {
			logx.WithContext(runCtx).Errorf("[crontask] task %s update manual last run failed: %v", task.TaskCode, err)
		}
	})
	return nil
}

// computeNextRun 基于 rrule 计算下一次调度时间。
// 以 max(NextRun, now) 为基准避免延迟后算出已过去的时间。
// 若已无更多触发计划（COUNT 耗尽、超出 Until），返回零值表示无下次调度。
func computeNextRun(cfg *TaskConfig) (time.Time, error) {
	if cfg.RRuleStr == "" {
		return time.Time{}, nil
	}
	now := carbon.Now().StdTime()
	base := now
	if cfg.NextRun.After(now) {
		base = cfg.NextRun
	}
	return NextAfter(cfg.RRuleStr, base)
}

// NextAfter 返回 RRULE 或 RRULE set 在指定时间之后的首次计划时间。
// 空规则和已耗尽规则都返回零时间；非法非空规则返回解析错误。
func NextAfter(value string, after time.Time) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	set, err := rrule.StrToRRuleSet(value)
	if err == nil {
		next := set.After(after, false)
		if next.IsZero() {
			return time.Time{}, nil
		}
		return next, nil
	}

	rule, err := rrule.StrToRRule(value)
	if err != nil {
		return time.Time{}, err
	}
	next := rule.After(after, false)
	if next.IsZero() {
		return time.Time{}, nil
	}
	return next, nil
}

// ValidateRRule 校验非空 RRULE 或 RRULE set 是否能被调度器解析。
// 空字符串表示一次性任务，是合法配置。
func ValidateRRule(value string) error {
	if value == "" {
		return nil
	}
	if _, err := rrule.StrToRRuleSet(value); err == nil {
		return nil
	}
	_, err := rrule.StrToRRule(value)
	return err
}
