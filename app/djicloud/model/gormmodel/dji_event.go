package gormmodel

import (
	"database/sql"
	"time"

	"zero-service/common/gormx"
)

// DjiHmsAlert 是大疆 HMS 健康告警记录表。
//
// 功能：记录机巢上报的 HMS 告警，用于告警列表、告警确认、设备健康看板和后续告警通知。
// 数据来源：thing/product/{gateway_sn}/events method=device_hms。
// 写入策略：按每次上报逐条插入，保留告警历史；当前阶段不做去重合并，避免丢失上报细节。
// 使用场景：查询未确认告警、按设备/机巢/等级筛选告警、人工确认处理。
type DjiHmsAlert struct {
	gormx.LegacyBaseModel
	GatewaySn      string       `gorm:"column:gateway_sn;type:varchar(64);index;not null;comment:上报告警的网关机巢SN"`
	Level          int          `gorm:"column:level;index;not null;default:0;comment:告警等级"`
	Module         int          `gorm:"column:module;not null;default:0;comment:告警模块"`
	Code           string       `gorm:"column:code;type:varchar(64);not null;default:'';comment:大疆HMS告警码"`
	DeviceType     string       `gorm:"column:device_type;type:varchar(32);default:'';comment:告警设备类型"`
	Imminent       int          `gorm:"column:imminent;default:0;comment:是否紧急"`
	InTheSky       int          `gorm:"column:in_the_sky;default:0;comment:是否空中告警"`
	ComponentIndex int          `gorm:"column:component_index;default:0;comment:部件索引"`
	SensorIndex    int          `gorm:"column:sensor_index;default:0;comment:传感器索引"`
	ItemJSON       string       `gorm:"column:item_json;type:jsonb;default:'{}';comment:HMS告警条目原始JSON"`
	Acked          int          `gorm:"column:acked;index;not null;default:0;comment:确认状态，0未确认，1已确认"`
	AckedAt        sql.NullTime `gorm:"column:acked_at;comment:确认时间"`
	AckedBy        string       `gorm:"column:acked_by;type:varchar(64);default:'';comment:确认人"`
	ReportedAt     time.Time    `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
}

func (DjiHmsAlert) TableName() string { return "dji_hms_alert" }

// DjiDockFlightTask 是机巢航线任务最新快照表。
type DjiDockFlightTask struct {
	gormx.LegacyBaseModel
	FlightId   string    `gorm:"column:flight_id;type:varchar(64);uniqueIndex:idx_dji_dock_flight_task_gateway_flight;not null;comment:大疆航线任务ID"`
	GatewaySn  string    `gorm:"column:gateway_sn;type:varchar(64);uniqueIndex:idx_dji_dock_flight_task_gateway_flight;not null;comment:网关机巢SN"`
	ReportedAt time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	EventJSON  string    `gorm:"column:event_json;type:jsonb;default:'{}';comment:完整flighttask_progress事件data原始JSON"`
	ExtJSON    string    `gorm:"column:ext_json;type:jsonb;default:'{}';comment:flighttask_progress.ext原始JSON"`

	Status               string  `gorm:"column:status;type:varchar(64);index;not null;default:'';comment:官方flighttask_progress status字段"`
	CurrentStep          int     `gorm:"column:current_step;default:0;comment:官方progress.current_step字段"`
	WaylineMissionState  int     `gorm:"column:wayline_mission_state;default:0;comment:官方ext.wayline_mission_state字段"`
	CurrentWaypointIndex int     `gorm:"column:current_waypoint_index;default:0;comment:官方ext.current_waypoint_index字段"`
	MediaCount           int     `gorm:"column:media_count;default:0;comment:官方ext.media_count字段"`
	ProgressPercent      float64 `gorm:"column:progress_percent;default:0;comment:官方progress.percent字段"`
	TrackId              string  `gorm:"column:track_id;type:varchar(64);index;not null;default:'';comment:官方ext.track_id字段"`
	WaylineId            int     `gorm:"column:wayline_id;default:0;comment:官方ext.wayline_id字段"`
	BreakPointJSON       string  `gorm:"column:break_point_json;type:jsonb;default:'{}';comment:官方ext.break_point原始JSON"`
}

func (DjiDockFlightTask) TableName() string { return "dji_dock_flight_task" }

// DjiDockDeviceFlightTaskState 是机巢当前航线任务状态表。
type DjiDockDeviceFlightTaskState struct {
	gormx.LegacyBaseModel
	GatewaySn  string    `gorm:"column:gateway_sn;type:varchar(64);uniqueIndex;not null;comment:网关机巢SN"`
	FlightId   string    `gorm:"column:flight_id;type:varchar(64);index;not null;comment:大疆航线任务ID"`
	ReportedAt time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	EventJSON  string    `gorm:"column:event_json;type:jsonb;default:'{}';comment:完整flighttask_progress事件data原始JSON"`
	ExtJSON    string    `gorm:"column:ext_json;type:jsonb;default:'{}';comment:flighttask_progress.ext原始JSON"`

	Status               string  `gorm:"column:status;type:varchar(64);index;not null;default:'';comment:官方flighttask_progress status字段"`
	CurrentStep          int     `gorm:"column:current_step;default:0;comment:官方progress.current_step字段"`
	WaylineMissionState  int     `gorm:"column:wayline_mission_state;default:0;comment:官方ext.wayline_mission_state字段"`
	CurrentWaypointIndex int     `gorm:"column:current_waypoint_index;default:0;comment:官方ext.current_waypoint_index字段"`
	MediaCount           int     `gorm:"column:media_count;default:0;comment:官方ext.media_count字段"`
	ProgressPercent      float64 `gorm:"column:progress_percent;default:0;comment:官方progress.percent字段"`
	TrackId              string  `gorm:"column:track_id;type:varchar(64);index;not null;default:'';comment:官方ext.track_id字段"`
	WaylineId            int     `gorm:"column:wayline_id;default:0;comment:官方ext.wayline_id字段"`
	BreakPointJSON       string  `gorm:"column:break_point_json;type:jsonb;default:'{}';comment:官方ext.break_point原始JSON"`
}

func (DjiDockDeviceFlightTaskState) TableName() string {
	return "dji_dock_device_flight_task_state"
}

// DjiFlightTaskReady 是飞行任务就绪通知记录表。
//
// 功能：记录大疆 flighttask_ready 上行事件，机巢上报满足就绪条件的任务 ID 列表。
// 数据来源：thing/product/{gateway_sn}/events method=flighttask_ready。
// 写入策略：按事件逐条插入，保留历史记录。
// 使用场景：排查任务就绪推送、统计就绪任务数、后续任务编排扩展。
type DjiFlightTaskReady struct {
	gormx.LegacyBaseModel
	GatewaySn    string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;comment:网关机巢SN"`
	FlightIdJSON string    `gorm:"column:flight_id_json;type:jsonb;default:'[]';comment:就绪任务ID列表原始JSON"`
	EventJSON    string    `gorm:"column:event_json;type:jsonb;default:'{}';comment:完整flighttask_ready事件原始JSON"`
	FlightCount  int       `gorm:"column:flight_count;default:0;comment:就绪任务数量"`
	ReportedAt   time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
}

