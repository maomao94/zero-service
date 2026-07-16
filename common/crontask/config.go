package crontask

import (
	"encoding/json"
	"time"
)

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
// RRuleStr 为空表示一次性任务（执行后自动禁用）。
type TaskConfig struct {
	ID       string          `json:"id"`
	TaskCode string          `json:"task_code"` // 全局唯一任务编码
	TaskName string          `json:"task_name"`
	RRuleStr string          `json:"rrule_str"` // RFC 5545 规则，空串为一次性任务
	Priority int             `json:"priority"`  // 调度优先级，数字越大越优先
	Payload  json.RawMessage `json:"payload"`   // 执行业务参数
	Extra    json.RawMessage `json:"extra"`     // 业务扩展字段 JSON
	Status   TaskStatus      `json:"status"`
	NextRun  time.Time       `json:"next_run"`           // 下次计划调度时间
	LastRun  *time.Time      `json:"last_run,omitempty"` // 上次执行时间
	Version  int64           `json:"version"`            // 乐观锁版本号
}
