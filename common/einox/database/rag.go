package database

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// 向量库类型定义
// =============================================================================

// VectorStoreType 向量库类型
type VectorStoreType string

const (
	VectorStoreRedis    VectorStoreType = "redis"    // Redis
	VectorStoreMilvus   VectorStoreType = "milvus"   // Milvus 2.x
	VectorStoreQdrant   VectorStoreType = "qdrant"   // Qdrant
	VectorStoreES       VectorStoreType = "es"       // Elasticsearch 8.x
	VectorStoreMemory   VectorStoreType = "memory"   // 内存（开发测试用）
	VectorStoreChroma   VectorStoreType = "chroma"   // Chroma
	VectorStorePgVector VectorStoreType = "pgvector" // PostgreSQL pgvector
)

// =============================================================================
// 配置定义
// =============================================================================

// VectorStoreConfig 向量库配置
type VectorStoreConfig struct {
	// 向量库类型
	Type VectorStoreType `json:"type"`

	// 连接配置
	Host     string `json:"host"`     // 主机地址
	Port     int    `json:"port"`     // 端口
	Username string `json:"username"` // 用户名（可选）
	Password string `json:"password"` // 密码（可选）
	Database string `json:"database"` // 数据库名/索引名

	// 向量配置
	Collection string `json:"collection"` // 集合名/表名
	Dimension  int    `json:"dimension"`  // 向量维度

	// Redis 特有配置
	RedisKeyPrefix string `json:"redis_key_prefix"` // Redis key 前缀

	// Milvus 特有配置
	MilvusPartition string `json:"milvus_partition"` // Milvus 分区名

	// Qdrant 特有配置
	QdrantGrpcPort int `json:"qdrant_grpc_port"` // Qdrant gRPC 端口

	// ES 特有配置
	ESIndex string `json:"es_index"` // ES 索引名

	// 其他配置
	Extra map[string]any `json:"extra"` // 额外配置
}

// RAGConfig RAG 配置
type RAGConfig struct {
	// 向量库配置
	VectorStore VectorStoreConfig `json:"vector_store"`

	// Embedding 配置
	Embedding EmbeddingConfig `json:"embedding"`

	// 检索配置
	TopK           int     `json:"top_k"`           // 返回文档数量
	ScoreThreshold float64 `json:"score_threshold"` // 相似度阈值

	// 索引配置
	BatchSize int `json:"batch_size"` // 批处理大小
}

// DefaultRAGConfig 默认 RAG 配置
func DefaultRAGConfig() *RAGConfig {
	return &RAGConfig{
		VectorStore: VectorStoreConfig{
			Type: VectorStoreMemory,
		},
		TopK:           5,
		ScoreThreshold: 0.7,
		BatchSize:      20,
	}
}

// =============================================================================
// RAG 服务接口
// =============================================================================

// RAGService RAG 服务接口
type RAGService interface {
	// Index 索引文档
	Index(ctx context.Context, docs []*Document) ([]string, error)
	// Retrieve 检索文档
	Retrieve(ctx context.Context, query string, topK int) ([]*Document, error)
	// Delete 删除文档
	Delete(ctx context.Context, ids []string) error
	// Close 关闭连接
	Close() error
}

// =============================================================================
// 统一 RAG 实现
// =============================================================================

// UnifiedRAG 统一 RAG 实现
type UnifiedRAG struct {
	config    *RAGConfig
	embedder  embedding.Embedder
	indexer   indexer.Indexer
	retriever retriever.Retriever
}

// NewRAGService 创建 RAG 服务
func NewRAGService(ctx context.Context, cfg *RAGConfig) (RAGService, error) {
	if cfg == nil {
		cfg = DefaultRAGConfig()
	}

	// 创建 Embedder
	embedder, err := NewEmbedder(ctx, &cfg.Embedding)
	if err != nil {
		return nil, fmt.Errorf("create embedder failed: %w", err)
	}

	// 根据向量库类型创建对应实现
	switch cfg.VectorStore.Type {
	case VectorStoreMemory:
		return newMemoryRAG(cfg, embedder)
	case VectorStoreRedis:
		return newRedisRAG(ctx, cfg, embedder)
	case VectorStoreMilvus:
		return newMilvusRAG(ctx, cfg, embedder)
	case VectorStoreQdrant:
		return newQdrantRAG(ctx, cfg, embedder)
	case VectorStoreES:
		return newESRAG(ctx, cfg, embedder)
	default:
		// 默认使用内存实现
		return newMemoryRAG(cfg, embedder)
	}
}

