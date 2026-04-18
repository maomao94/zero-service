package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/einox/protocol"
	"zero-service/common/gormx"
)

// GormxStore 会话与中断元数据的关系库实现，供多实例 aisolo 共享。
type GormxStore struct {
	db                    *gormx.DB
	nullLeaseRecoverGrace time.Duration
}

// NewGormxStore 创建存储；db 不可为空。nullLeaseRecoverGrace 为零时使用 2 分钟。
func NewGormxStore(db *gormx.DB, nullLeaseRecoverGrace time.Duration) (*GormxStore, error) {
	if db == nil {
		return nil, fmt.Errorf("session.gormx: db is nil")
	}
	if nullLeaseRecoverGrace <= 0 {
		nullLeaseRecoverGrace = 2 * time.Minute
	}
	s := &GormxStore{db: db, nullLeaseRecoverGrace: nullLeaseRecoverGrace}
	if err := s.db.DB.AutoMigrate(&sessionRow{}, &interruptRow{}); err != nil {
		return nil, fmt.Errorf("session.gormx: migrate: %w", err)
	}
	return s, nil
}

func (s *GormxStore) CreateSession(ctx context.Context, sess *Session) error {
	if sess == nil || sess.ID == "" || sess.UserID == "" {
		return fmt.Errorf("session.gormx: empty id/user")
	}
	row, err := sessionToRow(sess)
	if err != nil {
		return err
	}
	return s.db.DB.WithContext(ctx).Create(row).Error
}

func (s *GormxStore) GetSession(ctx context.Context, userID, sessionID string) (*Session, error) {
	var row sessionRow
	err := s.db.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("session: not found")
	}
	if err != nil {
		return nil, fmt.Errorf("session.gormx: get: %w", err)
	}
	return rowToSession(&row)
}

func (s *GormxStore) UpdateSession(ctx context.Context, sess *Session) error {
	if sess == nil || sess.ID == "" {
		return fmt.Errorf("session.gormx: empty")
	}
	row, err := sessionToRow(sess)
	if err != nil {
		return err
	}
	res := s.db.DB.WithContext(ctx).Save(row)
	if res.Error != nil {
		return fmt.Errorf("session.gormx: update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("session: not found")
	}
	return nil
}

func (s *GormxStore) ListSessions(ctx context.Context, userID string, page, pageSize int) ([]*Session, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var total int64
	if err := s.db.DB.WithContext(ctx).Model(&sessionRow{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("session.gormx: count: %w", err)
	}
	offset := (page - 1) * pageSize
	var rows []sessionRow
	if err := s.db.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("session.gormx: list: %w", err)
	}
	out := make([]*Session, 0, len(rows))
	for i := range rows {
		sess, err := rowToSession(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, sess)
	}
	return out, total, nil
}

func (s *GormxStore) DeleteSession(ctx context.Context, userID, sessionID string) error {
	return s.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ? AND user_id = ?", sessionID, userID).Delete(&interruptRow{}).Error; err != nil {
			return fmt.Errorf("session.gormx: delete interrupts: %w", err)
		}
		res := tx.Where("id = ? AND user_id = ?", sessionID, userID).Delete(&sessionRow{})
		if res.Error != nil {
			return fmt.Errorf("session.gormx: delete session: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("session: not found")
		}
		return nil
	})
}

