package holiday

import (
	"context"
	"embed"
)

//go:embed data/*.json
var embeddedData embed.FS

// EmbeddedSource 加载包内嵌的节假日数据。
type EmbeddedSource struct{}

// NewEmbeddedSource 创建使用包内嵌 JSON 数据的数据源。
func NewEmbeddedSource() *EmbeddedSource {
	return &EmbeddedSource{}
}

// Load 加载内嵌节假日数据。
func (s *EmbeddedSource) Load(ctx context.Context) (map[string]Entry, error) {
	return loadEntriesFromFS(ctx, embeddedData, "data")
}
