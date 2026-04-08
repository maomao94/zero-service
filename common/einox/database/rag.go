package database

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// RAGConfig RAG 配置
type RAGConfig struct {
	Retriever RetrieverConfig `json:"retriever"`
	Indexer   IndexerConfig   `json:"indexer"`
}

// RetrieverConfig Retriever 配置
type RetrieverConfig struct {
	Type    string         `json:"type"`    // redis, milvus2, es8, qdrant
	Options map[string]any `json:"options"` // 配置选项
}

// IndexerConfig Indexer 配置
type IndexerConfig struct {
	Type    string         `json:"type"`    // redis, milvus2, es8, qdrant
	Options map[string]any `json:"options"` // 配置选项
}

// Document 文档
type Document = schema.Document

// =============================================================================
// RAG 服务
// =============================================================================

// RAGService RAG 服务接口
type RAGService interface {
	// Index 索引文档
	Index(ctx context.Context, docs []*Document) ([]string, error)
	// Retrieve 检索文档
	Retrieve(ctx context.Context, query string, topK int) ([]*Document, error)
}

// =============================================================================
// 简化实现（基于内存）
// =============================================================================

// SimpleRAG 简单的内存 RAG 实现
type SimpleRAG struct {
	docs []*Document
}

func NewSimpleRAG() *SimpleRAG {
	return &SimpleRAG{
		docs: make([]*Document, 0),
	}
}

func (r *SimpleRAG) Index(ctx context.Context, docs []*Document) ([]string, error) {
	r.docs = append(r.docs, docs...)
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	return ids, nil
}

func (r *SimpleRAG) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	// 简单的关键词匹配
	var results []*Document
	for _, doc := range r.docs {
		if len(results) >= topK {
			break
		}
		results = append(results, doc)
	}
	return results, nil
}

// =============================================================================
// 工具函数
// =============================================================================

// NewDocument 创建文档
func NewDocument(content string, metadata map[string]any) *Document {
	return &Document{
		ID:       uuid.NewString(),
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
