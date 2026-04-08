package memory

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// 消息类型
// =============================================================================

// Message 对话消息
type Message struct {
	Role    string `json:"role"`    // user, assistant, system, tool
	Content string `json:"content"` // 消息内容
}

// Session 会话
type Session struct {
	ID       string    `json:"id"`       // 会话 ID
	Messages []Message `json:"messages"` // 消息历史
	System   string    `json:"system"`   // 系统提示
}

// =============================================================================
// Memory 接口与 Storage 适配器
// =============================================================================

// Memory 记忆接口
type Memory interface {
	Save(ctx context.Context, sessionID string, msg Message) error
	Get(ctx context.Context, sessionID string, limit int) ([]Message, error)
	Clear(ctx context.Context, sessionID string) error
}

// memoryAdapter Memory 接口到 Storage 的适配器
type memoryAdapter struct {
	storage *MemoryStorage
}

// NewMemory 创建 Memory 实例
func NewMemory(opts ...StorageOption) Memory {
	return &memoryAdapter{storage: NewMemoryStorage(opts...)}
}

func (a *memoryAdapter) Save(ctx context.Context, sessionID string, msg Message) error {
	return a.storage.Save(ctx, sessionID, ToSchemaMessage(msg))
}

func (a *memoryAdapter) Get(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	msgs, err := a.storage.Get(ctx, sessionID, limit)
	if err != nil {
		return nil, err
	}
	return FromSchemaMessages(msgs), nil
}

func (a *memoryAdapter) Clear(ctx context.Context, sessionID string) error {
	return a.storage.Clear(ctx, sessionID)
}

// =============================================================================
// 工具函数
// =============================================================================

// ToSchemaMessage 转换 Message 到 schema.Message
func ToSchemaMessage(msg Message) *schema.Message {
	return &schema.Message{
		Role:    schema.RoleType(msg.Role),
		Content: msg.Content,
	}
}

// ToSchemaMessages 批量转换
func ToSchemaMessages(session *Session) []*schema.Message {
	messages := make([]*schema.Message, 0, len(session.Messages))
	if session.System != "" {
		messages = append(messages, &schema.Message{
			Role:    schema.System,
			Content: session.System,
		})
	}
	for _, msg := range session.Messages {
		messages = append(messages, ToSchemaMessage(msg))
	}
	return messages
}

// FromSchemaMessage 转换 schema.Message 到 Message
func FromSchemaMessage(msg *schema.Message) Message {
	return Message{
		Role:    string(msg.Role),
		Content: msg.Content,
	}
}

// FromSchemaMessages 批量转换 schema.Message 到 Message
func FromSchemaMessages(msgs []*schema.Message) []Message {
	result := make([]Message, len(msgs))
	for i, msg := range msgs {
		result[i] = FromSchemaMessage(msg)
	}
	return result
}
