package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"zero-service/common/gormx"
)

// GormxStorage 基于 common/gormx 的关系型数据库存储实现。
type GormxStorage struct {
	db *gormx.DB
}

// NewGormxStorage 创建 gormx 存储。
// 调用方需传入已初始化好的 *gormx.DB。
func NewGormxStorage(db *gormx.DB) (*GormxStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("memory.gormx: db is nil")
	}
	s := &GormxStorage{db: db}
	if err := s.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("memory.gormx: auto migrate: %w", err)
	}
	return s, nil
}

// AutoMigrate 创建表结构。
func (s *GormxStorage) AutoMigrate() error {
	return s.db.DB.AutoMigrate(&MessageModel{})
}

// SaveMessage 保存一条消息。
func (s *GormxStorage) SaveMessage(ctx context.Context, msg *ConversationMessage) error {
	if msg == nil {
		return nil
	}
	m, err := newMessageModel(msg)
	if err != nil {
		return fmt.Errorf("memory.gormx: build model: %w", err)
	}
	return s.db.DB.WithContext(ctx).Create(m).Error
}

// GetMessages 获取会话消息（按创建时间升序）。
func (s *GormxStorage) GetMessages(ctx context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error) {
	var rows []MessageModel
	q := s.db.DB.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("created_at ASC")
	if limit > 0 {
		// 取最新的 limit 条：按 created_at DESC 取 limit 条后再反转
		q = s.db.DB.WithContext(ctx).
			Where("user_id = ? AND session_id = ?", userID, sessionID).
			Order("created_at DESC").
			Limit(limit)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("memory.gormx: query: %w", err)
	}

	msgs := make([]*ConversationMessage, 0, len(rows))
	for i := range rows {
		msg, err := rows[i].toConversationMessage()
		if err != nil {
			return nil, fmt.Errorf("memory.gormx: decode message %s: %w", rows[i].ID, err)
		}
		msgs = append(msgs, msg)
	}

	if limit > 0 {
		// 由于上面 DESC 取的，反转为升序
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
	}
	return msgs, nil
}

// DeleteSession 删除会话全部消息。
func (s *GormxStorage) DeleteSession(ctx context.Context, userID, sessionID string) error {
	return s.db.DB.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Delete(&MessageModel{}).Error
}

// Close 关闭存储。
func (s *GormxStorage) Close() error {
	// 连接生命周期由调用方持有的 *gormx.DB 管理，这里不主动关闭。
	return nil
}

// =============================================================================
// 表模型
// =============================================================================

// MessageModel 对话消息持久化结构。
//
// 为避免 MySQL 早期版本对 utf8mb4 索引长度的限制，字符串主键使用 varchar(64)。
type MessageModel struct {
	ID        string `gorm:"column:id;type:varchar(64);primaryKey"`
	SessionID string `gorm:"column:session_id;type:varchar(64);index:idx_session,priority:2"`
	UserID    string `gorm:"column:user_id;type:varchar(64);index:idx_session,priority:1"`
	Role      string `gorm:"column:role;type:varchar(32)"`
	Content   string `gorm:"column:content;type:longtext"`
	PartsJSON []byte `gorm:"column:parts_json;type:mediumtext"`
	ToolsJSON []byte `gorm:"column:tools_json;type:mediumtext"`

	ToolCallID       string    `gorm:"column:tool_call_id;type:varchar(128)"`
	ToolName         string    `gorm:"column:tool_name;type:varchar(128)"`
	ReasoningContent string    `gorm:"column:reasoning_content;type:longtext"`
	CreatedAt        time.Time `gorm:"column:created_at;index:idx_session,priority:3"`
}

// TableName 自定义表名。
func (MessageModel) TableName() string {
	return "einox_conversation_message"
}

func newMessageModel(msg *ConversationMessage) (*MessageModel, error) {
	id := msg.ID
	if id == "" {
		id = uuid.NewString()
	}
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	var partsJSON []byte
	if len(msg.Parts) > 0 {
		data, err := json.Marshal(msg.Parts)
		if err != nil {
			return nil, err
		}
		partsJSON = data
	}
	var toolsJSON []byte
	if len(msg.ToolCalls) > 0 {
		data, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			return nil, err
		}
		toolsJSON = data
	}

	return &MessageModel{
		ID:               id,
		SessionID:        msg.SessionID,
		UserID:           msg.UserID,
		Role:             msg.Role,
		Content:          msg.Content,
		PartsJSON:        partsJSON,
		ToolsJSON:        toolsJSON,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		ReasoningContent: msg.ReasoningContent,
		CreatedAt:        createdAt,
	}, nil
}

func (m *MessageModel) toConversationMessage() (*ConversationMessage, error) {
	msg := &ConversationMessage{
		ID:               m.ID,
		SessionID:        m.SessionID,
		UserID:           m.UserID,
		Role:             m.Role,
		Content:          m.Content,
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
		ReasoningContent: m.ReasoningContent,
		CreatedAt:        m.CreatedAt,
	}
	if len(m.PartsJSON) > 0 {
		if err := json.Unmarshal(m.PartsJSON, &msg.Parts); err != nil {
			return nil, err
		}
	}
	if len(m.ToolsJSON) > 0 {
		if err := json.Unmarshal(m.ToolsJSON, &msg.ToolCalls); err != nil {
			return nil, err
		}
	}
	return msg, nil
}