// =============================================================================
// 内存 RAG 实现
// =============================================================================

// MemoryRAG 内存 RAG 实现（用于开发测试）
type MemoryRAG struct {
	config   *RAGConfig
	embedder embedding.Embedder
	docs     []*Document
	vectors  [][]float64
}

func newMemoryRAG(cfg *RAGConfig, embedder embedding.Embedder) (*MemoryRAG, error) {
	return &MemoryRAG{
		config:   cfg,
		embedder: embedder,
		docs:     make([]*Document, 0),
		vectors:  make([][]float64, 0),
	}, nil
}

// Index 索引文档
func (r *MemoryRAG) Index(ctx context.Context, docs []*Document) ([]string, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	// 生成向量
	contents := make([]string, len(docs))
	for i, doc := range docs {
		contents[i] = doc.Content
	}

	vectors, err := r.embedder.EmbedStrings(ctx, contents)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	// 存储
	ids := make([]string, len(docs))
	for i, doc := range docs {
		if doc.ID == "" {
			doc.ID = generateID()
		}
		ids[i] = doc.ID
		r.docs = append(r.docs, doc)
		r.vectors = append(r.vectors, vectors[i])
	}

	return ids, nil
}

// Retrieve 检索文档
func (r *MemoryRAG) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	if len(r.docs) == 0 {
		return nil, nil
	}

	// 生成查询向量
	queryVectors, err := r.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embedding query failed: %w", err)
	}
	queryVector := queryVectors[0]

	// 计算相似度
	scored := make([]struct {
		doc   *Document
		score float64
	}, len(r.docs))
	for i, doc := range r.docs {
		score := cosineSimilarity(queryVector, r.vectors[i])
		scored[i] = struct {
			doc   *Document
			score float64
		}{doc: doc, score: score}
	}

	// 排序并返回 topK
	sortScoredDocs(scored)

	if topK <= 0 {
		topK = r.config.TopK
	}
	if topK > len(scored) {
		topK = len(scored)
	}

	results := make([]*Document, topK)
	for i := 0; i < topK; i++ {
		results[i] = scored[i].doc
	}

	return results, nil
}

// Delete 删除文档
func (r *MemoryRAG) Delete(ctx context.Context, ids []string) error {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	newDocs := make([]*Document, 0)
	newVectors := make([][]float64, 0)
	for i, doc := range r.docs {
		if !idSet[doc.ID] {
			newDocs = append(newDocs, doc)
			newVectors = append(newVectors, r.vectors[i])
		}
	}
	r.docs = newDocs
	r.vectors = newVectors

	return nil
}

// Close 关闭连接
func (r *MemoryRAG) Close() error {
	return nil
}

// =============================================================================
// Redis RAG 实现（占位，需要 eino-ext 支持）
// =============================================================================

type redisRAG struct {
	config   *RAGConfig
	embedder embedding.Embedder
	// 实际实现需要引入 eino-ext/components/indexer/redis 和 retriever/redis
}

func newRedisRAG(ctx context.Context, cfg *RAGConfig, embedder embedding.Embedder) (*redisRAG, error) {
	// TODO: 实际实现需要引入 eino-ext 的 Redis 组件
	// 这里返回一个占位实现，实际使用时需要完善
	return &redisRAG{
		config:   cfg,
		embedder: embedder,
	}, nil
}

func (r *redisRAG) Index(ctx context.Context, docs []*Document) ([]string, error) {
	// TODO: 实现 Redis 索引
	return nil, fmt.Errorf("redis RAG not implemented, please use eino-ext/components/indexer/redis")
}