func (DjiFlightTaskReady) TableName() string { return "dji_flight_task_ready" }

// DjiRemoteLogEvent 是远程日志上传进度事件记录表。
//
// 功能：记录大疆 fileupload_progress 上行事件，追踪远程日志上传进度。
// 数据来源：thing/product/{gateway_sn}/events method=fileupload_progress。
// 写入策略：按事件逐条插入，原始数据完整保存在 EventJSON。
// 使用场景：排查日志上传进度、运维审计。
type DjiRemoteLogEvent struct {
	gormx.LegacyBaseModel
	GatewaySn  string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;comment:网关机巢SN"`
	Method     string    `gorm:"column:method;type:varchar(64);index;not null;default:'';comment:事件方法名(fileupload_progress)"`
	EventJSON  string    `gorm:"column:event_json;type:jsonb;default:'{}';comment:完整事件原始JSON"`
	FileCount  int       `gorm:"column:file_count;default:0;comment:文件数量"`
	ReportedAt time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
}

func (DjiRemoteLogEvent) TableName() string { return "dji_remote_log_event" }

// DjiReturnHomeEvent 大疆无人机返航信息事件。是返航事件记录表。
//
// 功能：记录大疆 return_home_info 上行事件，用于观察无人机返航过程和排查返航异常。
// 数据来源：thing/product/{gateway_sn}/events method=return_home_info。
// 写入策略：按事件逐条插入，原始数据完整保存在 EventJSON。
// 使用场景：机巢运维、返航过程追踪、后续告警/通知扩展。
type DjiReturnHomeEvent struct {
	gormx.LegacyBaseModel
	FlightId   string    `gorm:"column:flight_id;type:varchar(64);index;default:'';comment:关联航线任务ID"`
	GatewaySn  string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;comment:网关机巢SN"`
	ReportedAt time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	EventJSON  string    `gorm:"column:event_json;type:jsonb;default:'{}';comment:完整返航事件原始数据JSON"`

	HomeDockSn            string `gorm:"column:home_dock_sn;type:varchar(64);index;default:'';comment:返航目标机场SN，蛙跳任务场景下上报"`
	LastPointType         int    `gorm:"column:last_point_type;default:0;comment:返航路径最后一个点类型，0返航点上空，1非返航点上空"`
	PlannedPathPointCount int    `gorm:"column:planned_path_point_count;default:0;comment:规划返航轨迹点数量"`
}

func (DjiReturnHomeEvent) TableName() string { return "dji_return_home_event" }
