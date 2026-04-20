package memory

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// ConversationMessage 对话消息结构。
//
// 与模型无关，包含多轮对话上下文所需的全部字段。
type ConversationMessage struct {
	ID        string                    `json:"id,omitempty"`    // 消息 ID（存储层生成）
	SessionID string                    `json:"sessionId"`       // 会话 ID
	UserID    string                    `json:"userId"`          // 用户 ID
	Role      string                    `json:"role"`            // 角色：user/assistant/system/tool
	Content   string                    `json:"content"`         // 消息内容
	Parts     []schema.MessageInputPart `json:"parts,omitempty"` // 多部分内容

	ToolCalls  []schema.ToolCall `json:"toolCalls,omitempty"`
	ToolCallID string            `json:"toolCallId,omitempty"`
	ToolName   string            `json:"toolName,omitempty"`

	ReasoningContent string    `json:"reasoningContent,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

// ToSchemaMessage 将 ConversationMessage 转换为 eino schema.Message。
func (m *ConversationMessage) ToSchemaMessage() *schema.Message {
	msg := &schema.Message{
		Role:             schema.RoleType(m.Role),
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		ToolCalls:        m.ToolCalls,
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
	}
	if len(m.Parts) > 0 {
		msg.UserInputMultiContent = m.Parts
	}
	return msg
}

// FromSchemaMessage 从 eino schema.Message 创建 ConversationMessage。
func FromSchemaMessage(userID, sessionID string, msg *schema.Message) *ConversationMessage {
	return &ConversationMessage{
		SessionID:        sessionID,
		UserID:           userID,
		Role:             string(msg.Role),
		Content:          msg.Content,
		Parts:            msg.UserInputMultiContent,
		ToolCalls:        msg.ToolCalls,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		ReasoningContent: msg.ReasoningContent,
		CreatedAt:        time.Now(),
	}
}

// MessagesToText 将消息列表转换为纯文本，方便调试。
func MessagesToText(msgs []*ConversationMessage) string {
	var text string
	for _, msg := range msgs {
		role := msg.Role
		if role == "" {
			role = "user"
		}
		text += role + ": " + msg.Content + "\n\n"
	}
	return text
}
