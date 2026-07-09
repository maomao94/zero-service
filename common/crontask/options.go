package crontask

import "time"

type SchedulerOptions struct {
	Interval          time.Duration
	LockExpire        time.Duration
	MaxDelay          time.Duration // 最大延迟容忍，超过则跳过执行直接计算下次时间，0=不限制
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

// WithMaxDelay 设置最大延迟容忍。任务 next_run 距当前时间超过此值则跳过执行，直接计算下次时间。
func WithMaxDelay(d time.Duration) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.MaxDelay = d
	}
}
