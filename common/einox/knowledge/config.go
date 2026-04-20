package knowledge

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Config 描述向量知识库：Embedding + 存储后端 + 分块参数。YAML 键名为 knowledge。
// 后端与 Eino 生态对齐思路：memory 进程内；gorm 任意 SQL（含 sqlite 文件）；redis 共享缓存；milvus 向量库。
type Config struct {
	Enabled bool `json:"enabled,optional"`
	// Backend: memory | gorm | redis | milvus | sqlite（sqlite 为 gorm+本地文件的别名，兼容旧配置）
	Backend string `json:"backend,optional,default=memory"`
	DataDir string `json:"dataDir,optional,default=./data/knowledge"`
	// DSN 非空时 gorm 后端按 DSN 连接（mysql/postgres/sqlite）；空且 Backend=gorm|sqlite 时用 DataDir 下 einox_knowledge.sqlite
	DSN string `json:"dsn,optional"`

	Redis RedisConfig `json:"redis,optional"`

	Milvus MilvusConfig `json:"milvus,optional"`

	// MaxChunksPerBase 仅 backend=redis 时生效：单库下 chunk 数软上限，0 表示不限制（大规模请用 milvus）。
	MaxChunksPerBase int `json:"maxChunksPerBase,optional"`

	Embedding struct {
		Provider  string `json:"provider,optional,default=ark"`
		APIKey    string `json:"api_key,optional"`
		BaseURL   string `json:"base_url,optional"`
		Model     string `json:"model,optional"`
		ArkRegion string `json:"ark_region,optional"`
		// ExpectedDim 非 0 时强制校验 embedding 输出维（与模型一致；Milvus 时也应与 Milvus.vectorDim 一致）。
		ExpectedDim int `json:"expectedDim,optional"`
	} `json:"embedding,optional"`
	TopK              int `json:"topK,optional,default=5"`
	MaxChunkRunes     int `json:"maxChunkRunes,optional,default=900"`
	ChunkOverlapRunes int `json:"chunkOverlapRunes,optional,default=120"`
}

// RedisConfig go-redis 单机/哨兵可扩展；此处 Addr 必填（host:port）。
type RedisConfig struct {
	Addr     string `json:"addr,optional"`
	Username string `json:"username,optional"`
	Password string `json:"password,optional"`
	DB       int    `json:"db,optional"`
}

// MilvusConfig 对齐 milvus-sdk-go / eino-ext indexer 使用场景；VectorDim 须与 Embedding 模型输出维一致。
type MilvusConfig struct {
	Addr       string `json:"addr,optional"`
	Username   string `json:"username,optional"`
	Password   string `json:"password,optional"`
	Collection string `json:"collection,optional"`
	// VectorDim 向量维度，必填（由所用 embedding 模型决定，如 1024/1536）
	VectorDim int `json:"vectorDim,optional"`
}

// Validate 在连接存储前检查与 backend 匹配的必填项，避免启动到一半才失败。
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	switch c.EffectiveBackend() {
	case "redis":
		if strings.TrimSpace(c.Redis.Addr) == "" {
			return fmt.Errorf("knowledge: backend=redis requires Redis.addr (e.g. 127.0.0.1:6379)")
		}
	case "milvus":
		if strings.TrimSpace(c.Milvus.Addr) == "" {
			return fmt.Errorf("knowledge: backend=milvus requires Milvus.addr")
		}
		if c.Milvus.VectorDim <= 0 {
			return fmt.Errorf("knowledge: backend=milvus requires Milvus.vectorDim > 0 (must match embedding model output dimension)")
		}
		if ed := c.Embedding.ExpectedDim; ed > 0 && ed != c.Milvus.VectorDim {
			return fmt.Errorf("knowledge: embedding.expectedDim (%d) must equal Milvus.vectorDim (%d)", ed, c.Milvus.VectorDim)
		}
	case "gorm":
		// DSN 为空时使用 DataDir 下 sqlite 文件，无需额外字段
	default:
	}
	return nil
}

// EffectiveBackend 归一化存储后端名。
func (c Config) EffectiveBackend() string {
	b := strings.ToLower(strings.TrimSpace(c.Backend))
	switch b {
	case "memory", "gorm", "redis", "milvus":
		return b
	case "sqlite":
		return "gorm"
	default:
		return "memory"
	}
}

// EffectiveDataDir 数据目录。
func (c Config) EffectiveDataDir() string {
	d := strings.TrimSpace(c.DataDir)
	if d == "" {
		return "./data/knowledge"
	}
	if filepath.IsAbs(d) {
		return filepath.Clean(d)
	}
	return filepath.Clean(d)
}

// EffectiveMaxChunksPerBase Redis 下单库 chunk 上限，0 不限制。
func (c Config) EffectiveMaxChunksPerBase() int {
	if c.MaxChunksPerBase < 0 {
		return 0
	}
	return c.MaxChunksPerBase
}

// EffectiveMilvusCollection 集合名。
func (c Config) EffectiveMilvusCollection() string {
	s := strings.TrimSpace(c.Milvus.Collection)
	if s == "" {
		return "einox_knowledge"
	}
	return s
}

// EffectiveTopK 默认 top-k。
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
