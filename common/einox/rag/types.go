package rag

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// Collection 对应一个可被索引与检索的文档集合（知识库/索引名）。
// 与多租户场景下按 user_id + collection_id 隔离存储一致。
type Collection struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// IngestedSource 一次写入的源文件元数据（对应 chatwitheino 中 load 阶段的单文件语义）。
type IngestedSource struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Chunks    int       `json:"chunks"` // 分块后的 *schema.Document 条数
	CreatedAt time.Time `json:"createdAt"`
}

// RetrievalHit 单次向量检索命中（Retriever 单条结果，分数为相似度）。
// Text 为写入索引时的 Document.Content 摘要或全文，供前端卡片与引用展示。
type RetrievalHit struct {
	ChunkID  string  `json:"chunkId"`
	SourceID string  `json:"sourceId"` // 源文件/文档 ID，对应 IngestedSource.ID
	Filename string  `json:"filename,omitempty"`
	Text     string  `json:"text"`
	Score    float64 `json:"score"`
}

// RetrievalResult 一次检索的聚合结果：命中列表 + 拼进模型的上下文 + 可选的原始文档块。
// Documents 与 chatwitheino 中 filter 之后传入 answer 节点的 []*schema.Document 角色一致。
type RetrievalResult struct {
	Hits        []RetrievalHit     `json:"hits"`
	Context     string             `json:"context"` // 已排版、可直接拼进 system/user 的文本
	Documents   []*schema.Document `json:"-"`       // 可选：供 Graph/Chain 继续编排
	QueryVector []float32          `json:"-"`       // 调试/可观测，默认不序列化给前端
}
