package crontask

import (
	"context"
	"errors"
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
	tracer            oteltrace.Tracer
	invalidTimeFilter InvalidTimeFilter
	guard             Guard
}

// NewScheduler 创建调度器，默认扫描间隔 2s，锁过期 30s。
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
	logx.Info("[crontask] scheduler started")
	go s.scanLoop()
}

// Stop 停止调度器。
func (s *Scheduler) Stop() {
	close(s.stopCh)
	logx.Info("[crontask] scheduler stopped")
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

		task, err := s.store.LockAndFetch(context.Background(), carbon.Now().StdTime(), s.lockExpire)
		if err == nil && task != nil {
			threading.GoSafe(func() {
				s.executeTask(task)
			})
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			logx.Errorf("[crontask] scan loop error: %v", err)
		}

		var sleepDuration time.Duration
		if err == nil && task != nil {
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

// executeTask 执行单个任务。handler 成功后计算下次调度时间并更新。
// 若 MaxDelay > 0 且任务已延迟超过 MaxDelay，跳过执行直接计算下次时间。
func (s *Scheduler) executeTask(task *TaskConfig) {
	ctx := context.Background()
	ctx, span := s.tracer.Start(ctx, "crontask-execute",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	defer span.End()
	span.SetAttributes(
		attribute.String("crontask.code", task.TaskCode),
		attribute.String("crontask.name", task.TaskName),
		attribute.Int64("crontask.id", task.ID),
		attribute.Int64("crontask.version", task.Version),
	)
	ctx = logx.ContextWithFields(ctx,
		logx.Field("task_code", task.TaskCode),
		logx.Field("task_id", task.ID),
	)

	stale := false
	if s.maxDelay > 0 && time.Since(task.NextRun) > s.maxDelay {
		logx.WithContext(ctx).Infof("[crontask] task %s skipped: delayed %v > max %v", task.TaskCode, time.Since(task.NextRun), s.maxDelay)
		stale = true
	}

	if !stale {
		if err := s.handler(ctx, task); err != nil {
			logx.WithContext(ctx).Errorf("[crontask] task %s execute failed: %v", task.TaskCode, err)
			return
		}
	}

	nextRun, err := computeNextRun(task)
	if err != nil {
		logx.WithContext(ctx).Errorf("[crontask] task %s compute next run failed: %v", task.TaskCode, err)
		return
	}
	if s.invalidTimeFilter != nil {
		nextRun = s.invalidTimeFilter(task, nextRun)
	}
	if err := s.store.UpdateNextRun(ctx, task.ID, nextRun, carbon.Now().StdTime()); err != nil {
		logx.WithContext(ctx).Errorf("[crontask] task %s update next run failed: %v", task.TaskCode, err)
	}
}

// RunNow 立即异步触发一次任务执行，不修改 next_run。
func (s *Scheduler) RunNow(ctx context.Context, taskCode string) error {
	task, err := s.store.GetByCode(ctx, taskCode)
	if err != nil {
		return err
	}
	go s.executeTask(task)
	return nil
}

// computeNextRun 基于 rrule 计算下一次调度时间。
// 以 max(NextRun, now) 为基准避免延迟后算出已过去的时间。
// 若已无更多触发计划（COUNT 耗尽、超出 Until），返回 100 年后避免重复调度。
func computeNextRun(cfg *TaskConfig) (time.Time, error) {
	base := cfg.NextRun
	if now := carbon.Now().StdTime(); base.Before(now) {
		base = now
	}

	set, err := rrule.StrToRRuleSet(cfg.RRuleStr)
	if err == nil {
		next := set.After(base, false)
		if next.IsZero() {
			return carbon.Now().AddYears(100).StdTime(), nil
		}
		return next, nil
	}

	rule, err := rrule.StrToRRule(cfg.RRuleStr)
	if err != nil {
		return time.Time{}, err
	}
	next := rule.After(base, false)
	if next.IsZero() {
		return carbon.Now().AddYears(100).StdTime(), nil
	}
	return next, nil
}
