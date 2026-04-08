package database

import (
	"context"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

func newOpenAIEmbedder(ctx context.Context, cfg EmbeddingConfig) (embedding.Embedder, error) {
	return openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	})
}

func newOllamaEmbedder(ctx context.Context, cfg EmbeddingConfig) (embedding.Embedder, error) {
	url := cfg.BaseURL
	if url == "" {
		url = "http://localhost:11434"
	}
	return ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
		Model:   cfg.Model,
		BaseURL: url,
	})
}
