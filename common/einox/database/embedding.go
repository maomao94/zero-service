package database

import (
	"context"
	"fmt"

	ark "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

// =============================================================================
// Embedding Provider 类型定义
// =============================================================================

// EmbeddingProviderType Embedding 提供商类型
type EmbeddingProviderType string

const (
	EmbeddingProviderOpenAI     EmbeddingProviderType = "openai"
	EmbeddingProviderOllama     EmbeddingProviderType = "ollama"
	EmbeddingProviderDashScope  EmbeddingProviderType = "dashscope"  // 阿里通义
	EmbeddingProviderArk        EmbeddingProviderType = "ark"        // 火山引擎
	EmbeddingProviderVolcEngine EmbeddingProviderType = "volcengine" // 火山引擎别名
)

// =============================================================================
// 配置定义
// =============================================================================

// EmbeddingConfig Embedding 配置
type EmbeddingConfig struct {
	Provider  EmbeddingProviderType `json:"provider"`   // 提供商类型
	APIKey    string                `json:"api_key"`    // API Key
	BaseURL   string                `json:"base_url"`   // 基础 URL（可选）
	Model     string                `json:"model"`      // 模型名称
	BatchSize int                   `json:"batch_size"` // 批处理大小
	Dimension int                   `json:"dimension"`  // 向量维度（可选）

	// 火山引擎特有配置
	ArkRegion string `json:"ark_region"` // 火山引擎区域，默认 cn-beijing
}

// DefaultEmbeddingConfig 默认配置
func DefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Provider:  EmbeddingProviderOpenAI,
		BatchSize: 20,
	}
}

// =============================================================================
// Embedder 工厂
// =============================================================================

// NewEmbedder 创建 Embedder
func NewEmbedder(ctx context.Context, cfg *EmbeddingConfig) (embedding.Embedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("embedding config is nil")
	}

	// 设置默认值
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 20
	}

	switch cfg.Provider {
	case EmbeddingProviderOpenAI:
		return newOpenAIEmbedder(ctx, cfg)
	case EmbeddingProviderOllama:
		return newOllamaEmbedder(ctx, cfg)
	case EmbeddingProviderDashScope:
		return newDashScopeEmbedder(ctx, cfg)
	case EmbeddingProviderArk, EmbeddingProviderVolcEngine:
		return newArkEmbedder(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}
}

// =============================================================================
// OpenAI Embedder
// =============================================================================

func newOpenAIEmbedder(ctx context.Context, cfg *EmbeddingConfig) (embedding.Embedder, error) {
	config := &openai.EmbeddingConfig{
		Model: cfg.Model,
	}

	if cfg.APIKey != "" {
		config.APIKey = cfg.APIKey
	}
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}

	return openai.NewEmbedder(ctx, config)
}

// =============================================================================
// Ollama Embedder
// =============================================================================

func newOllamaEmbedder(ctx context.Context, cfg *EmbeddingConfig) (embedding.Embedder, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
		Model:   cfg.Model,
		BaseURL: baseURL,
	})
}

// =============================================================================
// DashScope (阿里通义) Embedder
// =============================================================================

func newDashScopeEmbedder(ctx context.Context, cfg *EmbeddingConfig) (embedding.Embedder, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	return openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		Model:   cfg.Model,
		BaseURL: baseURL,
		APIKey:  cfg.APIKey,
	})
}

// =============================================================================
// Ark (火山引擎) Embedder
// =============================================================================

func newArkEmbedder(ctx context.Context, cfg *EmbeddingConfig) (embedding.Embedder, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		region := cfg.ArkRegion
		if region == "" {
			region = "cn-beijing"
		}
		baseURL = fmt.Sprintf("https://ark.%s.volces.com/api/v3", region)
	}

	config := &ark.EmbeddingConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: baseURL,
	}

	return ark.NewEmbedder(ctx, config)
}

// =============================================================================
// 便捷构造函数
// =============================================================================

// NewOpenAIEmbedder 创建 OpenAI Embedder
func NewOpenAIEmbedder(ctx context.Context, apiKey, model string, baseURL ...string) (embedding.Embedder, error) {
	cfg := &EmbeddingConfig{
		Provider: EmbeddingProviderOpenAI,
		APIKey:   apiKey,
		Model:    model,
	}
	if len(baseURL) > 0 && baseURL[0] != "" {
		cfg.BaseURL = baseURL[0]
	}
	return NewEmbedder(ctx, cfg)
}

// NewArkEmbedder 创建火山引擎 Embedder
func NewArkEmbedder(ctx context.Context, apiKey, model string, region ...string) (embedding.Embedder, error) {
	cfg := &EmbeddingConfig{
		Provider: EmbeddingProviderArk,
		APIKey:   apiKey,
		Model:    model,
	}
	if len(region) > 0 && region[0] != "" {
		cfg.ArkRegion = region[0]
	}
	return NewEmbedder(ctx, cfg)
}

// NewDashScopeEmbedder 创建阿里通义 Embedder
func NewDashScopeEmbedder(ctx context.Context, apiKey, model string) (embedding.Embedder, error) {
	cfg := &EmbeddingConfig{
		Provider: EmbeddingProviderDashScope,
		APIKey:   apiKey,
		Model:    model,
	}
	return NewEmbedder(ctx, cfg)
}
