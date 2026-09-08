package svc

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"zero-service/app/live/model/gormmodel"
	"zero-service/common/gormx"

	"gorm.io/gorm"
)

// ErrMeetingNotFound 会议不存在。
var ErrMeetingNotFound = errors.New("meeting not found")

// MeetingRepo 会议与参会记录的持久化存取。
type MeetingRepo struct {
	db *gormx.DB
}

func NewMeetingRepo(db *gormx.DB) *MeetingRepo {
	return &MeetingRepo{db: db}
}

// CreateMeeting 创建会议单据。
func (r *MeetingRepo) CreateMeeting(ctx context.Context, m *gormmodel.LiveMeeting) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// GetMeeting 按会议号查询会议单据。
func (r *MeetingRepo) GetMeeting(ctx context.Context, meetingNo string) (*gormmodel.LiveMeeting, error) {
	var m gormmodel.LiveMeeting
	err := r.db.WithContext(ctx).Where("meeting_no = ?", meetingNo).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMeetingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMeetingByCode 按用户会议号（9位）查询会议单据。
func (r *MeetingRepo) GetMeetingByCode(ctx context.Context, meetingCode string) (*gormmodel.LiveMeeting, error) {
	var m gormmodel.LiveMeeting
	err := r.db.WithContext(ctx).Where("meeting_code = ?", meetingCode).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMeetingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// IsMeetingCodeExists 检查用户会议号是否已存在（排除已结束的会议）。
func (r *MeetingRepo) IsMeetingCodeExists(ctx context.Context, meetingCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&gormmodel.LiveMeeting{}).
		Where("meeting_code = ? AND status != ?", meetingCode, gormmodel.MeetingStatusEnded).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateMeetingEnded 标记会议已结束（仅 active → ended，返回是否更新成功）。
// operator/deptCode 非空时一并记录更新人/机构（webhook 场景为空字符串）。
func (r *MeetingRepo) UpdateMeetingEnded(ctx context.Context, meetingNo string, endedAt time.Time, operator, deptCode string) (bool, error) {
	updates := map[string]any{
		"status":   gormmodel.MeetingStatusEnded,
		"end_time": sql.NullTime{Time: endedAt, Valid: true},
	}
	if operator != "" {
		updates["update_user"] = sql.NullString{String: operator, Valid: true}
	}
	if deptCode != "" {
		updates["dept_code"] = sql.NullString{String: deptCode, Valid: true}
	}
	res := r.db.WithContext(ctx).Model(&gormmodel.LiveMeeting{}).
		Where("meeting_no = ? AND status = ?", meetingNo, gormmodel.MeetingStatusActive).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// MeetingListQuery 会议列表查询条件。
type MeetingListQuery struct {
	Status          int32
	Page            int64
	PageSize        int64
	CreateTimeStart string
	CreateTimeEnd   string
	DeptCode        string
	CreateUser      string
	Title           string
	// Identity 参会人身份（非空时只查该用户作为参会人的会议）
	Identity string
}

// ListMeetings 分页查询会议列表；status 为 0 时不过滤。
func (r *MeetingRepo) ListMeetings(ctx context.Context, q *MeetingListQuery) ([]gormmodel.LiveMeeting, int64, error) {
	db := r.db.WithContext(ctx).Model(&gormmodel.LiveMeeting{})
	if q.Identity != "" {
		// 只查询该用户作为参会人的会议
		subq := r.db.WithContext(ctx).Model(&gormmodel.LiveMeetingParticipant{}).
			Where("identity = ?", q.Identity).
			Select("meeting_no")
		db = db.Where("meeting_no IN (?)", subq)
	}
	if q.Status > 0 {
		db = db.Where("status = ?", q.Status)
	}
	if q.DeptCode != "" {
		db = db.Where("dept_code = ?", q.DeptCode)
	}
	if q.CreateUser != "" {
		db = db.Where("create_user = ?", q.CreateUser)
	}
	if q.Title != "" {
		db = db.Where("title LIKE ?", "%"+q.Title+"%")
	}
	if q.CreateTimeStart != "" {
		db = db.Where("create_time >= ?", q.CreateTimeStart)
	}
	if q.CreateTimeEnd != "" {
		db = db.Where("create_time < ?", q.CreateTimeEnd+" 23:59:59")
	}
	var meetings []gormmodel.LiveMeeting
	page, err := gormx.QueryPage(db.Order("create_time DESC"), q.Page, q.PageSize, &meetings)
	if err != nil {
		return nil, 0, err
	}
	return meetings, page.Total, nil
}

// IsParticipantInMeeting 校验指定参会者是否仍在会议中（status=joined）。
func (r *MeetingRepo) IsParticipantInMeeting(ctx context.Context, meetingNo, identity string) bool {
	var count int64
	r.db.WithContext(ctx).
		Model(&gormmodel.LiveMeetingParticipant{}).
		Where("meeting_no = ? AND identity = ? AND status = ?", meetingNo, identity, gormmodel.ParticipantStatusJoined).
		Count(&count)
	return count > 0
}

// UpsertParticipant 插入或更新参会记录（同一会议同一身份；冲突时保留首次 join_time）。
// 使用 Where + Assign + FirstOrCreate（有则更新 Assign 字段，无则插入），
// 不依赖 ON CONFLICT（高斯数据库兼容），与 djicloud 写入模式一致。
func (r *MeetingRepo) UpsertParticipant(ctx context.Context, p *gormmodel.LiveMeetingParticipant) error {
	updateData := map[string]any{
		"name":      p.Name,
		"status":    p.Status,
		"left_time": p.LeftTime,
	}
	return r.db.Transact(func(tx *gormx.DB) error {
		return tx.WithContext(ctx).
			Where(map[string]any{"meeting_no": p.MeetingNo, "identity": p.Identity}).
			Assign(updateData).
			FirstOrCreate(p).Error
	})
}

// MarkParticipantLeft 标记参与者已离开。
func (r *MeetingRepo) MarkParticipantLeft(ctx context.Context, meetingNo, identity string, leftAt time.Time) error {
	return r.db.WithContext(ctx).Model(&gormmodel.LiveMeetingParticipant{}).
		Where("meeting_no = ? AND identity = ?", meetingNo, identity).
		Updates(map[string]any{
			"status":    gormmodel.ParticipantStatusLeft,
			"left_time": sql.NullTime{Time: leftAt, Valid: true},
		}).Error
}

// ListParticipants 查询会议全部参会记录。
func (r *MeetingRepo) ListParticipants(ctx context.Context, meetingNo string) ([]gormmodel.LiveMeetingParticipant, error) {
	var participants []gormmodel.LiveMeetingParticipant
	err := r.db.WithContext(ctx).
		Where("meeting_no = ?", meetingNo).
		Order("join_time ASC").
		Find(&participants).Error
	return participants, err
}

// CreateMessage 创建聊天消息。
func (r *MeetingRepo) CreateMessage(ctx context.Context, m *gormmodel.LiveMeetingMessage) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// ListMessages 分页查询会议聊天记录。
func (r *MeetingRepo) ListMessages(ctx context.Context, meetingNo string, page, pageSize int64) ([]gormmodel.LiveMeetingMessage, int64, error) {
	q := r.db.WithContext(ctx).Model(&gormmodel.LiveMeetingMessage{}).
		Where("meeting_no = ?", meetingNo)
	var messages []gormmodel.LiveMeetingMessage
	pageRes, err := gormx.QueryPage(q.Order("create_time DESC"), page, pageSize, &messages)
	if err != nil {
		return nil, 0, err
	}
	return messages, pageRes.Total, nil
}

// ===== SIP Provider =====

// CreateSipProvider 创建 SIP 供应商。
func (r *MeetingRepo) CreateSipProvider(ctx context.Context, p *gormmodel.LiveSipProvider) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// GetSipProviderByCode 按编码查询 SIP 供应商。
func (r *MeetingRepo) GetSipProviderByCode(ctx context.Context, code string) (*gormmodel.LiveSipProvider, error) {
	var p gormmodel.LiveSipProvider
	err := r.db.WithContext(ctx).Where("code = ? AND status = ?", code, gormmodel.SipProviderStatusEnabled).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListSipProviders 列出所有 SIP 供应商。
func (r *MeetingRepo) ListSipProviders(ctx context.Context) ([]gormmodel.LiveSipProvider, error) {
	var providers []gormmodel.LiveSipProvider
	err := r.db.WithContext(ctx).Order("create_time DESC").Find(&providers).Error
	return providers, err
}

// DeleteSipProvider 删除 SIP 供应商。
func (r *MeetingRepo) DeleteSipProvider(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&gormmodel.LiveSipProvider{}, "id = ?", id).Error
}

// UpdateSipProvider 更新 SIP 供应商。
func (r *MeetingRepo) UpdateSipProvider(ctx context.Context, id string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&gormmodel.LiveSipProvider{}).
		Where("id = ?", id).Updates(updates).Error
}

// GetSipProviderByID 按 ID 查询 SIP 供应商。
func (r *MeetingRepo) GetSipProviderByID(ctx context.Context, id string) (*gormmodel.LiveSipProvider, error) {
	var p gormmodel.LiveSipProvider
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
