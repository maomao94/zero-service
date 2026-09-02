package gormmodel

import (
	"database/sql"
	"time"

	"zero-service/common/gormx"
)

// MeetingStatus 会议状态。
const (
	// MeetingStatusCreated 已创建（创建即进行中，保留枚举位便于扩展"未开始"状态）
	MeetingStatusCreated = int(1)
	// MeetingStatusActive 进行中
	MeetingStatusActive = int(2)
	// MeetingStatusEnded 已结束
	MeetingStatusEnded = int(3)
)

// LiveMeeting 会议单据（业务会议号 = LiveKit 房间名）。
// 表风格与 app/trigger/model/gormmodel 的 plan/plan_batch 完全一致：
// LegacyStringBaseModel（string 主键 + create_time/update_time + 软删除）
// + VersionMixin（乐观锁）+ CreateUser/UpdateUser/DeptCode（取自 gRPC
// metadata 的 user-id，不通过 proto 传输）。
type LiveMeeting struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin

	CreateUser sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
	// 业务会议号（= LiveKit 房间名，全局唯一）
	MeetingNo string `gorm:"column:meeting_no;size:32;comment:业务会议号;uniqueIndex:uq_live_meetings_meeting_no"`
	// 会议标题
	Title string `gorm:"column:title;size:128;comment:会议标题"`
	// 状态：1-已创建，2-进行中，3-已结束
	Status int `gorm:"column:status;comment:状态:1-已创建,2-进行中,3-已结束;index:idx_live_meetings_status"`
	// 开始时间
	StartTime time.Time `gorm:"column:start_time;comment:开始时间"`
	// 结束时间
	EndTime sql.NullTime `gorm:"column:end_time;comment:结束时间;index:idx_live_meetings_end_time"`
}

func (LiveMeeting) TableName() string {
	return "live_meetings"
}

// ParticipantStatus 参会状态。
const (
	// ParticipantStatusJoined 已加入
	ParticipantStatusJoined = int(1)
	// ParticipantStatusLeft 已离开
	ParticipantStatusLeft = int(2)
)

// LiveMeetingParticipant 参会记录（同一会议同一身份唯一）。
type LiveMeetingParticipant struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin

	CreateUser sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
	// 会议号（冗余房间名，便于 webhook 反查）
	MeetingNo string `gorm:"column:meeting_no;size:32;comment:会议号;uniqueIndex:uq_live_meeting_participants_meeting_identity,priority:1;index:idx_live_meeting_participants_meeting"`
	// 参与者身份（房间内唯一）
	Identity string `gorm:"column:identity;size:64;comment:参与者身份;uniqueIndex:uq_live_meeting_participants_meeting_identity,priority:2"`
	// 展示名
	Name string `gorm:"column:name;size:64;comment:展示名"`
	// 状态：1-已加入，2-已离开
	Status int `gorm:"column:status;comment:状态:1-已加入,2-已离开;index:idx_live_meeting_participants_status"`
	// 加入时间
	JoinTime time.Time `gorm:"column:join_time;comment:加入时间"`
	// 离开时间
	LeftTime sql.NullTime `gorm:"column:left_time;comment:离开时间"`
}

func (LiveMeetingParticipant) TableName() string {
	return "live_meeting_participants"
}

// LiveMeetingMessage 会议聊天消息。
type LiveMeetingMessage struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	MeetingNo   string    `gorm:"column:meeting_no;size:32;not null;index:idx_live_meeting_messages_meeting"`
	MessageID   string    `gorm:"column:message_id;size:64;not null;uniqueIndex"`
	SenderID    string    `gorm:"column:sender_id;size:64;not null"`
	SenderName  string    `gorm:"column:sender_name;size:64"`
	Content     string    `gorm:"column:content;type:text;not null"`
	MessageType string    `gorm:"column:message_type;size:32;default:text"`
	CreateTime  time.Time `gorm:"column:create_time;autoCreateTime"`
}

func (LiveMeetingMessage) TableName() string {
	return "live_meeting_messages"
}