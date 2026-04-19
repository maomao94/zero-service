package knowledge

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// Base 表示一个知识库（按 user_id + id 隔离）。
type Base struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// IndexedDocument 一次入库的源文档元数据。
type IndexedDocument struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Chunks    int       `json:"chunks"`
	CreatedAt time.Time `json:"createdAt"`
}

// Citation 单次向量检索命中。
type Citation struct {
	ChunkID  string  `json:"chunkId"`
	SourceID string  `json:"sourceId"`
	Filename string  `json:"filename,omitempty"`
	Text     string  `json:"text"`
	Score    float64 `json:"score"`
}

// SearchResult 检索聚合结果。
type SearchResult struct {
	Hits        []Citation         `json:"hits"`
	Context     string             `json:"context"`
	Documents   []*schema.Document `json:"-"`
	QueryVector []float32          `json:"-"`
}
