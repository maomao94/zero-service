package model

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ChatModelFactory ChatModel 工厂函数类型
type ChatModelFactory func() (model.BaseChatModel, error)

// NewChatModelByConfig 根据配置创建 ChatModel
func NewChatModelByConfig(cfg Config) (model.BaseChatModel, error) {
	return NewChatModel(context.Background(), cfg)
}

// WithSystemPrompt 返回带有系统提示的 Messages
func WithSystemPrompt(system string, messages ...*schema.Message) []*schema.Message {
	if system == "" {
		return messages
	}
	return append([]*schema.Message{
		{Role: schema.System, Content: system},
	}, messages...)
}

// UserMessage 创建用户消息
func UserMessage(content string) *schema.Message {
	return &schema.Message{
		Role:    schema.User,
		Content: content,
	}
}

// AssistantMessage 创建助手消息
func AssistantMessage(content string) *schema.Message {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: content,
	}
}

// ToolMessage 创建工具消息
func ToolMessage(content, toolCallID string) *schema.Message {
	return &schema.Message{
		Role:       schema.Tool,
		Content:    content,
		ToolCallID: toolCallID,
	}
}

// BuildMessages 构建对话消息列表
//
// 示例：
//
//	messages := model.BuildMessages(
//	    model.UserMessage("Hello"),
//	    model.AssistantMessage("Hi!"),
//	    model.UserMessage("How are you?"),
//	)
func BuildMessages(msgs ...*schema.Message) []*schema.Message {
	return msgs
}
