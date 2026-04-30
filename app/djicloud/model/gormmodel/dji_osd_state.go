package gormmodel

import (
	"time"

	"zero-service/common/gormx"
)

// DjiDeviceOsdSnapshot 是设备 OSD 遥测快照表。
//
// 功能：每个设备只保留最近一次 OSD 上报，用于设备态势感知、地图定位、电量/姿态展示和在线状态刷新。
// 数据来源：thing/product/{device_sn}/osd；保存 DJI 物模型中 pushMode=0 的定频属性快照。
// 写入策略：按 device_sn Upsert 覆盖最新值，不保存历史时序；历史轨迹后续如有需要再单独建表。
// 使用场景：设备详情、机巢/无人机实时状态面板、控制前状态检查。
type DjiDeviceOsdSnapshot struct {
	gormx.LegacyBaseModel
	DeviceSn   string    `gorm:"column:device_sn;uniqueIndex;type:varchar(64);not null;comment:设备SN，机巢或无人机"`
	GatewaySn  string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;default:'';comment:所属机巢SN"`
	ReportedAt time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	RawJSON    string    `gorm:"column:raw_json;type:jsonb;default:'{}';comment:完整OSD原始数据JSON"`
}

func (DjiDeviceOsdSnapshot) TableName() string { return "dji_device_osd_snapshot" }

// DjiDeviceStateSnapshot 是设备 State 状态快照表。
//
// 功能：每个设备只保留最近一次 State 上报，用于记录机巢盖子状态、子设备挂载状态、直播状态、负载状态、固件/硬件版本等非定频物模型状态。
// 数据来源：thing/product/{device_sn}/state；保存 DJI 物模型中 pushMode=1 的状态变化属性快照。
// 写入策略：按 device_sn Upsert 覆盖最新值；State 不刷新设备在线状态，在线状态以 OSD 有效上行为准。
// 使用场景：机巢管理页展示机巢工作状态、判断无人机挂载关系、补充控制前置校验。
type DjiDeviceStateSnapshot struct {
	gormx.LegacyBaseModel
	DeviceSn   string    `gorm:"column:device_sn;uniqueIndex;type:varchar(64);not null;comment:设备SN，机巢或无人机"`
	GatewaySn  string    `gorm:"column:gateway_sn;type:varchar(64);index;not null;default:'';comment:所属机巢SN"`
	ReportedAt time.Time `gorm:"column:reported_at;index;not null;comment:设备上报时间"`
	RawJSON    string    `gorm:"column:raw_json;type:jsonb;default:'{}';comment:完整State原始数据JSON"`
}

func (DjiDeviceStateSnapshot) TableName() string { return "dji_device_state_snapshot" }
