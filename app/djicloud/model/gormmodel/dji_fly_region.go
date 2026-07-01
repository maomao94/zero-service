package gormmodel

import "zero-service/common/gormx"

// DjiFlyRegion 是大疆自定义飞行区配置记录表。
//
// 功能：记录每次通过平台接口设置的飞行区 GeoJSON 文件信息（OSS 地址、校验值等）。
// 数据来源：SetCustomFlyRegion RPC 生成 GeoJSON 上传 OSS 后写入。
// 写入策略：Insert-only，每次设置新建记录，历史可追溯；查最新按 create_time DESC。
// 使用场景：设备 flight_areas_get 查询当前文件列表、飞行区配置审计回溯。
type DjiFlyRegion struct {
	gormx.LegacyBaseModel
	GatewaySn    string `gorm:"column:gateway_sn;index;not null;comment:目标机巢序列号"`
	Name         string `gorm:"column:name;type:varchar(128);not null;default:'';comment:用户自定义飞行区名称（来自 gRPC 请求，如"航点A区""禁飞区-1号"）"`
	FileId       string `gorm:"column:file_id;type:varchar(64);uniqueIndex;not null;comment:文件唯一标识(UUID)，用于文件名中的 uuid"`
	BucketName   string `gorm:"column:bucket_name;uniqueIndex:idx_bucket_file;not null;default:'';comment:OSS 存储桶名称"`
	FileName     string `gorm:"column:file_name;uniqueIndex:idx_bucket_file;not null;comment:OSS 文件对象 key（自动生成，如 dfence_abc123.json）"`
	FileSize     int64  `gorm:"column:file_size;not null;default:0;comment:文件大小（字节）"`
	Checksum     string `gorm:"column:checksum;type:varchar(128);not null;default:'';comment:SHA256 校验值"`
	GeofenceJSON string `gorm:"column:geofence_json;type:text;default:'';comment:原始 GeoJSON 内容"`
}

func (DjiFlyRegion) TableName() string {
	return "dji_fly_region"
}

// DjiFlyRegionSyncStatus 是大疆飞行区文件同步状态表。
//
// 功能：记录每次同步事件的上报状态，用于追溯历史同步过程。
// 数据来源：flight_areas_sync_progress 事件上报时按 gateway_sn + file_name 匹配 DjiFlyRegion 后插入。
// 写入策略：Insert-only，每次同步事件创建新记录，不更新已有记录，完整保留同步历史。
// 使用场景：分页查询同步历史；排障时回溯设备同步过程。
type DjiFlyRegionSyncStatus struct {
	gormx.LegacyBaseModel
	GatewaySn   string `gorm:"column:gateway_sn;index;not null;comment:机巢序列号"`
	FlyRegionID int64  `gorm:"column:fly_region_id;index;not null;default:0;comment:关联 dji_fly_region.id"`
	SyncStatus  string `gorm:"column:sync_status;type:varchar(32);not null;default:'';comment:同步状态(notified/synchronized/failed)"`
	SyncReason  int    `gorm:"column:sync_reason;not null;default:0;comment:同步失败原因码，0 表示无异常"`
}

func (DjiFlyRegionSyncStatus) TableName() string {
	return "dji_fly_region_sync_status"
}
