package gormmodel

import (
	"database/sql"
	"time"

	"zero-service/common/gormx"
)

// DjiDeviceOsdSnapshot 是设备 OSD 遥测快照表。
//
// 功能：每个设备只保留最近一次 OSD 上报，用于设备态势感知、地图定位、电量/姿态展示和在线状态辅助判断。
// 数据来源：thing/product/{device_sn}/osd；机巢和无人机会分别上报 OSD。
// 写入策略：按 device_sn Upsert 覆盖最新值，不保存历史时序，历史轨迹后续如有需要再单独建表。
// 使用场景：设备详情、机巢/无人机实时状态面板、控制前状态检查。
type DjiDeviceOsdSnapshot struct {
	gormx.LegacyBaseModel
	DeviceSn     string    `gorm:"column:device_sn;uniqueIndex;type:varchar(64);not null;comment:设备SN，机巢或无人机"`
	GatewaySn    string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;default:'';comment:所属机巢SN"`
	DeviceDomain string    `gorm:"column:device_domain;index;type:varchar(8);not null;default:'';comment:大疆设备领域domain，0飞机类，1负载类，2遥控器类，3机场类"`
	ReportedAt   time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	DataJSON     string    `gorm:"column:data_json;type:jsonb;default:'{}';comment:完整OSD原始数据JSON"`

	Latitude        float64 `gorm:"column:latitude;default:0;comment:纬度"`
	Longitude       float64 `gorm:"column:longitude;default:0;comment:经度"`
	Altitude        float64 `gorm:"column:altitude;default:0;comment:海拔高度"`
	Height          float64 `gorm:"column:height;default:0;comment:相对起飞点高度"`
	SpeedH          float64 `gorm:"column:speed_h;default:0;comment:水平速度"`
	SpeedV          float64 `gorm:"column:speed_v;default:0;comment:垂直速度"`
	Heading         float64 `gorm:"column:heading;default:0;comment:航向角"`
	AttitudePitch   float64 `gorm:"column:attitude_pitch;default:0;comment:俯仰角"`
	AttitudeRoll    float64 `gorm:"column:attitude_roll;default:0;comment:横滚角"`
	AttitudeHeading float64 `gorm:"column:attitude_heading;default:0;comment:机头朝向"`
	BatteryPercent  int     `gorm:"column:battery_percent;default:0;comment:电池电量百分比"`
	Elevation       float64 `gorm:"column:elevation;default:0;comment:设备所在海拔或高程"`
	ModeCode        int     `gorm:"column:mode_code;default:0;comment:设备模式码"`
}

func (DjiDeviceOsdSnapshot) TableName() string { return "dji_device_osd_snapshot" }

func (s *DjiDeviceOsdSnapshot) IsDock() bool {
	return s.DeviceDomain == DjiDeviceDomainDock
}

// DjiDeviceStateSnapshot 是设备 State 状态快照表。
//
// 功能：每个设备只保留最近一次 State 上报，用于记录机巢盖子状态、子设备挂载/在线状态、直播状态、负载状态等非高频遥测信息。
// 数据来源：thing/product/{device_sn}/state。
// 写入策略：按 device_sn Upsert 覆盖最新值。
// 使用场景：机巢管理页展示机巢工作状态、判断无人机是否挂载/在线、后续补充控制前置校验。
type DjiDeviceStateSnapshot struct {
	gormx.LegacyBaseModel
	DeviceSn     string    `gorm:"column:device_sn;uniqueIndex;type:varchar(64);not null;comment:设备SN，机巢或无人机"`
	GatewaySn    string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;default:'';comment:所属机巢SN"`
	DeviceDomain string    `gorm:"column:device_domain;index;type:varchar(8);not null;default:'';comment:大疆设备领域domain，0飞机类，1负载类，2遥控器类，3机场类"`
	ReportedAt   time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	DataJSON     string    `gorm:"column:data_json;type:jsonb;default:'{}';comment:完整State原始数据JSON"`

	SubDeviceSn     sql.NullString `gorm:"column:sub_device_sn;type:varchar(64);comment:机巢state里携带的当前子设备SN"`
	SubDeviceOnline sql.NullBool   `gorm:"column:sub_device_online;comment:机巢state里携带的子设备在线状态"`
}

func (DjiDeviceStateSnapshot) TableName() string { return "dji_device_state_snapshot" }
