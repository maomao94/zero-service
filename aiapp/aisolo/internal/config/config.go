package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

// Config aisolo 服务配置
type Config struct {
	zrpc.RpcServerConf

	// 模型配置
	DefaultModel string           `json:",optional"`
	Models       []ModelConfig    `json:",optional"`
	Providers    []ProviderConfig `json:",optional"`

	// Agent 配置
	Agents AgentsConfig `json:",optional"`

	// 智能路由配置
	Router RouterConfig `json:",optional"`

	// 记忆系统配置
	Memory MemoryConfig `json:",optional"`

	// Embedding 配置
	Embedding EmbeddingConfig `json:",optional"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Id                  string `json:"id"`
	Provider            string `json:"provider"`
	BackendModel        string `json:"backendModel"`
	DisplayName         string `json:"displayName"`
	Description         string `json:"description"`
	MaxTokens           int    `json:"maxTokens,default=4096"`
	SupportsStreaming   bool   `json:"supportsStreaming,default=true"`
	SupportsToolCalling bool   `json:"supportsToolCalling,default=true"`
	SupportsThinking    bool   `json:"supportsThinking,default=false"`
}

// ProviderConfig Provider 配置
type ProviderConfig struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Endpoint string `json:"endpoint"`
	ApiKey   string `json:"apiKey"`
}

// AgentsConfig Agent 配置集合
type AgentsConfig struct {
	ChatModel   ChatModelAgentConfig   `json:"chatModel"`
	Sequential  SequentialAgentConfig  `json:"sequential"`
	Loop        LoopAgentConfig        `json:"loop"`
	Parallel    ParallelAgentConfig    `json:"parallel"`
	Supervisor  SupervisorAgentConfig  `json:"supervisor"`
	PlanExecute PlanExecuteAgentConfig `json:"planExecute"`
	Deep        DeepAgentConfig        `json:"deep"`
	Multi       MultiAgentConfig       `json:"multi"`
}

// SequentialAgentConfig SequentialAgent 配置
type SequentialAgentConfig struct {
	Name          string   `json:"name,default=SequentialAgent"`
	Description   string   `json:"description,default=顺序执行 Agent"`
	Instruction   string   `json:"instruction"`
	SubAgentNames []string `json:"subAgentNames"`
}

// LoopAgentConfig LoopAgent 配置
type LoopAgentConfig struct {
	Name          string   `json:"name,default=LoopAgent"`
	Description   string   `json:"description,default=循环执行 Agent"`
	Instruction   string   `json:"instruction"`
	MaxIterations int      `json:"maxIterations,default=10"`
	SubAgentNames []string `json:"subAgentNames"`
}

// ParallelAgentConfig ParallelAgent 配置
type ParallelAgentConfig struct {
	Name          string   `json:"name,default=ParallelAgent"`
	Description   string   `json:"description,default=并行执行 Agent"`
	Instruction   string   `json:"instruction"`
	SubAgentNames []string `json:"subAgentNames"`
}

// SupervisorAgentConfig SupervisorAgent 配置
type SupervisorAgentConfig struct {
	Name          string   `json:"name,default=SupervisorAgent"`
	Description   string   `json:"description,default=监督者 Agent"`
	Instruction   string   `json:"instruction"`
	MaxTokens     int      `json:"maxTokens,default=4096"`
	SubAgentNames []string `json:"subAgentNames"`
}

// PlanExecuteAgentConfig PlanExecuteAgent 配置
type PlanExecuteAgentConfig struct {
	Name                 string `json:"name,default=PlanExecuteAgent"`
	Description          string `json:"description,default=规划-执行 Agent"`
	PlannerInstruction   string `json:"plannerInstruction"`
	ExecutorInstruction  string `json:"executorInstruction"`
	ReplannerInstruction string `json:"replannerInstruction"`
	MaxIterations        int    `json:"maxIterations,default=10"`
}

// ChatModelAgentConfig ChatModelAgent 配置
type ChatModelAgentConfig struct {
	Name          string `json:"name,default=ChatModelAgent"`
	Description   string `json:"description,default=ReAct 工具调用 Agent"`
	Instruction   string `json:"instruction"`
	MaxIterations int    `json:"maxIterations,default=20"`
}

// DeepAgentConfig DeepAgent 配置
type DeepAgentConfig struct {
	Name              string           `json:"name,default=DeepAgent"`
	Description       string           `json:"description,default=深度规划 Agent"`
	Instruction       string           `json:"instruction"`
	MaxIterations     int              `json:"maxIterations,default=30"`
	EnableFileSystem  bool             `json:"enableFileSystem,default=true"`
	FileSystemBackend string           `json:"fileSystemBackend,default=memory"`
	EnableSubAgents   bool             `json:"enableSubAgents,default=true"`
	SubAgents         []SubAgentConfig `json:"subAgents"`
}

// SubAgentConfig 子 Agent 配置
type SubAgentConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MultiAgentConfig MultiAgent 配置
type MultiAgentConfig struct {
	Name                   string              `json:"name,default=MultiAgent"`
	Description            string              `json:"description,default=多 Agent 协作系统"`
	CoordinatorInstruction string              `json:"coordinatorInstruction"`
	ExpertAgents           []ExpertAgentConfig `json:"expertAgents"`
}

// ExpertAgentConfig 专家 Agent 配置
type ExpertAgentConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Instruction string `json:"instruction"`
}

// RouterConfig 智能路由配置
type RouterConfig struct {
	Strategy        string `json:"strategy,default=auto"`
	SimpleThreshold int    `json:"simpleThreshold,default=50"`
	LLMRouterModel  string `json:"llmRouterModel"`
	LLMRouterPrompt string `json:"llmRouterPrompt"`
}

// MemoryConfig 记忆系统配置
type MemoryConfig struct {
	EnableUserMemories   bool           `json:"enableUserMemories,default=true"`
	EnableSessionSummary bool           `json:"enableSessionSummary,default=true"`
	Retrieval            string         `json:"retrieval,default=last_n"`
	MemoryLimit          int            `json:"memoryLimit,default=20"`
	AsyncWorkerPoolSize  int            `json:"asyncWorkerPoolSize,default=5"`
	SummaryTrigger       SummaryTrigger `json:"summaryTrigger"`
	Cleanup              CleanupConfig  `json:"cleanup"`
}

// SummaryTrigger 摘要触发配置
type SummaryTrigger struct {
	Strategy         string `json:"strategy,default=smart"`
	MessageThreshold int    `json:"messageThreshold,default=10"`
	MinInterval      int    `json:"minInterval,default=600"`
}

// CleanupConfig 清理配置
type CleanupConfig struct {
	SessionCleanupInterval int `json:"sessionCleanupInterval,default=24"`
	SessionRetentionTime   int `json:"sessionRetentionTime,default=168"`
	MessageHistoryLimit    int `json:"messageHistoryLimit,default=1000"`
	CleanupInterval        int `json:"cleanupInterval,default=12"`
}

// EmbeddingConfig Embedding 配置
type EmbeddingConfig struct {
	Provider string `json:"provider,optional"`
	Model    string `json:"model,optional"`
	ApiKey   string `json:"apiKey,optional"`
	Region   string `json:"region,optional"`
}
