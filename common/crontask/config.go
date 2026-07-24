package crontask

import (
	"encoding/json"
	"time"
)

// MinLockTimeout 是单次任务 claim 的最小锁超时。
// 锁时间会按数据库 timestamp 精度截到整秒，过短的锁容易在写入后立即到期。
const MinLockTimeout = 30 * time.Second

// TaskStatus 任务启用状态。
type TaskStatus int

const (
	StatusDisabled TaskStatus = 0 // 禁用
	StatusEnabled  TaskStatus = 1 // 启用
)

func (s TaskStatus) String() string {
	switch s {
	case StatusEnabled:
		return "enabled"
	case StatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// TaskConfig 通用周期任务配置，与存储实现无关。
// Extra 字段存储业务方自定义扩展字段 JSON，调度器不解析。
// RRuleStr 为空表示一次性任务，成功执行后 NextRun 进入零值终态。
type TaskConfig struct {
	ID       string `json:"id"`
	TaskCode string `json:"task_code"` // 全局唯一任务编码
	TaskName string `json:"task_name"`
	RRuleStr string `json:"rrule_str"` // RFC 5545 规则，空串为一次性任务
	Priority int    `json:"priority"`  // 调度优先级，数字越大越优先
	// LockTimeout 是单次抢占的锁超时；零值表示使用 Scheduler 配置的默认锁超时。
	LockTimeout time.Duration   `json:"lock_timeout"`
	Payload     json.RawMessage `json:"payload"` // 执行业务参数
	Extra       json.RawMessage `json:"extra"`   // 业务扩展字段 JSON
	Status      TaskStatus      `json:"status"`
	NextRun     time.Time       `json:"next_run"`           // 下次计划调度时间，零值表示无下次调度
	LastRun     time.Time       `json:"last_run,omitempty"` // 上次执行时间，零值表示从未执行
}

// ResolveLockTimeout 返回任务本次抢占实际使用的锁超时。
// 任务未配置正数锁超时时使用 Scheduler 默认值，最终结果不低于 MinLockTimeout。
func ResolveLockTimeout(lockTimeout, defaultLockTimeout time.Duration) time.Duration {
	resolved := defaultLockTimeout
	if lockTimeout > 0 {
		resolved = lockTimeout
	}
	if resolved < MinLockTimeout {
		return MinLockTimeout
	}
	return resolved
}
