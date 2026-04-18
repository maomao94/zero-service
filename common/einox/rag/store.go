package rag

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// vectorStore 向量索引持久化（Indexer 后端），对用户 + 集合隔离。
type vectorStore interface {
	CreateCollection(ctx context.Context, userID, id, name string) error
	DeleteCollection(ctx context.Context, userID, collectionID string) error
	ListCollections(ctx context.Context, userID string) ([]Collection, error)

	UpsertChunks(ctx context.Context, userID, collectionID, sourceID, filename string, pairs []chunkVectorPair) error
	DeleteSource(ctx context.Context, userID, collectionID, sourceID string) error
	ListSources(ctx context.Context, userID, collectionID string) ([]IngestedSource, error)

	// Search 返回按相似度降序的命中（Retriever）。
	Search(ctx context.Context, userID, collectionID string, query []float32, topK int) ([]storedHit, error)
}

type chunkVectorPair struct {
	ChunkID string
	Text    string
	Vector  []float32
}

type storedHit struct {
	ChunkID  string
	SourceID string
	Filename string
	Text     string
	Score    float64
}

// newVectorStore 按配置构造存储后端。
func newVectorStore(cfg Config) (vectorStore, error) {
	switch cfg.EffectiveBackend() {
	case "memory":
		return newMemoryStore(), nil
	case "sqlite":
		return newSQLiteStore(cfg.EffectiveDataDir())
	case "postgres":
		return nil, fmt.Errorf("rag backend %q is not implemented; use sqlite or memory", cfg.Backend)
	default:
		return newSQLiteStore(cfg.EffectiveDataDir())
	}
}

// chunkDocumentsToPairs 将文档块与向量按序配对（长度须一致）。
func chunkDocumentsToPairs(docs []*schema.Document, vecs [][]float32, idPrefix string) ([]chunkVectorPair, error) {
	if len(docs) != len(vecs) {
		return nil, fmt.Errorf("rag: %d chunks vs %d vectors", len(docs), len(vecs))
	}
	pairs := make([]chunkVectorPair, len(docs))
	for i := range docs {
		pairs[i] = chunkVectorPair{
			ChunkID: fmt.Sprintf("%s-%d", idPrefix, i),
			Text:    docs[i].Content,
			Vector:  vecs[i],
		}
	}
	return pairs, nil
}
