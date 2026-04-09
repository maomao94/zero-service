package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// SQLStorage GORM SQL 存储实现
// =============================================================================

// SQLStorage 基于 GORM 的 SQL 存储实现
// 适用于生产环境，支持多实例部署
type SQLStorage struct {
	db *gorm.DB
}

// NewSQLStorage 创建 SQL 存储
func NewSQLStorage(db *gorm.DB) *SQLStorage {
	return &SQLStorage{db: db}
}

// =============================================================================
// 数据模型
// =============================================================================

// ConversationMessageModel 数据库模型
type ConversationMessageModel struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SessionID string    `gorm:"index;size:64;not null" json:"sessionId"`
	UserID    string    `gorm:"index;size:64;not null" json:"userId"`
	Role      string    `gorm:"size:20;not null" json:"role"`
	Content   string    `gorm:"type:text" json:"content"`
	Parts     string    `gorm:"type:text" json:"parts,omitempty"` // JSON 序列化
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

// TableName 表名
func (ConversationMessageModel) TableName() string {
	return "eino_conversation_messages"
}

// UserMemoryModel 用户记忆数据库模型
type UserMemoryModel struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	UserID    string    `gorm:"uniqueIndex;size:64;not null" json:"userId"`
	Memory    string    `gorm:"type:text" json:"memory"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName 表名
func (UserMemoryModel) TableName() string {
	return "eino_user_memories"
}

// SessionSummaryModel 会话摘要数据库模型
type SessionSummaryModel struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SessionID string    `gorm:"index;size:64;not null" json:"sessionId"`
	UserID    string    `gorm:"index;size:64;not null" json:"userId"`
	Summary   string    `gorm:"type:text" json:"summary"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName 表名
func (SessionSummaryModel) TableName() string {
	return "eino_session_summaries"
}

// =============================================================================
// Storage 接口实现
// =============================================================================

// SaveMessage 保存消息
func (s *SQLStorage) SaveMessage(ctx context.Context, msg *ConversationMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	model := &ConversationMessageModel{
		ID:        msg.ID,
		SessionID: msg.SessionID,
		UserID:    msg.UserID,
		Role:      msg.Role,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}

	return s.db.WithContext(ctx).Create(model).Error
}

// GetMessages 获取会话消息
func (s *SQLStorage) GetMessages(ctx context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error) {
	var models []ConversationMessageModel
	query := s.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	messages := make([]*ConversationMessage, len(models))
	for i, m := range models {
		messages[i] = &ConversationMessage{
			ID:        m.ID,
			SessionID: m.SessionID,
			UserID:    m.UserID,
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
		}
	}
	return messages, nil
}

// GetMessageCount 获取会话消息数量
func (s *SQLStorage) GetMessageCount(ctx context.Context, userID, sessionID string) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&ConversationMessageModel{}).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Count(&count).Error
	return int(count), err
}

// CleanupMessagesByLimit 清理超限消息（保留最新的 N 条）
func (s *SQLStorage) CleanupMessagesByLimit(ctx context.Context, userID, sessionID string, keepCount int) error {
	// 先获取要保留的消息 ID
	var lastMessages []ConversationMessageModel
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("created_at DESC").
		Limit(keepCount).
		Find(&lastMessages).Error
	if err != nil {
		return err
	}

	if len(lastMessages) == 0 {
		return nil
	}

	// 保留消息的 ID
	keepIDs := make([]string, len(lastMessages))
	for i, m := range lastMessages {
		keepIDs[i] = m.ID
	}

	// 删除不在保留列表中的消息
	return s.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ? AND id NOT IN ?", userID, sessionID, keepIDs).
		Delete(&ConversationMessageModel{}).Error
}

// CleanupMessagesByTime 清理过期消息
func (s *SQLStorage) CleanupMessagesByTime(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return s.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&ConversationMessageModel{}).Error
}

// GetUserMemory 获取用户记忆
func (s *SQLStorage) GetUserMemory(ctx context.Context, userID string) (*UserMemory, error) {
	var model UserMemoryModel
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &UserMemory{
		ID:        model.ID,
		UserID:    model.UserID,
		Memory:    model.Memory,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}, nil
}

// SaveUserMemory 保存用户记忆
func (s *SQLStorage) SaveUserMemory(ctx context.Context, memory *UserMemory) error {
	now := time.Now()
	if memory.ID == "" {
		memory.ID = uuid.NewString()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = now
	}
	memory.UpdatedAt = now

	model := &UserMemoryModel{
		ID:        memory.ID,
		UserID:    memory.UserID,
		Memory:    memory.Memory,
		CreatedAt: memory.CreatedAt,
		UpdatedAt: memory.UpdatedAt,
	}

	// 使用 upsert
	return s.db.WithContext(ctx).
		Where("user_id = ?", memory.UserID).
		Assign(model).
		FirstOrCreate(model).Error
}

// GetSessionSummary 获取会话摘要
func (s *SQLStorage) GetSessionSummary(ctx context.Context, userID, sessionID string) (*SessionSummary, error) {
	var model SessionSummaryModel
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &SessionSummary{
		ID:        model.ID,
		SessionID: model.SessionID,
		UserID:    model.UserID,
		Summary:   model.Summary,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}, nil
}

// SaveSessionSummary 保存会话摘要
func (s *SQLStorage) SaveSessionSummary(ctx context.Context, summary *SessionSummary) error {
	now := time.Now()
	if summary.ID == "" {
		summary.ID = uuid.NewString()
	}
	if summary.CreatedAt.IsZero() {
		summary.CreatedAt = now
	}
	summary.UpdatedAt = now

	model := &SessionSummaryModel{
		ID:        summary.ID,
		SessionID: summary.SessionID,
		UserID:    summary.UserID,
		Summary:   summary.Summary,
		CreatedAt: summary.CreatedAt,
		UpdatedAt: summary.UpdatedAt,
	}

	// 使用 upsert
	return s.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", summary.UserID, summary.SessionID).
		Assign(model).
		FirstOrCreate(model).Error
}

// CleanupOldSessions 清理旧会话
func (s *SQLStorage) CleanupOldSessions(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// 查找需要清理的会话
	var sessions []struct {
		UserID    string
		SessionID string
	}
	err := s.db.WithContext(ctx).
		Model(&ConversationMessageModel{}).
		Select("DISTINCT user_id, session_id").
		Where("created_at < ?", cutoff).
		Find(&sessions).Error
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return nil
	}

	// 删除这些会话的消息和摘要
	for _, session := range sessions {
		// 删除消息
		if err := s.db.WithContext(ctx).
			Where("user_id = ? AND session_id = ?", session.UserID, session.SessionID).
			Delete(&ConversationMessageModel{}).Error; err != nil {
			return err
		}
		// 删除摘要
		if err := s.db.WithContext(ctx).
			Where("user_id = ? AND session_id = ?", session.UserID, session.SessionID).
			Delete(&SessionSummaryModel{}).Error; err != nil {
			return err
		}
	}

	return nil
}

// AutoMigrate 自动迁移
func (s *SQLStorage) AutoMigrate() error {
	return s.db.AutoMigrate(
		&ConversationMessageModel{},
		&UserMemoryModel{},
		&SessionSummaryModel{},
	)
}
