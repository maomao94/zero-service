package gormmodel

import (
	"database/sql"

	"zero-service/common/gormx"
)

// CronJob 是 Trigger 基于 RRULE 的周期任务配置。
// 核心字段与 crontask.TaskConfig 对齐，Trigger 业务字段平铺为列便于查询和后续扩展。
type CronJob struct {
	gormx.LegacyStringBaseModel

	// crontask.TaskConfig 对齐字段。
	TaskCode    string       `gorm:"column:task_code;size:64;uniqueIndex:uq_cron_job_task_code;comment:全局唯一任务编码"`
	TaskName    string       `gorm:"column:task_name;size:128;comment:任务名称"`
	RRuleStr    string       `gorm:"column:rrule_str;type:text;comment:RFC 5545 规则字符串"`
	Priority    int          `gorm:"column:priority;default:0;index:idx_cron_job_priority;comment:调度优先级，数字越大越优先"`
	LockTimeout int64        `gorm:"column:lock_timeout;default:0;comment:单次调度锁超时（毫秒），0 使用调度器默认值"`
	Payload     string       `gorm:"column:payload;type:text;comment:业务执行参数 JSON"`
	Extra       string       `gorm:"column:extra;type:text;comment:Trigger 业务扩展字段 JSON"`
	Status      int          `gorm:"column:status;index:idx_cron_job_scan,priority:1;comment:状态：0-禁用，1-启用"`
	NextRun     sql.NullTime `gorm:"column:next_run;type:timestamp;index:idx_cron_job_scan,priority:2;comment:下次计划调度时间，NULL 表示无下次调度"`
	LastRun     sql.NullTime `gorm:"column:last_run;type:timestamp;comment:上次成功执行时间，NULL 表示从未成功执行"`
	// ScheduledTime 在回调重试期间保存最初的计划时间，成功完成或重新启用后清空。
	ScheduledTime sql.NullTime `gorm:"column:scheduled_time;type:timestamp;comment:在途执行的原计划时间，NULL 表示当前没有重试中的执行"`

	// Trigger 创建请求业务字段。
	DeptCode     string         `gorm:"column:dept_code;size:64;index:idx_cron_job_dept_code;comment:机构编码"`
	Type         string         `gorm:"column:type;size:64;index:idx_cron_job_type;comment:任务类型"`
	GroupId      string         `gorm:"column:group_id;size:64;index:idx_cron_job_group_id;comment:任务分组 ID"`
	Description  string         `gorm:"column:description;size:200;comment:任务描述"`
	StartTime    sql.NullTime   `gorm:"column:start_time;type:timestamp;comment:规则生效开始时间，NULL 表示调用方未指定"`
	EndTime      sql.NullTime   `gorm:"column:end_time;type:timestamp;comment:规则生效结束时间，NULL 表示调用方未指定"`
	Rule         string         `gorm:"column:rule;type:text;comment:PlanRulePb JSON"`
	ExcludeDates sql.NullString `gorm:"column:exclude_dates;type:text;comment:排除日期列表 JSON，NULL 表示未配置"`
	Ext1         string         `gorm:"column:ext_1;size:256;comment:扩展字段 1"`
	Ext2         string         `gorm:"column:ext_2;size:256;comment:扩展字段 2"`
	Ext3         string         `gorm:"column:ext_3;size:256;comment:扩展字段 3"`
	Ext4         string         `gorm:"column:ext_4;size:256;comment:扩展字段 4"`
	Ext5         string         `gorm:"column:ext_5;size:256;comment:扩展字段 5"`
}

// TableName 返回 Cron Job 配置表名。
func (CronJob) TableName() string { return "cron_job" }
