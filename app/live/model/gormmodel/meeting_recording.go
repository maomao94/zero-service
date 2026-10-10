package gormmodel

import (
	"database/sql"
	"time"

	"zero-service/common/gormx"
)

// 录制状态直接对齐 LiveKit EgressStatus（github.com/livekit/protocol/livekit.EgressStatus），
// 不做业务映射，便于直接对照 Egress 排查问题。
// 进行中：STARTING(0)/ACTIVE(1)/ENDING(2)；终态：COMPLETE(3)/FAILED(4)/ABORTED(5)/LIMIT_REACHED(6)。
const (
	// RecordingStatusStarting Egress 启动中
	RecordingStatusStarting = int(0)
	// RecordingStatusActive Egress 录制中
	RecordingStatusActive = int(1)
	// RecordingStatusEnding Egress 收尾中（已停止，等待生成文件）
	RecordingStatusEnding = int(2)
	// RecordingStatusComplete Egress 已完成
	RecordingStatusComplete = int(3)
	// RecordingStatusFailed Egress 失败
	RecordingStatusFailed = int(4)
	// RecordingStatusAborted Egress 已中止
	RecordingStatusAborted = int(5)
	// RecordingStatusLimitReached Egress 超出时长/配额限制而结束
	RecordingStatusLimitReached = int(6)
)

// RecordingStatusIsActive 判断状态是否为进行中（STARTING/ACTIVE/ENDING）。
func RecordingStatusIsActive(status int) bool {
	return status >= RecordingStatusStarting && status <= RecordingStatusEnding
}

// RecordingStatusIsTerminal 判断状态是否为终态（COMPLETE/FAILED/ABORTED/LIMIT_REACHED）。
func RecordingStatusIsTerminal(status int) bool {
	return status >= RecordingStatusComplete
}

// LiveMeetingRecording 会议录制记录（一次录制 = 一个 LiveKit Egress 任务）。
// 表风格与 LiveMeeting 一致：LegacyStringBaseModel（string 主键 + 时间 + 软删除）
// + CreateUser/UpdateUser/DeptCode（取自 gRPC metadata）。
type LiveMeetingRecording struct {
	gormx.LegacyStringBaseModel

	CreateUser sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
	// 会议号（= LiveKit 房间名）
	MeetingNo string `gorm:"column:meeting_no;size:32;not null;comment:会议号;index:idx_live_meeting_recordings_meeting"`
	// LiveKit Egress 任务 ID（唯一，webhook 按此反查）
	EgressId string `gorm:"column:egress_id;size:64;not null;comment:Egress任务ID;uniqueIndex:uq_live_meeting_recordings_egress_id"`
	// LiveKit 房间名（冗余，便于 webhook 反查）
	RoomName string `gorm:"column:room_name;size:64;comment:房间名"`
	// 录制状态（对齐 LiveKit EgressStatus：0-启动中,1-录制中,2-收尾中,3-已完成,4-失败,5-已中止,6-超限）
	Status int `gorm:"column:status;not null;default:1;comment:状态(对齐EgressStatus):0-启动中,1-录制中,2-收尾中,3-已完成,4-失败,5-已中止,6-超限;index:idx_live_meeting_recordings_status"`
	// 录制文件名（相对输出根目录，用于拼接播放地址）
	FileName string `gorm:"column:file_name;size:256;comment:录制文件名"`
	// 录制文件路径（Egress 返回的原始位置/本地路径，仅内部使用，不对外返回）
	FilePath string `gorm:"column:file_path;size:512;comment:录制文件路径"`
	// 文件大小（字节）
	FileSize int64 `gorm:"column:file_size;comment:文件大小(字节)"`
	// 录制时长（秒）
	Duration int64 `gorm:"column:duration;comment:录制时长(秒)"`
	// 录制开始时间
	StartTime time.Time `gorm:"column:start_time;comment:开始时间"`
	// 录制结束时间
	EndTime sql.NullTime `gorm:"column:end_time;comment:结束时间"`
	// 失败原因（失败时有值）
	Error string `gorm:"column:error;size:512;comment:失败原因"`
}

func (LiveMeetingRecording) TableName() string {
	return "live_meeting_recordings"
}
