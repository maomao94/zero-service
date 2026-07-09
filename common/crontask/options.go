package crontask

import "time"

type SchedulerOptions struct {
	Interval          time.Duration
	LockExpire        time.Duration
	InvalidTimeFilter InvalidTimeFilter
}

// InvalidTimeFilter 在计算下次执行时间后调用，跳过不可用时间段。
// task 为当前任务，next 为 rrule 计算的下次时间。
// 若 next 在不可用范围内，应循环调用 rrule.After 直到跳出范围。
type InvalidTimeFilter func(task *TaskConfig, next time.Time) time.Time

type SchedulerOption func(*SchedulerOptions)

func WithInterval(d time.Duration) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.Interval = d
	}
}

func WithLockExpire(d time.Duration) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.LockExpire = d
	}
}

func WithInvalidTimeFilter(f InvalidTimeFilter) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.InvalidTimeFilter = f
	}
}
