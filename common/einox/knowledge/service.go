package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/einox/metrics"
)

// Service 知识库：创建库、入库、检索。
type Service struct {
	cfg   Config
	emb   Embedder
	store vectorStore

	embedDimMu sync.RWMutex
	embedDim   int // 进程内首次成功 embedding 的维数，用于与后续请求对齐
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
		return nil, fmt.Errorf("knowledge: enabled but embedding.api_key and fallback are empty")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	st, err := newVectorStore(cfg)
	if err != nil {
		return nil, err
	}
	emb := NewArkEmbedder(key, cfg.Embedding.BaseURL, cfg.Embedding.Model, cfg.Embedding.ArkRegion)
	logx.Infof("[knowledge] initialized backend=%s", cfg.EffectiveBackend())
	if cfg.EffectiveBackend() == "memory" {
		logx.Infof("[knowledge] hint: multi-process or shared index with gateway — use backend gorm (same DataDir/DSN), redis, or milvus")
	}
	return &Service{cfg: cfg, emb: emb, store: st}, nil
}

// Close 释放 redis/milvus/gorm 等底层连接（memory 无操作）。
func (s *Service) Close() error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.Close()
}

func logEntityShort(kind, id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return kind + ":empty"
	}
	h := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s:%s", kind, hex.EncodeToString(h[:6]))
}

func uniformEmbeddingDim(vecs [][]float32) (int, error) {
	if len(vecs) == 0 {
		return 0, fmt.Errorf("knowledge: empty embedding batch")
	}
	d := len(vecs[0])
	if d == 0 {
		return 0, fmt.Errorf("knowledge: zero-length embedding vector")
	}
	for i := 1; i < len(vecs); i++ {
		if len(vecs[i]) != d {
			return 0, fmt.Errorf("knowledge: embedding batch has mixed dimensions (%d vs %d)", d, len(vecs[i]))
		}
	}
	return d, nil
}

// assertEmbeddingOutputDim 校验当前 embedding 与配置、历史维数一致。
func (s *Service) assertEmbeddingOutputDim(dim int) error {
	if dim <= 0 {
		return fmt.Errorf("knowledge: embedding vector dimension is invalid")
	}
	if exp := s.cfg.Embedding.ExpectedDim; exp > 0 && dim != exp {
		return fmt.Errorf("knowledge: embedding output dim %d != configured expectedDim %d", dim, exp)
	}
	if s.cfg.EffectiveBackend() == "milvus" {
		if want := s.cfg.Milvus.VectorDim; want > 0 && dim != want {
			return fmt.Errorf("knowledge: embedding output dim %d != Milvus.vectorDim %d", dim, want)
		}
	}
	s.embedDimMu.RLock()
	cached := s.embedDim
	s.embedDimMu.RUnlock()
	if cached == 0 {
		s.embedDimMu.Lock()
		defer s.embedDimMu.Unlock()
		if s.embedDim == 0 {
			s.embedDim = dim
			return nil
		}
		cached = s.embedDim
	}
	if cached != dim {
		return fmt.Errorf("knowledge: embedding dimension mismatch: previously %d, now %d (same model/endpoint required)", cached, dim)
	}
	return nil
}

// CreateBase 新建知识库并返回 ID。
func (s *Service) CreateBase(ctx context.Context, userID, name string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("knowledge service is nil")
	}
	id := uuid.NewString()
	n := strings.TrimSpace(name)
	if n == "" {
		n = "default"
	}
	if err := s.store.CreateBase(ctx, userID, id, n); err != nil {
		return "", err
	}
	return id, nil
}

// ListBases 列出当前用户的知识库。
func (s *Service) ListBases(ctx context.Context, userID string) ([]Base, error) {
	if s == nil {
		return nil, fmt.Errorf("knowledge service is nil")
	}
	return s.store.ListBases(ctx, userID)
}

// DeleteBase 删除知识库及其向量。
func (s *Service) DeleteBase(ctx context.Context, userID, baseID string) error {
	if s == nil {
		return fmt.Errorf("knowledge service is nil")
	}
	return s.store.DeleteBase(ctx, userID, strings.TrimSpace(baseID))
}

