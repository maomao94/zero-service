package gormmodel

import (
	"database/sql"
	"time"

	"zero-service/common/gormx"
)

// GormTaskConfig 是 ISP 周期性任务配置表。
// 核心字段与 crontask.TaskConfig 对齐，ISP 业务字段平铺为表列方便查询与索引。
type GormTaskConfig struct {
	gormx.LegacyStringBaseModel // id / create_time / update_time / delete_time / is_deleted

	// --- crontask.TaskConfig 对齐字段 ---
	TaskCode string       `gorm:"column:task_code;size:64;uniqueIndex;comment:全局唯一任务编码"` // 全局唯一任务编码
	TaskName string       `gorm:"column:task_name;size:128;comment:任务名称"`                // 任务名称
	RRuleStr string       `gorm:"column:rrule_str;size:1048;comment:RFC 5545 规则字符串"`     // RFC 5545 规则字符串
	Priority int          `gorm:"column:priority;default:1;index;comment:任务优先级"`         // 任务优先级
	Payload  string       `gorm:"column:payload;type:text;comment:业务参数（如 device_list）"`  // 业务参数（如 device_list）
	Extra    string       `gorm:"column:extra;type:text;comment:业务扩展字段 JSON"`            // 业务扩展字段 JSON
	Status   int          `gorm:"column:status;default:1;index;comment:任务配置状态"`          // 任务配置状态
	NextRun  sql.NullTime `gorm:"column:next_run;type:timestamp;index;comment:下次执行时间"`   // 下次执行时间，NULL 表示无下次调度
	LastRun  sql.NullTime `gorm:"column:last_run;type:timestamp;comment:上次执行时间"`         // 上次执行时间

	// --- ISP 业务字段（平铺为列）---
	SubstationCode      string `gorm:"column:substation_code;size:64;index;comment:变电站编码"`           // 变电站编码
	PatrolType          string `gorm:"column:patrol_type;size:4;comment:巡视类型：1=例行，2=特殊，3=专项，4=自定义"`  // 巡视类型
	DeviceLevel         int    `gorm:"column:device_level;default:3;comment:设备层级"`                   // 设备层级
	DeviceList          string `gorm:"column:device_list;type:text;comment:设备列表（逗号分隔）"`              // 设备列表（逗号分隔）
	FixedStartTime      string `gorm:"column:fixed_start_time;size:32;comment:定期开始时间"`               // 定期开始时间
	CycleMonth          string `gorm:"column:cycle_month;size:32;comment:周期（月）"`                     // 周期（月）
	CycleWeek           string `gorm:"column:cycle_week;size:32;comment:周期（周）"`                      // 周期（周）
	CycleExecuteTime    string `gorm:"column:cycle_execute_time;size:16;comment:周期执行时间 HH:mm:ss"`    // 周期执行时间 HH:mm:ss
	CycleStartTime      string `gorm:"column:cycle_start_time;size:32;comment:周期开始时间"`               // 周期开始时间
	CycleEndTime        string `gorm:"column:cycle_end_time;size:32;comment:周期结束时间"`                 // 周期结束时间
	IntervalNumber      string `gorm:"column:interval_number;size:16;comment:间隔数量"`                  // 间隔数量
	IntervalType        string `gorm:"column:interval_type;size:4;comment:间隔类型：1=小时，2=天"`            // 间隔类型
	IntervalExecuteTime string `gorm:"column:interval_execute_time;size:16;comment:间隔执行时间 HH:mm:ss"` // 间隔执行时间 HH:mm:ss
	IntervalStartTime   string `gorm:"column:interval_start_time;size:32;comment:间隔开始时间"`            // 间隔开始时间
	IntervalEndTime     string `gorm:"column:interval_end_time;size:32;comment:间隔结束时间"`              // 间隔结束时间
	InvalidStartTime    string `gorm:"column:invalid_start_time;size:32;comment:不可用开始时间"`            // 不可用开始时间
	InvalidEndTime      string `gorm:"column:invalid_end_time;size:32;comment:不可用结束时间"`              // 不可用结束时间
	IsEnable            string `gorm:"column:isenable;size:4;comment:是否启用：0=启用，1=禁用，2=删除"`           // 是否启用 (0=启用 1=禁用 2=删除)
	IspCreator          string `gorm:"column:isp_creator;size:64;comment:编制人"`                       // 编制人
	IspCreateTime       string `gorm:"column:isp_create_time;size:32;comment:编制时间"`                  // 编制时间
}

func (GormTaskConfig) TableName() string { return "cron_task_config" }

// PatrolTaskState 记录 ISP 协议任务状态上报中的 task_state。
type PatrolTaskState string

const (
	PatrolTaskStateFinished   PatrolTaskState = "1" // 已执行
	PatrolTaskStateRunning    PatrolTaskState = "2" // 正在执行
	PatrolTaskStatePaused     PatrolTaskState = "3" // 暂停
	PatrolTaskStateTerminated PatrolTaskState = "4" // 终止
	PatrolTaskStatePending    PatrolTaskState = "5" // 未执行
	PatrolTaskStateOverdue    PatrolTaskState = "6" // 超期
)

// GormIspPatrolTask 是区域巡视主机侧的 ISP 协议巡视任务主表。
// 字段由任务状态通知（Type=41, Command=0）的报文信封和表 J.42 Item 属性组成。
type GormIspPatrolTask struct {
	gormx.LegacyStringBaseModel // id / create_time / update_time / delete_time / is_deleted

	SendCode          string    `gorm:"column:send_code;size:64;not null;index;comment:区域巡视主机唯一标识"`                    // SendCode：区域巡视主机唯一标识
	ReceiveCode       string    `gorm:"column:receive_code;size:64;not null;comment:上级系统唯一标识"`                         // ReceiveCode：上级系统唯一标识
	Code              string    `gorm:"column:code;size:64;not null;index;comment:变电站编码"`                              // Code：变电站编码
	TaskPatrolledID   string    `gorm:"column:task_patrolled_id;size:255;not null;uniqueIndex;comment:巡视任务执行 ID"`      // 巡视任务执行 ID
	TaskName          string    `gorm:"column:task_name;size:255;comment:任务名称"`                                        // 任务名称
	TaskCode          string    `gorm:"column:task_code;size:255;not null;index;comment:任务编码"`                         // 任务编码
	TaskState         string    `gorm:"column:task_state;size:8;index;comment:任务状态：1=已执行，2=正在执行，3=暂停，4=终止，5=未执行，6=超期"` // 任务状态
	PlanStartTime     time.Time `gorm:"column:plan_start_time;type:timestamp;comment:计划开始时间"`                          // 计划开始时间
	StartTime         time.Time `gorm:"column:start_time;type:timestamp;comment:开始时间"`                                 // 开始时间
	TaskProgress      string    `gorm:"column:task_progress;size:128;comment:任务进度"`                                    // 任务进度
	TaskEstimatedTime string    `gorm:"column:task_estimated_time;size:128;comment:任务预计剩余时间"`                          // 任务预计剩余时间
	Description       string    `gorm:"column:description;type:text;comment:描述"`                                       // 描述
}

func (GormIspPatrolTask) TableName() string { return "isp_patrol_task" }
