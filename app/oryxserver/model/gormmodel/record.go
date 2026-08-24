package gormmodel

import (
	"time"

	"zero-service/common/gormx"
)

// Record Oryx 录制任务生命周期记录（由 gtw 回调 on_record_begin/on_record_end 落库）。
type Record struct {
	gormx.LegacyStringBaseModel

	// 录制任务 UUID（Oryx 生成，唯一）
	UUID string `gorm:"column:uuid;size:64;comment:录制任务UUID;uniqueIndex:uq_oryx_record_uuid"`
	// 虚拟主机
	Vhost string `gorm:"column:vhost;size:128;comment:虚拟主机;index:idx_oryx_record_vhost"`
	// 应用名
	App string `gorm:"column:app;size:128;comment:应用名;index:idx_oryx_record_app"`
	// 流名称
	Stream string `gorm:"column:stream;size:256;comment:流名称;index:idx_oryx_record_stream"`
	// 回调透传凭证
	Opaque string `gorm:"column:opaque;size:256;comment:回调透传凭证"`
	// 录制状态：1-录制中，2-已完成，3-失败
	Status int32 `gorm:"column:status;comment:录制状态:1-录制中,2-已完成,3-失败;index:idx_oryx_record_status"`
	// 开始时间
	BeginTime time.Time `gorm:"column:begin_time;type:timestamp;comment:开始时间;index:idx_oryx_record_begin_time"`
	// 结束时间
	EndTime time.Time `gorm:"column:end_time;type:timestamp;comment:结束时间;index:idx_oryx_record_end_time"`
	// 产物错误码，0 表示成功
	ArtifactCode int32 `gorm:"column:artifact_code;comment:产物错误码,0表示成功"`
	// 产物容器内路径
	ArtifactPath string `gorm:"column:artifact_path;size:512;comment:产物容器内路径"`
	// 产物播放地址
	ArtifactURL string `gorm:"column:artifact_url;size:512;comment:产物播放地址"`
}

func (Record) TableName() string {
	return "oryx_record"
}

// 录制状态
const (
	// RecordStatusRecording 录制中
	RecordStatusRecording = int32(1)
	// RecordStatusCompleted 已完成
	RecordStatusCompleted = int32(2)
	// RecordStatusFailed 失败
	RecordStatusFailed = int32(3)
)
