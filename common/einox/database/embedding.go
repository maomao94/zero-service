package database

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"
)

// EmbeddingConfig Embedding 配置
type EmbeddingConfig struct {
	Provider  string `json:"provider"` // openai, ollama, dashscope
	APIKey    string `json:"api_key"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	BatchSize int    `json:"batch_size"` // 批处理大小
}

// NewEmbedder 创建 Embedder
func NewEmbedder(ctx context.Context, cfg EmbeddingConfig) (embedding.Embedder, error) {
	// 根据 provider 选择实现
	switch cfg.Provider {
	case "openai":
		return newOpenAIEmbedder(ctx, cfg)
	case "ollama":
		return newOllamaEmbedder(ctx, cfg)
	default:
		return nil, nil
	}
}