// IngestDocument 将纯文本分块、向量化并写入。
func (s *Service) IngestDocument(ctx context.Context, userID, baseID, filename, content string) (*IndexedDocument, error) {
	if s == nil {
		return nil, fmt.Errorf("knowledge service is nil")
	}
	t0 := time.Now()
	be := s.cfg.EffectiveBackend()
	baseID = strings.TrimSpace(baseID)
	fn := strings.TrimSpace(filename)
	if fn == "" {
		fn = "document.txt"
	}
	docs := SplitIntoDocuments(content, s.cfg.EffectiveMaxChunkRunes())
	if len(docs) == 0 {
		return nil, fmt.Errorf("knowledge: empty document after split")
	}
	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.Content
	}
	vecs, err := s.emb.Embed(ctx, texts)
	if err != nil {
		metrics.Global().RecordKnowledge(ctx, "ingest", "error", be, time.Since(t0))
		return nil, err
	}
	dim, err := uniformEmbeddingDim(vecs)
	if err != nil {
		metrics.Global().RecordKnowledge(ctx, "ingest", "error", be, time.Since(t0))
		return nil, err
	}
	if err := s.assertEmbeddingOutputDim(dim); err != nil {
		metrics.Global().RecordKnowledge(ctx, "ingest", "error", be, time.Since(t0))
		return nil, err
	}
	sourceID := uuid.NewString()
	pairs, err := chunkDocumentsToPairs(docs, vecs, sourceID)
	if err != nil {
		metrics.Global().RecordKnowledge(ctx, "ingest", "error", be, time.Since(t0))
		return nil, err
	}
	if err := s.store.UpsertChunks(ctx, userID, baseID, sourceID, fn, pairs); err != nil {
		metrics.Global().RecordKnowledge(ctx, "ingest", "error", be, time.Since(t0))
		return nil, err
	}
	metrics.Global().RecordKnowledge(ctx, "ingest", "ok", be, time.Since(t0))
	logx.WithContext(ctx).Debugf("[knowledge] ingest ok user=%s base=%s chunks=%d",
		logEntityShort("u", userID), logEntityShort("b", baseID), len(docs))
	return &IndexedDocument{ID: sourceID, Filename: fn, Chunks: len(docs), CreatedAt: time.Now()}, nil
}

// DeleteDocument 删除某次入库产生的全部向量块。
func (s *Service) DeleteDocument(ctx context.Context, userID, baseID, sourceID string) error {
	if s == nil {
		return fmt.Errorf("knowledge service is nil")
	}
	return s.store.DeleteSource(ctx, userID, strings.TrimSpace(baseID), strings.TrimSpace(sourceID))
}

// ListDocuments 列出知识库内已索引的源文档。
func (s *Service) ListDocuments(ctx context.Context, userID, baseID string) ([]IndexedDocument, error) {
	if s == nil {
		return nil, fmt.Errorf("knowledge service is nil")
	}
	return s.store.ListSources(ctx, userID, strings.TrimSpace(baseID))
}

// Search 对 query 向量化并在知识库内检索。topK<=0 时使用配置默认 TopK。
func (s *Service) Search(ctx context.Context, userID, baseID, query string, topK int) (*SearchResult, error) {
	if s == nil {
		return nil, fmt.Errorf("knowledge service is nil")
	}
	t0 := time.Now()
	be := s.cfg.EffectiveBackend()
	q := strings.TrimSpace(query)
	if q == "" {
		return &SearchResult{}, nil
	}
	if topK <= 0 {
		topK = s.cfg.EffectiveTopK()
	}
	vecs, err := s.emb.Embed(ctx, []string{q})
	if err != nil {
		metrics.Global().RecordKnowledge(ctx, "search", "error", be, time.Since(t0))
		return nil, err
	}
	if len(vecs) == 0 {
		metrics.Global().RecordKnowledge(ctx, "search", "ok", be, time.Since(t0))
		return &SearchResult{}, nil
	}
	dim, err := uniformEmbeddingDim(vecs)
	if err != nil {
		metrics.Global().RecordKnowledge(ctx, "search", "error", be, time.Since(t0))
		return nil, err
	}
	if err := s.assertEmbeddingOutputDim(dim); err != nil {
		metrics.Global().RecordKnowledge(ctx, "search", "error", be, time.Since(t0))
		return nil, err
	}
	hits, err := s.store.Search(ctx, userID, strings.TrimSpace(baseID), vecs[0], topK)
	if err != nil {
		metrics.Global().RecordKnowledge(ctx, "search", "error", be, time.Since(t0))
		return nil, err
	}
	metrics.Global().RecordKnowledge(ctx, "search", "ok", be, time.Since(t0))
	logx.WithContext(ctx).Debugf("[knowledge] search ok user=%s base=%s hits=%d",
		logEntityShort("u", userID), logEntityShort("b", baseID), len(hits))
	outHits := make([]Citation, 0, len(hits))
	docs := make([]*schema.Document, 0, len(hits))
	for _, h := range hits {
		outHits = append(outHits, Citation{
			ChunkID: h.ChunkID, SourceID: h.SourceID, Filename: h.Filename, Text: h.Text, Score: h.Score,
		})
		docs = append(docs, &schema.Document{
			ID:      h.ChunkID,
			Content: h.Text,
		})
	}
	ctxStr := formatCitationsBlock(outHits)
	return &SearchResult{
		Hits:        outHits,
		Context:     ctxStr,
		Documents:   docs,
		QueryVector: vecs[0],
	}, nil
}

func formatCitationsBlock(hits []Citation) string {
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
