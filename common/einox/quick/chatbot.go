package quick

import (
	"context"
	"fmt"

	"zero-service/common/einox"
	"zero-service/common/einox/agent"
)

// ChatBot 简单的聊天机器人
type ChatBot struct {
	agent *agent.Agent
}

// NewChatBot 创建聊天机器人
//
//	bot := quick.NewChatBot(ctx, &quick.Config{
//	    Provider: "ark",
//	    APIKey:   "your-api-key",
//	    Model:    "deepseek-v3-2-251201",
//	})
func NewChatBot(ctx context.Context, cfg *Config) (*ChatBot, error) {
	// 1. 创建模型
	chatModel, err := NewChatModel(ctx, *cfg)
	if err != nil {
		return nil, fmt.Errorf("create chat model: %w", err)
	}

	// 2. 创建 Agent
	opts := []agent.Option{
		agent.WithName(cfg.Name),
		agent.WithDescription(cfg.Description),
	}

	if cfg.SystemPrompt != "" {
		opts = append(opts, agent.WithInstruction(cfg.SystemPrompt))
	}

	einoAgent, err := agent.NewChatModelAgent(ctx, chatModel, opts...)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return &ChatBot{agent: einoAgent}, nil
}

// Chat 简单对话
//
//	resp, err := bot.Chat(ctx, "你好")
func (b *ChatBot) Chat(ctx context.Context, message string) (string, error) {
	if b.agent == nil {
		return "", fmt.Errorf("agent not initialized")
	}

	result, err := b.agent.Run(ctx, message)
	if err != nil {
		return "", err
	}

	if result.Err != nil {
		return "", result.Err
	}

	return result.Response, nil
}

// ChatWithHistory 带历史的对话
func (b *ChatBot) ChatWithHistory(ctx context.Context, message string, history []*Message) (string, error) {
	if b.agent == nil {
		return "", fmt.Errorf("agent not initialized")
	}

	result, err := b.agent.RunWithHistory(ctx, message,
		einox.WithUserID("default"),
		einox.WithSessionID("default"),
	)
	if err != nil {
		return "", err
	}

	if result.Err != nil {
		return "", result.Err
	}

	return result.Response, nil
}

// Message 对话消息
type Message struct {
	Role    string // user, assistant, system
	Content string
}

// NewChatBotWithTools 创建带工具的聊天机器人
func NewChatBotWithTools(ctx context.Context, cfg *Config, tools ...any) (*ChatBot, error) {
	return NewChatBot(ctx, cfg)
}
