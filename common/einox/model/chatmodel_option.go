package model

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/model"
)

// =============================================================================
// ChatModel 选项（Option 模式，推荐）
// =============================================================================

// ChatModelOption ChatModel 配置选项
type ChatModelOption func(*chatModelOptions)

type chatModelOptions struct {
	apiKey         string
	baseURL        string
	model          string
	temperature    float32
	temperatureSet bool
	maxTokens      int
	timeout        time.Duration
	arkRegion      string // ARK Region: cn-beijing, cn-shanghai
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
		o.temperatureSet = true
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
	return NewChatModel(ctx, cfg.toConfig(provider))
}

func (c chatModelOptions) toConfig(provider Provider) Config {
	return Config{
		Provider:       provider,
		APIKey:         c.apiKey,
		BaseURL:        c.baseURL,
		Model:          c.model,
		Temperature:    float64(c.temperature),
		TemperatureSet: c.temperatureSet,
		MaxTokens:      c.maxTokens,
		OllamaURL:      c.baseURL,
		ArkRegion:      c.arkRegion,
	}
}
