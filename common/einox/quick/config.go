package quick

import (
	"context"

	einomodel "zero-service/common/einox/model"

	"github.com/cloudwego/eino/components/model"
)

// Config 快速配置
type Config struct {
	// 模型配置
	Provider    string  `json:"provider"`    // 提供商: openai, deepseek, ollama, qwen, ark
	APIKey      string  `json:"api_key"`     // API Key
	BaseURL     string  `json:"base_url"`    // 自定义 Base URL（可选）
	Model       string  `json:"model"`       // 模型名称
	Temperature float64 `json:"temperature"` // 温度参数
	MaxTokens   int     `json:"max_tokens"`  // 最大 Token

	// Ollama 专用
	OllamaURL string `json:"ollama_url"` // Ollama 服务地址

	// ARK 专用
	ArkRegion string `json:"ark_region"` // Region: cn-beijing, cn-shanghai

	// Agent 配置
	SystemPrompt string `json:"system_prompt"` // 系统提示词
	Name         string `json:"name"`          // Agent 名称
	Description  string `json:"description"`   // Agent 描述
}

// NewChatModel 创建 ChatModel
func NewChatModel(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	mCfg := einomodel.Config{
		Provider:    einomodel.Provider(cfg.Provider),
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
		OllamaURL:   cfg.OllamaURL,
		ArkRegion:   cfg.ArkRegion,
	}
	return einomodel.NewChatModel(ctx, mCfg)
}
