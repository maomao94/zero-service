package gormmodel

import (
	"time"

	"zero-service/common/gormx"
)

// CronExecLog 记录每次 crontask 调度触发执行的日志。
type CronExecLog struct {
	gormx.LegacyStringBaseModel

	TraceId       string    `gorm:"column:trace_id;size:64;comment:追踪 ID"`
	JobId         string    `gorm:"column:job_id;size:64;index:idx_cron_exec_log_job_id;comment:cron_job ID"`
	TaskCode      string    `gorm:"column:task_code;size:64;index:idx_cron_exec_log_task_code;comment:任务编码"`
	TaskName      string    `gorm:"column:task_name;size:128;comment:任务名称"`
	ScheduledTime time.Time `gorm:"column:scheduled_time;type:timestamp;comment:原计划执行时间"`
	StartTime     time.Time `gorm:"column:start_time;type:timestamp;comment:实际开始执行时间"`
	EndTime       time.Time `gorm:"column:end_time;type:timestamp;comment:实际结束执行时间"`
	CostMs        int64     `gorm:"column:cost_ms;comment:执行耗时(毫秒)"`
	Status        int       `gorm:"column:status;comment:执行状态：1-成功 0-失败"`
	ErrorMessage  string    `gorm:"column:error_message;type:text;comment:失败时的错误信息"`
}

// TableName 返回执行日志表名。
func (CronExecLog) TableName() string { return "cron_exec_log" }
