package gormmodel

import (
	"database/sql"
	"time"

	"zero-service/common/gormx"
)

const (
	DjiDeviceDomainAircraft = "0"
	DjiDeviceDomainPayload  = "1"
	DjiDeviceDomainRemote   = "2"
	DjiDeviceDomainDock     = "3"
)

// DjiDevice 是大疆云平台设备主表。
//
// 功能：统一记录机巢、无人机、相机/负载等所有出现过的设备，作为云平台设备管理、在线状态查询、控制接口入参校验的基础表。
// 数据来源：
//   - 机巢设备来自 sys/product/{gateway_sn}/status 的 update_topo 上行；
//   - 子设备来自 update_topo.sub_devices、thing/product/{device_sn}/osd、thing/product/{device_sn}/state；
//   - 固件版本、硬件版本从 state 物模型快照中提取，对应 DJI 物模型 pushMode=1 的状态数据，空值不上屏覆盖；
//   - 在线状态由 osd 上行刷新，内存缓存用于高频在线判断，数据库保存最新在线快照并按 LastOnlineAt 懒过期清理。
//
// 蛙跳场景：同一架飞机可能先后或同时与多个机巢建立拓扑关系，多机巢绑定关系不放在本表表达，而由 DjiDeviceTopo 按 gateway_sn + sub_device_sn 保存；本表 GatewaySn 仅表示最近一次上报归属的机巢，用于快速展示和控制路由兜底。
// 在线状态：数据库默认 IsOnline=false，表示离线或尚未收到在线上报；收到设备 osd 有效上行后置为 true，status/update_topo 与 state 仅维护设备归属和状态快照。
// 使用场景：设备列表、机巢详情、无人机详情、设备在线判断、后续设备分组/别名管理。
type DjiDevice struct {
	gormx.LegacyBaseModel
	DeviceSn        string       `gorm:"column:device_sn;type:varchar(64);uniqueIndex;not null;comment:设备SN，机巢/无人机/负载设备唯一标识"`
	GatewaySn       string       `gorm:"column:gateway_sn;type:varchar(64);index;not null;default:'';comment:最近一次上报关联的网关机巢SN，机巢自身等于device_sn；蛙跳多绑定关系以dji_device_topo为准"`
	Alias           string       `gorm:"column:alias;type:varchar(128);default:'';comment:设备别名"`
	GroupName       string       `gorm:"column:group_name;type:varchar(128);default:'';comment:业务分组"`
	FirmwareVersion string       `gorm:"column:firmware_version;type:varchar(64);default:'';comment:固件版本"`
	HardwareVersion string       `gorm:"column:hardware_version;type:varchar(64);default:'';comment:硬件版本"`
	IsOnline        bool         `gorm:"column:is_online;index;not null;default:false;comment:当前是否在线，数据库默认false，收到有效在线上行后置true"`
	FirstOnlineAt   sql.NullTime `gorm:"column:first_online_at;comment:首次上线时间"`
	LastOnlineAt    sql.NullTime `gorm:"column:last_online_at;comment:最后在线时间"`
}

func (DjiDevice) TableName() string { return "dji_device" }

func (d *DjiDevice) TouchOnline(now time.Time) {
	d.IsOnline = true
	if !d.FirstOnlineAt.Valid {
		d.FirstOnlineAt = sql.NullTime{Time: now, Valid: true}
	}
	d.LastOnlineAt = sql.NullTime{Time: now, Valid: true}
}

// DjiDeviceTopo 是机巢与子设备的拓扑关系表。
//
// 功能：记录“机巢 gateway_sn -> 子设备 sub_device_sn”的绑定关系，解决一台机巢下挂无人机、相机、负载等设备的关系查询问题。
// 数据来源：sys/product/{gateway_sn}/status method=update_topo。
// 使用场景：机巢详情页展示子设备、按机巢查询无人机、控制无人机/云台前定位所属网关。
// 蛙跳场景：同一架飞机可以被多个机巢绑定，因此本表允许同一个 sub_device_sn 出现在多条不同 gateway_sn 记录中；业务查询按 gateway_sn + sub_device_sn 唯一确定某一次机巢绑定关系。
// 约束：同一个 gateway_sn + sub_device_sn 唯一，不对 sub_device_sn 单独做唯一约束。
type DjiDeviceTopo struct {
	gormx.LegacyBaseModel
	GatewaySn        string `gorm:"column:gateway_sn;uniqueIndex:idx_topo_pair;type:varchar(64);not null;comment:网关机巢SN"`
	SubDeviceSn      string `gorm:"column:sub_device_sn;uniqueIndex:idx_topo_pair;index:idx_topo_sub;type:varchar(64);not null;comment:子设备SN"`
	Domain           string `gorm:"column:domain;type:varchar(8);not null;default:'';comment:大疆设备领域domain，0飞机类，1负载类，2遥控器类，3机场类"`
	SubDeviceType    int    `gorm:"column:sub_device_type;not null;default:0;comment:子设备类型"`
	SubDeviceSubType int    `gorm:"column:sub_device_sub_type;not null;default:0;comment:子设备子类型"`
	SubDeviceIndex   string `gorm:"column:sub_device_index;type:varchar(32);default:'';comment:子设备索引"`
	ThingVersion     string `gorm:"column:thing_version;type:varchar(64);default:'';comment:物模型版本"`
}

func (DjiDeviceTopo) TableName() string { return "dji_device_topo" }
