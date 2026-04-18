package rag

import (
	"path/filepath"
	"strings"
)

// Config 描述 RAG 向量链路：Embedding（火山 ARK 等）+ 向量存储后端 + 分块参数。
// YAML 建议键名 rag（与包名一致）；若沿用 knowledge 键，可在业务 yaml 中做别名映射。
type Config struct {
	Enabled bool `json:"enabled,optional"`
	// Backend 向量索引持久化实现：memory | sqlite | postgres（与 eino-ext 多后端思路一致，此处为 einox 自管存储）。
	Backend string `json:"backend,optional,default=sqlite"`
	DataDir string `json:"dataDir,optional,default=./data/rag"`
	// DSN 仅 postgres 后端使用。
	DSN string `json:"dsn,optional"`
	// Embedding 调用配置（OpenAI 兼容 /embeddings，火山 ARK 同域）。
	Embedding struct {
		Provider  string `json:"provider,optional,default=ark"` // ark
		APIKey    string `json:"api_key,optional"`
		BaseURL   string `json:"base_url,optional"`
		Model     string `json:"model,optional"`      // 向量模型接入点 ID
		ArkRegion string `json:"ark_region,optional"` // cn-beijing | cn-shanghai
	} `json:"embedding,optional"`
	// TopK 检索返回条数上限（Retriever top-k）。
	TopK int `json:"topK,optional,default=5"`
	// MaxChunkRunes 单块最大字符（与 chatwitheino splitIntoChunks 的 chunkSize 同角色）。
	MaxChunkRunes int `json:"maxChunkRunes,optional,default=900"`
	// ChunkOverlapRunes 块重叠长度。
	ChunkOverlapRunes int `json:"chunkOverlapRunes,optional,default=120"`
}

// EffectiveBackend 归一化存储后端名。
func (c Config) EffectiveBackend() string {
	b := strings.ToLower(strings.TrimSpace(c.Backend))
	switch b {
	case "memory", "sqlite", "postgres":
		return b
	default:
		return "sqlite"
	}
}

// EffectiveDataDir 数据目录（sqlite 文件等）。
func (c Config) EffectiveDataDir() string {
	d := strings.TrimSpace(c.DataDir)
	if d == "" {
		return "./data/rag"
	}
	if filepath.IsAbs(d) {
		return filepath.Clean(d)
	}
	return filepath.Clean(d)
}

// EffectiveTopK Retriever 默认 top-k。
func (c Config) EffectiveTopK() int {
	if c.TopK <= 0 {
		return 5
	}
	if c.TopK > 50 {
		return 50
	}
	return c.TopK
}

// EffectiveMaxChunkRunes 分块大小。
func (c Config) EffectiveMaxChunkRunes() int {
	if c.MaxChunkRunes <= 0 {
		return 900
	}
	return c.MaxChunkRunes
}

// EffectiveChunkOverlapRunes 分块重叠。
func (c Config) EffectiveChunkOverlapRunes() int {
	if c.ChunkOverlapRunes <= 0 {
		return 120
	}
	return c.ChunkOverlapRunes
}
