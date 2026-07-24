package crontask

import "time"

type SchedulerOptions struct {
	Interval          time.Duration
	LockExpire        time.Duration // 默认锁超时；任务配置了正数 LockTimeout 时由任务值覆盖
	MaxDelay          time.Duration // 最大延迟容忍，超过则跳过执行直接计算下次时间，0=不限制
	InvalidTimeFilter InvalidTimeFilter
	Guard             Guard // 扫表前置条件，nil 表示不限制
}

// InvalidTimeFilter 在计算下次执行时间后调用，跳过不可用时间段。
// task 为当前任务，next 为 rrule 计算的下次时间。
// 若 next 在不可用范围内，应循环调用 rrule.After 直到跳出范围。
type InvalidTimeFilter func(task *TaskConfig, next time.Time) time.Time

// Guard 扫表前置条件，返回 false 则本次跳过扫表。
type Guard func() bool

type SchedulerOption func(*SchedulerOptions)

func WithInterval(d time.Duration) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.Interval = d
	}
}

// WithLockExpire 设置任务未配置 LockTimeout 时使用的默认锁超时。
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

// WithGuard 设置扫表前置条件。Guard 返回 false 时跳过本次 LockAndFetch。
func WithGuard(g Guard) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.Guard = g
	}
}
