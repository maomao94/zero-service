package model

import (
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// =============================================================================
// ChatModel 选项（Option 模式，推荐）
// =============================================================================

// ChatModelOption ChatModel 配置选项
type ChatModelOption func(*chatModelOptions)

type chatModelOptions struct {
	apiKey      string
	baseURL     string
	model       string
	temperature float32
	maxTokens   int
	timeout     time.Duration
	arkRegion   string // ARK Region: cn-beijing, cn-shanghai
}

// WithAPIKey 设置 API Key
func WithAPIKey(apiKey string) ChatModelOption {
	return func(o *chatModelOptions) {
		o.apiKey = apiKey
	}
}

// WithBaseURL 设置 Base URL
func WithBaseURL(baseURL string) ChatModelOption {
	return func(o *chatModelOptions) {
		o.baseURL = baseURL
	}
}

// WithModel 设置模型名称
func WithModel(model string) ChatModelOption {
	return func(o *chatModelOptions) {
		o.model = model
	}
}

// WithTemperature 设置温度
func WithTemperature(temp float32) ChatModelOption {
	return func(o *chatModelOptions) {
		o.temperature = temp
	}
}

// WithMaxTokens 设置最大 Token 数
func WithMaxTokens(maxTokens int) ChatModelOption {
	return func(o *chatModelOptions) {
		o.maxTokens = maxTokens
	}
}

// WithTimeout 设置超时
func WithTimeout(timeout time.Duration) ChatModelOption {
	return func(o *chatModelOptions) {
		o.timeout = timeout
	}
}

// WithArkRegion 设置 ARK Region
//
//	Region 可选值：
//	- cn-beijing: 北京 Region（默认）
//	- cn-shanghai: 上海 Region
func WithArkRegion(region string) ChatModelOption {
	return func(o *chatModelOptions) {
		o.arkRegion = region
	}
}

// =============================================================================
// NewChatModelByOption 工厂函数（Option 模式）
// =============================================================================

// NewChatModelByOption 创建 ChatModel（Option 模式，推荐）
//
// 返回 ToolCallingChatModel，支持并发安全的 WithTools 工具绑定。
func NewChatModelByOption(provider Provider, opts ...ChatModelOption) (model.BaseChatModel, error) {
	var cfg chatModelOptions
	for _, opt := range opts {
		opt(&cfg)
	}
	return newChatModelByOption(context.Background(), provider, &cfg)
}

func newChatModelByOption(ctx context.Context, provider Provider, cfg *chatModelOptions) (model.BaseChatModel, error) {
	switch provider {
	case ProviderOpenAI:
		return newOpenAIByOption(ctx, cfg)
	case ProviderDeepSeek:
		return newDeepSeekByOption(ctx, cfg)
	case ProviderOllama:
		return newOllamaByOption(ctx, cfg)
	case ProviderQwen:
		return newQwenByOption(ctx, cfg)
	case ProviderArk:
		return newArkByOption(ctx, cfg)
	case ProviderClaude:
		return newClaudeByOption(ctx, cfg)
	default:
		logx.Errorf("unsupported provider: %s", provider)
		return nil, nil
	}
}

func newOpenAIByOption(ctx context.Context, cfg *chatModelOptions) (model.BaseChatModel, error) {
	config := &openai.ChatModelConfig{
		APIKey: cfg.apiKey,
		Model:  cfg.model,
	}
	if cfg.baseURL != "" {
		config.BaseURL = cfg.baseURL
	}
	if cfg.temperature > 0 {
		config.Temperature = &cfg.temperature
	}
	if cfg.maxTokens > 0 {
		config.MaxTokens = &cfg.maxTokens
	}
	return openai.NewChatModel(ctx, config)
}

func newDeepSeekByOption(ctx context.Context, cfg *chatModelOptions) (model.BaseChatModel, error) {
	config := &deepseek.ChatModelConfig{
		APIKey: cfg.apiKey,
		Model:  cfg.model,
	}
	if cfg.baseURL != "" {
		config.BaseURL = cfg.baseURL
	}
	if cfg.temperature > 0 {
		config.Temperature = cfg.temperature
	}
	if cfg.maxTokens > 0 {
		config.MaxTokens = cfg.maxTokens
	}
	return deepseek.NewChatModel(ctx, config)
}

func newOllamaByOption(ctx context.Context, cfg *chatModelOptions) (model.ChatModel, error) {
	url := cfg.baseURL
	if url == "" {
		url = "http://localhost:11434"
	}
	config := &ollama.ChatModelConfig{
		BaseURL: url,
		Model:   cfg.model,
	}
	return ollama.NewChatModel(ctx, config)
}

func newQwenByOption(ctx context.Context, cfg *chatModelOptions) (model.BaseChatModel, error) {
	config := &qwen.ChatModelConfig{
		APIKey: cfg.apiKey,
		Model:  cfg.model,
	}
	if cfg.baseURL != "" {
		config.BaseURL = cfg.baseURL
	}
	if cfg.temperature > 0 {
		config.Temperature = &cfg.temperature
	}
	if cfg.maxTokens > 0 {
		config.MaxTokens = &cfg.maxTokens
	}
	return qwen.NewChatModel(ctx, config)
}

// newArkByOption 火山引擎 ARK（Option 模式）
func newArkByOption(ctx context.Context, cfg *chatModelOptions) (model.BaseChatModel, error) {
	config := &ark.ChatModelConfig{
		APIKey: cfg.apiKey,
		Model:  cfg.model,
	}

	// 设置 Region
	if cfg.arkRegion != "" {
		config.Region = cfg.arkRegion
	}

	// 设置自定义 BaseURL
	if cfg.baseURL != "" {
		config.BaseURL = cfg.baseURL
	}

	// 设置温度
	if cfg.temperature > 0 {
		config.Temperature = &cfg.temperature
	}

	// 设置最大 Token
	if cfg.maxTokens > 0 {
		config.MaxTokens = &cfg.maxTokens
	}

	return ark.NewChatModel(ctx, config)
}

func newClaudeByOption(ctx context.Context, cfg *chatModelOptions) (model.BaseChatModel, error) {
	logx.Errorf("claude provider not implemented yet, use openai with claude base url")
	return nil, nil
}
