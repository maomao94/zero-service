package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// Service 对外 RAG 服务：集合管理、写入（Indexer）、检索（Retriever）。
type Service struct {
	cfg   Config
	emb   Embedder
	store vectorStore
}

// NewService 构造服务；未启用时返回 (nil, nil)。
func NewService(cfg Config, fallbackAPIKey string) (*Service, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	key := strings.TrimSpace(cfg.Embedding.APIKey)
	if key == "" {
		key = strings.TrimSpace(fallbackAPIKey)
	}
	if key == "" {
		return nil, fmt.Errorf("rag: enabled but embedding.api_key and fallback are empty")
	}
	st, err := newVectorStore(cfg)
	if err != nil {
		return nil, err
	}
	emb := NewArkEmbedder(key, cfg.Embedding.BaseURL, cfg.Embedding.Model, cfg.Embedding.ArkRegion)
	return &Service{cfg: cfg, emb: emb, store: st}, nil
}

// Close 关闭 sqlite 等底层资源。
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	if st, ok := s.store.(*sqliteStore); ok {
		return st.Close()
	}
	return nil
}

// CreateCollection 新建集合并返回 ID。
func (s *Service) CreateCollection(ctx context.Context, userID, name string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("rag service is nil")
	}
	id := uuid.NewString()
	n := strings.TrimSpace(name)
	if n == "" {
		n = "default"
	}
	if err := s.store.CreateCollection(ctx, userID, id, n); err != nil {
		return "", err
	}
	return id, nil
}

// ListCollections 列出当前用户的集合。
func (s *Service) ListCollections(ctx context.Context, userID string) ([]Collection, error) {
	if s == nil {
		return nil, fmt.Errorf("rag service is nil")
	}
	return s.store.ListCollections(ctx, userID)
}

// DeleteCollection 删除集合及其中向量。
func (s *Service) DeleteCollection(ctx context.Context, userID, collectionID string) error {
	if s == nil {
		return fmt.Errorf("rag service is nil")
	}
	return s.store.DeleteCollection(ctx, userID, strings.TrimSpace(collectionID))
}

// IngestText 将纯文本分块、向量化并写入（多格式文件可在网关层读成文本后调用本方法）。
func (s *Service) IngestText(ctx context.Context, userID, collectionID, filename, content string) (*IngestedSource, error) {
	if s == nil {
		return nil, fmt.Errorf("rag service is nil")
	}
	collectionID = strings.TrimSpace(collectionID)
	fn := strings.TrimSpace(filename)
	if fn == "" {
		fn = "document.txt"
	}
	docs := SplitIntoDocuments(content, s.cfg.EffectiveMaxChunkRunes())
	if len(docs) == 0 {
		return nil, fmt.Errorf("rag: empty document after split")
	}
	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.Content
	}
	vecs, err := s.emb.Embed(ctx, texts)
	if err != nil {
		return nil, err
	}
	sourceID := uuid.NewString()
	pairs, err := chunkDocumentsToPairs(docs, vecs, sourceID)
	if err != nil {
		return nil, err
	}
	if err := s.store.UpsertChunks(ctx, userID, collectionID, sourceID, fn, pairs); err != nil {
		return nil, err
	}
	return &IngestedSource{ID: sourceID, Filename: fn, Chunks: len(docs), CreatedAt: time.Now()}, nil
}

// DeleteSource 删除某次写入产生的全部向量块。
func (s *Service) DeleteSource(ctx context.Context, userID, collectionID, sourceID string) error {
	if s == nil {
		return fmt.Errorf("rag service is nil")
	}
	return s.store.DeleteSource(ctx, userID, strings.TrimSpace(collectionID), strings.TrimSpace(sourceID))
}

// ListSources 列出集合内已入库的源文件。
func (s *Service) ListSources(ctx context.Context, userID, collectionID string) ([]IngestedSource, error) {
	if s == nil {
		return nil, fmt.Errorf("rag service is nil")
	}
	return s.store.ListSources(ctx, userID, strings.TrimSpace(collectionID))
}

// Retrieve 对 query 做向量化并在集合内检索（Retriever）。topK<=0 时使用配置默认 TopK。
func (s *Service) Retrieve(ctx context.Context, userID, collectionID, query string, topK int) (*RetrievalResult, error) {
	if s == nil {
		return nil, fmt.Errorf("rag service is nil")
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return &RetrievalResult{}, nil
	}
	if topK <= 0 {
		topK = s.cfg.EffectiveTopK()
	}
	vecs, err := s.emb.Embed(ctx, []string{q})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return &RetrievalResult{}, nil
	}
	hits, err := s.store.Search(ctx, userID, strings.TrimSpace(collectionID), vecs[0], topK)
	if err != nil {
		return nil, err
	}
	outHits := make([]RetrievalHit, 0, len(hits))
	docs := make([]*schema.Document, 0, len(hits))
	for _, h := range hits {
		outHits = append(outHits, RetrievalHit{
			ChunkID: h.ChunkID, SourceID: h.SourceID, Filename: h.Filename, Text: h.Text, Score: h.Score,
		})
		docs = append(docs, &schema.Document{
			ID:      h.ChunkID,
			Content: h.Text,
		})
	}
	ctxStr := buildRAGContextBlock(outHits)
	return &RetrievalResult{
		Hits:        outHits,
		Context:     ctxStr,
		Documents:   docs,
		QueryVector: vecs[0],
	}, nil
}

func buildRAGContextBlock(hits []RetrievalHit) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	for i, h := range hits {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		fmt.Fprintf(&b, "[%d] (score=%.3f", i+1, h.Score)
		if h.Filename != "" {
			fmt.Fprintf(&b, ", file=%s", h.Filename)
		}
		b.WriteString(")\n")
		b.WriteString(strings.TrimSpace(h.Text))
	}
	return b.String()
}
