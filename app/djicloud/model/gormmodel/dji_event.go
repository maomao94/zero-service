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
	DeviceSn       string       `gorm:"column:device_sn;type:varchar(64);index;not null;default:'';comment:告警关联设备SN，无法识别时为空"`
	Level          int          `gorm:"column:level;index;not null;default:0;comment:告警等级"`
	Module         int          `gorm:"column:module;not null;default:0;comment:告警模块"`
	Code           string       `gorm:"column:code;type:varchar(64);not null;default:'';comment:大疆HMS告警码"`
	DeviceType     string       `gorm:"column:device_type;type:varchar(32);default:'';comment:告警设备类型"`
	Imminent       int          `gorm:"column:imminent;default:0;comment:是否紧急"`
	InTheSky       int          `gorm:"column:in_the_sky;default:0;comment:是否空中告警"`
	ComponentIndex int          `gorm:"column:component_index;default:0;comment:部件索引"`
	SensorIndex    int          `gorm:"column:sensor_index;default:0;comment:传感器索引"`
	Message        string       `gorm:"column:message;type:varchar(512);default:'';comment:告警描述"`
	Acked          int          `gorm:"column:acked;index;not null;default:0;comment:确认状态，0未确认，1已确认"`
	AckedAt        sql.NullTime `gorm:"column:acked_at;comment:确认时间"`
	AckedBy        string       `gorm:"column:acked_by;type:varchar(64);default:'';comment:确认人"`
	ReportedAt     time.Time    `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
}

func (DjiHmsAlert) TableName() string { return "dji_hms_alert" }

// DjiFlightTaskProgress 是飞行任务进度推送记录表。
//
// 功能：轻量记录大疆 flighttask_progress 上行事件，作为 Java 服务之外的 Go 侧补充记录。
// 数据来源：thing/product/{gateway_sn}/events method=flighttask_progress。
// 设计边界：本表只保存进度推送，不设计任务主表、不维护任务状态机、不承担任务编排。
// 使用场景：排查任务执行过程、查询最近航线任务进度、后续对接推送/看板。
type DjiFlightTaskProgress struct {
	gormx.LegacyBaseModel
	FlightId   string    `gorm:"column:flight_id;type:varchar(64);index;not null;comment:大疆航线任务ID"`
	GatewaySn  string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;comment:网关机巢SN"`
	ReportedAt time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	ExtJSON    string    `gorm:"column:ext_json;type:jsonb;default:'{}';comment:完整任务进度原始数据JSON"`

	WaylineMissionState  int     `gorm:"column:wayline_mission_state;default:0;comment:航线任务状态"`
	CurrentWaypointIndex int     `gorm:"column:current_waypoint_index;default:0;comment:当前航点索引"`
	MediaCount           int     `gorm:"column:media_count;default:0;comment:媒体数量"`
	ProgressPercent      float64 `gorm:"column:progress_percent;default:0;comment:进度百分比"`
}

func (DjiFlightTaskProgress) TableName() string { return "dji_flight_task_progress" }

// DjiReturnHomeEvent 是返航事件记录表。
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

	HomeDockSn             string  `gorm:"column:home_dock_sn;type:varchar(64);index;default:'';comment:返航目标机场SN，蛙跳任务场景下上报"`
	LastPointType          int     `gorm:"column:last_point_type;default:0;comment:返航路径最后一个点类型，0返航点上空，1非返航点上空"`
	PlannedPathPointCount  int     `gorm:"column:planned_path_point_count;default:0;comment:规划返航轨迹点数量"`
	MultiDockHomeInfoCount int     `gorm:"column:multi_dock_home_info_count;default:0;comment:蛙跳任务机场返航信息数量"`
	NearestHomeDistance    float64 `gorm:"column:nearest_home_distance;default:0;comment:蛙跳机场返航信息中的最近home点距离"`
}

func (DjiReturnHomeEvent) TableName() string { return "dji_return_home_event" }
