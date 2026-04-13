package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// Provider LLM 提供商
type Provider string

const (
	ProviderOpenAI   Provider = "openai"
	ProviderDeepSeek Provider = "deepseek" // DeepSeek 官方 API
	ProviderOllama   Provider = "ollama"
	ProviderQwen     Provider = "qwen"
	ProviderClaude   Provider = "claude"
	ProviderArk      Provider = "ark" // 火山引擎 ARK（支持 DeepSeek、豆包等）
)

// ARK 默认端点
const (
	// ArkBaseURLCN 北京 Region
	ArkBaseURLCN = "https://ark.cn-beijing.volces.com/api/v3"
	// ArkBaseURLSH 上海 Region
	ArkBaseURLSH = "https://ark.cn-shanghai.volces.com/api/v3"
)

// Config ChatModel 配置（兼容旧代码）
type Config struct {
	Provider Provider `json:"provider"` // 提供商：openai, deepseek, ollama, qwen, ark

	// OpenAI / DeepSeek / Qwen / ARK
	APIKey      string  `json:"api_key"`     // API Key
	BaseURL     string  `json:"base_url"`    // API Base URL（可选）
	Model       string  `json:"model"`       // 模型名称
	Temperature float64 `json:"temperature"` // 温度参数
	MaxTokens   int     `json:"max_tokens"`  // 最大 token 数

	// Ollama
	OllamaURL string `json:"ollama_url"` // Ollama 服务地址，默认 http://localhost:11434

	// ARK
	ArkRegion string `json:"ark_region"` // ARK Region: cn-beijing (默认), cn-shanghai
}

// NewChatModel 创建 ChatModel 实例
//
// 返回 BaseChatModel，支持并发安全的 WithTools 工具绑定。
//
// 支持的提供商：
// - openai: OpenAI GPT 系列
// - deepseek: DeepSeek 系列（官方 API）
// - ollama: 本地 Ollama 模型
// - qwen: 阿里 Qwen 系列
// - ark: 火山引擎 ARK（支持 DeepSeek、豆包等）
func NewChatModel(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	switch cfg.Provider {
	case ProviderOpenAI:
		return newOpenAI(ctx, cfg)
	case ProviderDeepSeek:
		return newDeepSeek(ctx, cfg)
	case ProviderOllama:
		return newOllama(ctx, cfg)
	case ProviderQwen:
		return newQwen(ctx, cfg)
	case ProviderArk:
		return newArk(ctx, cfg)
	case ProviderClaude:
		return newClaude(ctx, cfg)
	default:
		logx.Errorf("unsupported provider: %s", cfg.Provider)
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

func newOpenAI(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	config := &openai.ChatModelConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	}
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}
	if cfg.Temperature > 0 {
		t := float32(cfg.Temperature)
		config.Temperature = &t
	}
	if cfg.MaxTokens > 0 {
		config.MaxTokens = &cfg.MaxTokens
	}

	return openai.NewChatModel(ctx, config)
}

func newDeepSeek(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	config := &deepseek.ChatModelConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	}
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}
	if cfg.Temperature > 0 {
		config.Temperature = float32(cfg.Temperature)
	}
	if cfg.MaxTokens > 0 {
		config.MaxTokens = cfg.MaxTokens
	}

	return deepseek.NewChatModel(ctx, config)
}

func newOllama(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	url := cfg.OllamaURL
	if url == "" {
		url = "http://localhost:11434"
	}

	config := &ollama.ChatModelConfig{
		BaseURL: url,
		Model:   cfg.Model,
	}

	return ollama.NewChatModel(ctx, config)
}

func newQwen(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	config := &qwen.ChatModelConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	}
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}
	if cfg.Temperature > 0 {
		t := float32(cfg.Temperature)
		config.Temperature = &t
	}
	if cfg.MaxTokens > 0 {
		config.MaxTokens = &cfg.MaxTokens
	}

	return qwen.NewChatModel(ctx, config)
}

// newArk 火山引擎 ARK
//
// ARK 是火山引擎提供的大模型服务平台，支持 DeepSeek、豆包等模型。
// 只需提供 API Key 和模型名称，自动使用 ARK 端点。
//
// 配置示例：
//
//	cfg := Config{
//	    Provider:  ProviderArk,
//	    APIKey:    "your-ark-api-key",
//	    Model:     "deepseek-v3-2-251201",
//	    ArkRegion: "cn-beijing", // 可选，默认 cn-beijing
//	}
func newArk(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	config := &ark.ChatModelConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	}

	// 设置 Region（可选，不设置默认 cn-beijing）
	if cfg.ArkRegion != "" {
		config.Region = cfg.ArkRegion
	}

	// 设置自定义 BaseURL（可选，会覆盖 Region）
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}

	// 设置温度
	if cfg.Temperature > 0 {
		t := float32(cfg.Temperature)
		config.Temperature = &t
	}

	// 设置最大 Token
	if cfg.MaxTokens > 0 {
		config.MaxTokens = &cfg.MaxTokens
	}

	return ark.NewChatModel(ctx, config)
}

// newClaude Claude 模型（待实现）
//
// 目前 eino-ext 暂不支持 Claude，如需使用请：
// 1. 使用 OpenAI 兼容接口 + Claude API Base URL
// 2. 或等待官方支持
func newClaude(ctx context.Context, cfg Config) (model.BaseChatModel, error) {
	logx.Errorf("claude provider not implemented yet, use openai with claude base url")
	return nil, fmt.Errorf("claude provider not implemented: use openai compatible interface with claude base url")
}
