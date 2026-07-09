package crontask

import "time"

type SchedulerOptions struct {
	Interval   time.Duration
	LockExpire time.Duration
}

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
