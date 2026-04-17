package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config aisolo 服务配置
type Config struct {
	zrpc.RpcServerConf

	Model        ModelConfig        `json:"model"`
	Memory       MemoryConfig       `json:"memory"`
	SessionStore SessionStoreConfig `json:",optional"`
	DB           DBConfig           `json:",optional"`
	Tools        ToolsConfig        `json:"tools"`
	Skills       SkillsConfig       `json:"skills"`
	Agent        AgentConfig        `json:"agent"`
	Checkpoint   CheckpointConfig   `json:"checkpoint"`
	Metrics      MetricsConfig      `json:"metrics"`
	Limit        LimitConfig        `json:"limit"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	PoolMaxIdle int             `json:"poolMaxIdle"`
	PoolMaxLive time.Duration   `json:"poolMaxLive"`
	Deep        DeepAgentConfig `json:"deep,optional"`
}

// DeepAgentConfig Deep 模式专属（见 blueprint_deep）。
type DeepAgentConfig struct {
	// DisableLocalFilesystem 为 true 时不挂载 Eino 本地文件系统工具（含 grep，依赖本机 ripgrep）。
	// 为 false 或未配置时启用。Skill 中间件仍可由 Skills 单独打开，与该项独立。
	DisableLocalFilesystem bool `json:"disableLocalFilesystem,optional"`
	// FilesystemAllowedRoots 用户可见工作区（知识库/项目根等），绝对路径在启动时解析；与 SessionBaseDir 独立。
	// 未配置且未配置 SessionBaseDir 时 Deep 文件工具不限制路径（历史行为）。
	FilesystemAllowedRoots []string `json:"filesystemAllowedRoots,optional"`
	// FilesystemSessionBaseDir 会话工作区父目录；每会话子目录为 filepath.Join(该目录, sessionId)。
	// 非空时 Validate 要求目录已存在；CreateSession / Ask 会 MkdirAll 会话子目录。
	FilesystemSessionBaseDir string `json:"filesystemSessionBaseDir,optional"`
	// FilesystemPolicy 按区域控制读/写/改；各字段见 default tag（用户区默认只读，会话区默认可写可改）。
	FilesystemPolicy DeepFilesystemPolicy `json:"filesystemPolicy,optional"`
	// FilesystemLegacyUserRootsFullAccess 为 true（默认）且仅配置了用户 roots、未配会话目录时，
	// 在用户 roots 内保持历史行为（读写改均允许）。设为 false 则强制使用 FilesystemPolicy。
	FilesystemLegacyUserRootsFullAccess bool `json:"filesystemLegacyUserRootsFullAccess,optional,default=true"`
}

// DeepFilesystemPolicy Deep 本地文件分区权限（user=FilesystemAllowedRoots 下；session=会话子目录下）。
type DeepFilesystemPolicy struct {
	ReadUser     bool `json:"readUser,optional,default=true"`
	WriteUser    bool `json:"writeUser,optional,default=false"`
	EditUser     bool `json:"editUser,optional,default=false"`
	ReadSession  bool `json:"readSession,optional,default=true"`
	WriteSession bool `json:"writeSession,optional,default=true"`
	EditSession  bool `json:"editSession,optional,default=true"`
}

// DeepLocalFilesystemEnabled 是否挂载 Deep 的本地 filesystem 工具（默认 true）。
func (a AgentConfig) DeepLocalFilesystemEnabled() bool {
	return !a.Deep.DisableLocalFilesystem
}

// CheckpointConfig Agent 中断/恢复的快照存储配置。
// 与 Memory/SessionStore 保持一致的 memory|jsonl|gormx 三后端模型。
type CheckpointConfig struct {
	Type    string `json:",default=memory,options=memory|jsonl|gormx"`
	BaseDir string `json:",optional"` // JSONL 存储目录
}

// MetricsConfig 监控配置
type MetricsConfig struct {
	Enabled bool `json:"enabled"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	APIKey      string  `json:"apiKey"`
	BaseURL     string  `json:"baseURL,omitempty"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
}

// MemoryConfig 记忆配置（消息存储）。
type MemoryConfig struct {
	Type    string `json:",default=memory,options=memory|jsonl|gormx"`
	BaseDir string `json:",optional"` // JSONL 存储目录
}

// SessionStoreConfig 会话存储配置。
type SessionStoreConfig struct {
	Type    string `json:",default=memory,options=memory|jsonl|gormx"`
	BaseDir string `json:",optional"` // JSONL 存储目录
}

// DBConfig gormx 数据库配置。
type DBConfig struct {
	Enabled    bool   `json:",optional,default=false"`
	DataSource string `json:",optional"`
	LogLevel   string `json:",optional,default=error"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	Enabled        bool          `json:"enabled"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"maxRetries"`
	MaxConcurrency int           `json:"maxConcurrency"`
}

// SkillsConfig Skills 配置（go-zero / aisolo.yaml）。
// 仅控制 skill 中间件使用的目录；Deep 文件系统见 Agent.Deep.disableLocalFilesystem。
type SkillsConfig struct {
	Dir     string `json:"dir,optional"`
	Enabled bool   `json:"enabled,optional,default=true"`
	// Strict 为 true 时：Enabled 且必须配置 dir，且启动前目录必须存在，否则 Validate 失败。
	Strict bool `json:"strict,optional,default=false"`
}

// LimitConfig 限流配置
type LimitConfig struct {
	MaxConcurrency int           `json:"maxConcurrency"` // 最大并发数
	RateLimit      int           `json:"rateLimit"`      // 每秒请求限制
	RequestTimeout time.Duration `json:"requestTimeout"` // 请求超时时间
}
