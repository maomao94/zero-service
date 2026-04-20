package config

import (
	"os"
	"strings"
	"time"

	einoxkb "zero-service/common/einox/knowledge"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config aisolo 服务配置
type Config struct {
	zrpc.RpcServerConf

	Model        ModelConfig        `json:"model"`
	Memory       MemoryConfig       `json:"memory"`
	SessionStore SessionStoreConfig `json:",optional"`
	Knowledge    einoxkb.Config     `json:"knowledge,optional"`
	// SessionRun 控制 RUNNING 租约（多实例 / 持久化会话时避免误清与健康实例冲突）。
	// YAML 键必须为 sessionRun（camelCase，与 json 标签一致）；勿写 SessionRun: 且仅注释子行，否则 conf 报 type mismatch。
	SessionRun SessionRunConfig `json:"sessionRun,optional"`
	DB         DBConfig         `json:",optional"`
	Tools      ToolsConfig      `json:"tools"`
	Skills     SkillsConfig     `json:"skills"`
	Agent      AgentConfig      `json:"agent"`
	Checkpoint CheckpointConfig `json:"checkpoint"`
	Metrics    MetricsConfig    `json:"metrics"`
	Limit      LimitConfig      `json:"limit"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	PoolMaxIdle int             `json:"poolMaxIdle"`
	PoolMaxLive time.Duration   `json:"poolMaxLive"`
	Deep        DeepAgentConfig `json:"deep,optional"`
	// PlanMaxIterations PlanExecute 模式最大迭代（默认 10，≤0 时按 10）。
	PlanMaxIterations int `json:"planMaxIterations,optional"`
	// DemoSurveyEcho 为 true 时在默认 Agent 模式挂载联调用 Survey→Echo 子 Agent（生产请关闭）。
	DemoSurveyEcho bool `json:"demoSurveyEcho,optional"`
}

// DeepAgentConfig Deep 模式专属（见 blueprint_deep）。
type DeepAgentConfig struct {
	// EnableLocalFilesystem 为 true 时 Deep 模式挂载 Eino 本地文件系统工具（grep 等）。默认 false（安全基线）。
	EnableLocalFilesystem bool `json:"enableLocalFilesystem,optional"`
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

// DeepLocalFilesystemEnabled 是否挂载 Deep 的本地 filesystem 工具（默认关闭）。
func (a AgentConfig) DeepLocalFilesystemEnabled() bool {
	return a.Deep.EnableLocalFilesystem
}

// EffectivePlanMaxIterations Plan 模式最大迭代次数。
func (a AgentConfig) EffectivePlanMaxIterations() int {
	if a.PlanMaxIterations > 0 {
		return a.PlanMaxIterations
	}
	return 10
}

// EffectiveSessionRunLeaseTTL RUNNING 租约时长；≤0 时默认 30m。
func (c Config) EffectiveSessionRunLeaseTTL() time.Duration {
	t := c.SessionRun.LeaseTTL
	if t <= 0 {
		return 30 * time.Minute
	}
	return t
}

// EffectiveNullLeaseRecoverGrace gormx/jsonl 启动恢复时，无租约 RUNNING 按 updated_at 判陈旧阈值。
func (c Config) EffectiveNullLeaseRecoverGrace() time.Duration {
	if c.SessionStore.NullLeaseRecoverGrace > 0 {
		return c.SessionStore.NullLeaseRecoverGrace
	}
	return 2 * time.Minute
}

// listenPortFromRpcListenOn 从当前 RpcServerConf.ListenOn（根 yaml 的 gRPC 监听，如 0.0.0.0:23002）解析端口。
func listenPortFromRpcListenOn(listenOn string) string {
	listenOn = strings.TrimSpace(listenOn)
	if listenOn == "" {
		return ""
	}
	if i := strings.LastIndexByte(listenOn, ':'); i >= 0 && i+1 < len(listenOn) {
		return listenOn[i+1:]
	}
	return ""
}

// EffectiveRunInstanceID 写入会话 run_owner：显式 sessionRun.instanceID 优先；否则 hostname:port，port 仅来自本进程 Rpc ListenOn（与根 yaml 一致，Docker 每实例一份配置即可区分）。
func (c Config) EffectiveRunInstanceID() string {
	id := strings.TrimSpace(c.SessionRun.InstanceID)
	if id != "" {
		return id
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	if p := listenPortFromRpcListenOn(c.ListenOn); p != "" {
		return host + ":" + p
	}
	return host
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
	// NullLeaseRecoverGrace：RUNNING 且无租约时，超过该时间未更新则可被启动恢复清为 IDLE（gormx/jsonl，默认 2m）。
	NullLeaseRecoverGrace time.Duration `json:"nullLeaseRecoverGrace,optional"`
}

// SessionRunConfig 一轮 Ask/Resume 持有 RUNNING 时的租约行为。
type SessionRunConfig struct {
	// LeaseTTL：进入 RUNNING 时写入 run_lease_until = now+LeaseTTL；默认 30m。≤0 时按 30m 处理。
	LeaseTTL time.Duration `json:"leaseTTL,optional"`
	// InstanceID 写入 run_owner；一般留空即可（默认主机名 + 根 yaml ListenOn 的 RPC 端口）。仅特殊场景再覆盖。
	InstanceID string `json:"instanceID,optional"`
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
