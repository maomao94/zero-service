package crontask

import (
	"context"
	"time"
)

type SchedulerOptions struct {
	Interval             time.Duration
	LockExpire           time.Duration // 默认锁超时；任务配置了正数 LockTimeout 时由任务值覆盖
	MaxDelay             time.Duration // 最大延迟容忍，超过则跳过执行直接计算下次时间，0=不限制
	MisfireDefault       MisfirePolicy // 任务未配置 Misfire 时使用的默认错过执行策略；已确认为 MisfirePolicySkip（跳过）
	InvalidTimePredicate InvalidTimePredicate
	Guard                Guard       // 扫表前置条件，nil 表示不限制
	OnSkip               SkipHandler // 跳过上报回调，nil 表示不上报
}

// InvalidTimePredicate 判断给定候选计划时间是否落在不可用区间，true 表示该时刻无效应跳过。
// 谓词只做排除，不返回时间；推进与跳过由调度器在单趟迭代中完成。
// task 为当前任务，t 为 rrule 计算的候选时间。
type InvalidTimePredicate func(task *TaskConfig, t time.Time) bool

// Guard 扫表前置条件，返回 false 则本次跳过扫表。
type Guard func() bool

// SkipReason 描述一次计划执行被跳过或未被正常执行的原因。
type SkipReason string

const (
	// SkipReasonMisfire 表示任务超过延迟容忍且策略不补跑，本次计划执行被跳过。
	SkipReasonMisfire SkipReason = "misfire"
)

// SkipEvent 描述一次被跳过的计划执行，供跳过上报回调留痕。
type SkipEvent struct {
	Task          *TaskConfig
	ScheduledTime time.Time
	LockedUntil   time.Time
	Delay         time.Duration
	MaxDelay      time.Duration
	Policy        MisfirePolicy
	Reason        SkipReason
}

// SkipHandler 跳过上报回调；调度器跳过某次计划执行时调用，用于留痕、避免静默漏检。
type SkipHandler func(ctx context.Context, event SkipEvent)

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

// WithMisfireDefault 设置任务未配置 Misfire 时使用的默认错过执行策略。
// 全局默认已确认为跳过；传入 MisfirePolicyDefault 表示保持调度器当前默认（跳过）。
func WithMisfireDefault(p MisfirePolicy) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.MisfireDefault = p
	}
}

// WithOnSkip 设置跳过上报回调。调度器跳过某次计划执行（错过执行且策略不补跑）时调用，用于留痕。
func WithOnSkip(h SkipHandler) SchedulerOption {
	return func(o *SchedulerOptions) {
		o.OnSkip = h
	}
}
