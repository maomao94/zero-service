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

// MisfirePolicy 定义任务错过计划执行时间（misfire）后的补偿策略。
// 零值表示使用调度器默认策略；默认值已于 2026-10-16 业务确认为 MisfirePolicySkip（跳过，保持现状）。
type MisfirePolicy int

const (
	// MisfirePolicyDefault 使用调度器配置的默认策略；当前全局默认已确认为 MisfirePolicySkip（跳过）。
	MisfirePolicyDefault MisfirePolicy = 0
	// MisfirePolicySkip 跳过本次错过执行，直接推进到下一个计划点（保持现状，当前默认）。
	MisfirePolicySkip MisfirePolicy = 1
	// MisfirePolicyRunImmediately 立即补跑一次错过的执行，完成后推进到下一个计划点。
	MisfirePolicyRunImmediately MisfirePolicy = 2
	// MisfirePolicyDelayNext 不补跑，直接顺延到下一个计划点。
	// 当前推进实现下与 MisfirePolicySkip 结果一致，保留独立语义供后续补偿策略扩展。
	MisfirePolicyDelayNext MisfirePolicy = 3
)

func (p MisfirePolicy) String() string {
	switch p {
	case MisfirePolicySkip:
		return "skip"
	case MisfirePolicyRunImmediately:
		return "run_immediately"
	case MisfirePolicyDelayNext:
		return "delay_next"
	default:
		return "default"
	}
}

// TaskConfig 通用周期任务配置，与存储实现无关。
// Extra 字段存储业务方自定义扩展字段 JSON，调度器不解析。
// RRuleStr 为空表示一次性任务，成功执行后 NextRun 进入零值终态。
type TaskConfig struct {
	ID         string    `json:"id"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
	TaskCode   string    `json:"task_code"` // 全局唯一任务编码
	TaskName   string    `json:"task_name"`
	RRuleStr   string    `json:"rrule_str"` // RFC 5545 规则，空串为一次性任务
	Priority   int       `json:"priority"`  // 调度优先级，数字越大越优先
	// LockTimeout 是单次抢占的锁超时；零值表示使用 Scheduler 配置的默认锁超时。
	LockTimeout time.Duration `json:"lock_timeout"`
	MaxDelay    time.Duration `json:"max_delay,omitempty"` // 最大延迟容忍，0 使用调度器默认值
	// Misfire 是任务级错过执行策略；零值使用调度器默认策略（已确认为跳过）。
	Misfire   MisfirePolicy   `json:"misfire"`
	Payload   json.RawMessage `json:"payload"` // 执行业务参数
	Extra     json.RawMessage `json:"extra"`   // 业务扩展字段 JSON
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	Status    TaskStatus      `json:"status"`
	// NextRun 一列两义：未被 claim 时是下次计划调度时间；被 claim 后到 Complete 期间本字段即 lease 截止时间。
	// 这是有意的单列锁设计（借鉴 dtm 扫表），不拆分独立锁列，Store 实现与消费方必须按此语义理解。
	NextRun          time.Time `json:"next_run"`
	ScheduledTime    time.Time `json:"scheduled_time"`     // 当前在途执行的原计划时间，未 claim 时为零值
	LastRun          time.Time `json:"last_run"`           // 最近一次 handler 成功完成的实际时间
	LastScheduledRun time.Time `json:"last_scheduled_run"` // 最近一次成功周期执行的原计划时间，手动执行不更新
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