func (r *redisRAG) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	// TODO: 实现 Redis 检索
	return nil, fmt.Errorf("redis RAG not implemented, please use eino-ext/components/retriever/redis")
}

func (r *redisRAG) Delete(ctx context.Context, ids []string) error {
	return fmt.Errorf("redis RAG not implemented")
}

func (r *redisRAG) Close() error {
	return nil
}

// =============================================================================
// Milvus RAG 实现（占位）
// =============================================================================

type milvusRAG struct {
	config   *RAGConfig
	embedder embedding.Embedder
}

func newMilvusRAG(ctx context.Context, cfg *RAGConfig, embedder embedding.Embedder) (*milvusRAG, error) {
	return &milvusRAG{
		config:   cfg,
		embedder: embedder,
	}, nil
}

func (r *milvusRAG) Index(ctx context.Context, docs []*Document) ([]string, error) {
	return nil, fmt.Errorf("milvus RAG not implemented, please use eino-ext/components/indexer/milvus")
}

func (r *milvusRAG) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	return nil, fmt.Errorf("milvus RAG not implemented, please use eino-ext/components/retriever/milvus")
}

func (r *milvusRAG) Delete(ctx context.Context, ids []string) error {
	return fmt.Errorf("milvus RAG not implemented")
}

func (r *milvusRAG) Close() error {
	return nil
}

// =============================================================================
// Qdrant RAG 实现（占位）
// =============================================================================

type qdrantRAG struct {
	config   *RAGConfig
	embedder embedding.Embedder
}

func newQdrantRAG(ctx context.Context, cfg *RAGConfig, embedder embedding.Embedder) (*qdrantRAG, error) {
	return &qdrantRAG{
		config:   cfg,
		embedder: embedder,
	}, nil
}

func (r *qdrantRAG) Index(ctx context.Context, docs []*Document) ([]string, error) {
	return nil, fmt.Errorf("qdrant RAG not implemented, please use eino-ext/components/indexer/qdrant")
}

func (r *qdrantRAG) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	return nil, fmt.Errorf("qdrant RAG not implemented, please use eino-ext/components/retriever/qdrant")
}

func (r *qdrantRAG) Delete(ctx context.Context, ids []string) error {
	return fmt.Errorf("qdrant RAG not implemented")
}

func (r *qdrantRAG) Close() error {
	return nil
}

// =============================================================================
// Elasticsearch RAG 实现（占位）
// =============================================================================

type esRAG struct {
	config   *RAGConfig
	embedder embedding.Embedder
}

func newESRAG(ctx context.Context, cfg *RAGConfig, embedder embedding.Embedder) (*esRAG, error) {
	return &esRAG{
		config:   cfg,
		embedder: embedder,
	}, nil
}

func (r *esRAG) Index(ctx context.Context, docs []*Document) ([]string, error) {
	return nil, fmt.Errorf("elasticsearch RAG not implemented, please use eino-ext/components/indexer/es")
}

func (r *esRAG) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	return nil, fmt.Errorf("elasticsearch RAG not implemented, please use eino-ext/components/retriever/es")
}

func (r *esRAG) Delete(ctx context.Context, ids []string) error {
	return fmt.Errorf("elasticsearch RAG not implemented")
}

func (r *esRAG) Close() error {
	return nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// Document 文档别名
type Document = schema.Document

// NewDocument 创建文档
func NewDocument(content string, metadata map[string]any) *Document {
	return &Document{
		ID:       generateID(),
		Content:  content,
		MetaData: metadata,
	}
}

// NewDocuments 批量创建文档
func NewDocuments(contents []string, metadata map[string]any) []*Document {
	docs := make([]*Document, len(contents))
	for i, content := range contents {
		docs[i] = NewDocument(content, metadata)
	}
	return docs
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("doc_%d", len(make([]byte, 0)))
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (sqrt(normA) * sqrt(normB))
}

// sortScoredDocs 排序（简单冒泡，生产环境应使用 sort.Slice）
func sortScoredDocs(scored []struct {
	doc   *Document
	score float64
}) {
	n := len(scored)
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}

// sqrt 简单平方根实现
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