func (s *GormxStore) SaveInterrupt(ctx context.Context, r *InterruptRecord) error {
	if r == nil || r.InterruptID == "" {
		return fmt.Errorf("session.gormx: empty interrupt")
	}
	dataJSON, err := marshalInterruptData(r.Data)
	if err != nil {
		return err
	}
	row := interruptRow{
		InterruptID: r.InterruptID,
		SessionID:   r.SessionID,
		UserID:      r.UserID,
		Kind:        int32(r.Kind),
		ToolName:    r.ToolName,
		Question:    r.Question,
		DataJSON:    dataJSON,
		CreatedAt:   r.CreatedAt,
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	return s.db.DB.WithContext(ctx).Save(&row).Error
}

func (s *GormxStore) GetInterrupt(ctx context.Context, id string) (*InterruptRecord, error) {
	var row interruptRow
	err := s.db.DB.WithContext(ctx).Where("interrupt_id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("session: interrupt %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("session.gormx: get interrupt: %w", err)
	}
	return rowToInterrupt(&row)
}

// RecoverRunningSessions 仅恢复租约已过期、或「无租约且 updated_at 过旧」的 RUNNING 会话。
func (s *GormxStore) RecoverRunningSessions(ctx context.Context) (int, error) {
	now := time.Now()
	nullCutoff := now.Add(-s.nullLeaseRecoverGrace)
	stRunning := int32(aisolo.SessionStatus_SESSION_STATUS_RUNNING)
	stIdle := int32(aisolo.SessionStatus_SESSION_STATUS_IDLE)

	res := s.db.DB.WithContext(ctx).Exec(`
UPDATE einox_aisolo_session
SET status = ?, run_owner = '', run_lease_until = NULL, updated_at = ?
WHERE status = ? AND (
  (run_lease_until IS NOT NULL AND run_lease_until < ?)
  OR (run_lease_until IS NULL AND updated_at < ?)
)`, stIdle, now, stRunning, now, nullCutoff)
	if res.Error != nil {
		return 0, fmt.Errorf("session.gormx: recover running: %w", res.Error)
	}
	return int(res.RowsAffected), nil
}

func (s *GormxStore) Close() error { return nil }

// --- DB models ---

type sessionRow struct {
	ID            string     `gorm:"column:id;type:varchar(64);primaryKey"`
	UserID        string     `gorm:"column:user_id;type:varchar(64);index:idx_sess_user_upd,priority:1"`
	Title         string     `gorm:"type:varchar(512)"`
	Mode          int32      `gorm:"column:mode"`
	Status        int32      `gorm:"column:status"`
	InterruptID   string     `gorm:"column:interrupt_id;type:varchar(128)"`
	MessageCount  int32      `gorm:"column:message_count"`
	LastMessage   string     `gorm:"column:last_message;type:text"`
	UILang        string     `gorm:"column:ui_lang;type:varchar(16)"`
	RunOwner      string     `gorm:"column:run_owner;type:varchar(128)"`
	RunLeaseUntil *time.Time `gorm:"column:run_lease_until"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;index:idx_sess_user_upd,priority:2"`
}

func (sessionRow) TableName() string { return "einox_aisolo_session" }

type interruptRow struct {
	InterruptID string    `gorm:"primaryKey;column:interrupt_id;type:varchar(128)"`
	SessionID   string    `gorm:"column:session_id;type:varchar(64);index:idx_ir_sess"`
	UserID      string    `gorm:"column:user_id;type:varchar(64)"`
	Kind        int32     `gorm:"column:kind"`
	ToolName    string    `gorm:"column:tool_name;type:varchar(256)"`
	Question    string    `gorm:"column:question;type:text"`
	DataJSON    []byte    `gorm:"column:data_json;type:longblob"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (interruptRow) TableName() string { return "einox_aisolo_interrupt" }

func sessionToRow(s *Session) (*sessionRow, error) {
	var lease *time.Time
	if !s.RunLeaseUntil.IsZero() {
		t := s.RunLeaseUntil
		lease = &t
	}
	return &sessionRow{
		ID:            s.ID,
		UserID:        s.UserID,
		Title:         s.Title,
		Mode:          int32(s.Mode),
		Status:        int32(s.Status),
		InterruptID:   s.InterruptID,
		MessageCount:  s.MessageCount,
		LastMessage:   s.LastMessage,
		UILang:        s.UILang,
		RunOwner:      s.RunOwner,
		RunLeaseUntil: lease,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}, nil
}

func rowToSession(r *sessionRow) (*Session, error) {
	s := &Session{
		ID:           r.ID,
		UserID:       r.UserID,
		Title:        r.Title,
		Mode:         aisolo.AgentMode(r.Mode),
		Status:       aisolo.SessionStatus(r.Status),
		InterruptID:  r.InterruptID,
		MessageCount: r.MessageCount,
		LastMessage:  r.LastMessage,
		UILang:       r.UILang,
		RunOwner:     r.RunOwner,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	if r.RunLeaseUntil != nil {
		s.RunLeaseUntil = *r.RunLeaseUntil
	}
	return s, nil
}

func marshalInterruptData(d *protocol.InterruptData) ([]byte, error) {
	if d == nil {
		return nil, nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("session.gormx: marshal interrupt data: %w", err)
	}
	return b, nil
}

func rowToInterrupt(r *interruptRow) (*InterruptRecord, error) {
	rec := &InterruptRecord{
		InterruptID: r.InterruptID,
		SessionID:   r.SessionID,
		UserID:      r.UserID,
		Kind:        aisolo.InterruptKind(r.Kind),
		ToolName:    r.ToolName,
		Question:    r.Question,
		CreatedAt:   r.CreatedAt,
	}
	if len(r.DataJSON) > 0 {
		var d protocol.InterruptData
		if err := json.Unmarshal(r.DataJSON, &d); err != nil {
			return nil, fmt.Errorf("session.gormx: unmarshal interrupt data: %w", err)
		}
		rec.Data = &d
	}
	return rec, nil
}

var _ Store = (*GormxStore)(nil)
