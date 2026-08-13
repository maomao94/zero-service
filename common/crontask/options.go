package crontask

import "time"

type SchedulerOptions struct {
	Interval             time.Duration
	LockExpire           time.Duration // 默认锁超时；任务配置了正数 LockTimeout 时由任务值覆盖
	MaxDelay             time.Duration // 最大延迟容忍，超过则跳过执行直接计算下次时间，0=不限制
	InvalidTimePredicate InvalidTimePredicate
	Guard                Guard // 扫表前置条件，nil 表示不限制
}

// InvalidTimePredicate 判断给定候选计划时间是否落在不可用区间，true 表示该时刻无效应跳过。
// 谓词只做排除，不返回时间；推进与跳过由调度器在单趟迭代中完成。
// task 为当前任务，t 为 rrule 计算的候选时间。
type InvalidTimePredicate func(task *TaskConfig, t time.Time) bool

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

func WithInvalidTimePredicate(f InvalidTimePredicate) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.InvalidTimePredicate = f
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
