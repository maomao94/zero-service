package holiday

import (
	"context"
	"os"
)

// DirSource 从文件系统目录加载节假日 JSON 数据。
type DirSource struct {
	path string
}

// NewDirSource 创建从目录加载 yyyy.json 文件的数据源。
func NewDirSource(path string) *DirSource {
	return &DirSource{path: path}
}

// Load 从配置目录加载节假日数据。
func (s *DirSource) Load(ctx context.Context) (map[string]Entry, error) {
	return loadEntriesFromFS(ctx, os.DirFS(s.path), ".")
}
