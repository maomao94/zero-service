package knowledge

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// vectorStore 知识库向量与元数据存储抽象；实现含 memory / gorm / redis / milvus。
// 设计参考 Eino 的 Indexer+Retriever 分层，本包对外仍保持 CreateBase/Ingest/Search 业务 API。
type vectorStore interface {
	CreateBase(ctx context.Context, userID, id, name string) error
	DeleteBase(ctx context.Context, userID, baseID string) error
	ListBases(ctx context.Context, userID string) ([]Base, error)

	UpsertChunks(ctx context.Context, userID, baseID, sourceID, filename string, pairs []chunkVectorPair) error
	DeleteSource(ctx context.Context, userID, baseID, sourceID string) error
	ListSources(ctx context.Context, userID, baseID string) ([]IndexedDocument, error)

	Search(ctx context.Context, userID, baseID string, query []float32, topK int) ([]storedHit, error)

	Close() error
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

func newVectorStore(cfg Config) (vectorStore, error) {
	switch cfg.EffectiveBackend() {
	case "memory":
		return newMemoryStore(), nil
	case "gorm":
		return newGORMStore(cfg)
	case "redis":
		return newRedisStore(cfg)
	case "milvus":
		return newMilvusStore(cfg)
	default:
		return newMemoryStore(), nil
	}
}

func chunkDocumentsToPairs(docs []*schema.Document, vecs [][]float32, idPrefix string) ([]chunkVectorPair, error) {
	if len(docs) != len(vecs) {
		return nil, fmt.Errorf("knowledge: %d chunks vs %d vectors", len(docs), len(vecs))
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
