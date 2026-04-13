package agent

import (
	"context"
	"errors"
	"os"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/model"
)

// =============================================================================
// Eino Agent 类型常量（对应 eino adk 包的 Agent 类型）
// =============================================================================

// EinoAgentType eino 框架提供的 Agent 类型
type EinoAgentType string

const (
	EinoTypeChatModel   EinoAgentType = "chat_model"   // ReAct 工具调用
	EinoTypeSequential  EinoAgentType = "sequential"   // 顺序执行
	EinoTypeLoop        EinoAgentType = "loop"         // 循环执行
	EinoTypeParallel    EinoAgentType = "parallel"     // 并行执行
	EinoTypeSupervisor  EinoAgentType = "supervisor"   // 监督者
	EinoTypePlanExecute EinoAgentType = "plan_execute" // 规划-执行
	EinoTypeDeep        EinoAgentType = "deep"         // 深度 Agent
)

// =============================================================================
// 错误定义
// =============================================================================

var (
	ErrNoSubAgents       = errors.New("at least one sub agent is required")
	ErrInvalidSubAgent   = errors.New("invalid sub agent")
	ErrInvalidSupervisor = errors.New("invalid supervisor agent")
	ErrInvalidPlanner    = errors.New("invalid planner agent")
	ErrInvalidExecutor   = errors.New("invalid executor agent")
	ErrInvalidReplanner  = errors.New("invalid replanner agent")
)

// =============================================================================
// 基础 Agent 工厂
// =============================================================================

// NewChatModelAgent 创建 ReAct 工具调用 Agent
// 最基础的 Agent 类型，支持 LLM 推理和工具调用
func NewChatModelAgent(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	return New(ctx, append(opts, WithModel(chatModel))...)
}

// NewDeepAgent 创建深度 Agent
// 预构建 Agent，内置 WriteTodos、文件系统和子 Agent 委派
func NewDeepAgent(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	return newDeepAgentImpl(ctx, chatModel, opts...)
}

// =============================================================================
// 组合型 Agent 工厂 - 接收 Agent 实例
// =============================================================================

// NewSequentialAgent 创建顺序执行 Agent
// subAgents 按顺序执行，前一个的输出作为后一个的输入
func NewSequentialAgent(ctx context.Context, subAgents []*Agent, opts ...Option) (*Agent, error) {
	return newSequentialAgentImpl(ctx, subAgents, opts...)
}

// NewLoopAgent 创建循环执行 Agent
// subAgents 循环执行直到达到 maxIterations 或产生 ExitAction
func NewLoopAgent(ctx context.Context, subAgents []*Agent, maxIterations int, opts ...Option) (*Agent, error) {
	return newLoopAgentImpl(ctx, subAgents, maxIterations, opts...)
}

// NewParallelAgent 创建并行执行 Agent
// subAgents 并发执行，结果合并
func NewParallelAgent(ctx context.Context, subAgents []*Agent, opts ...Option) (*Agent, error) {
	return newParallelAgentImpl(ctx, subAgents, opts...)
}

// NewSupervisorAgent 创建监督者 Agent
// supervisor 协调 subAgents，动态分配任务
func NewSupervisorAgent(ctx context.Context, supervisorAgent *Agent, subAgents []*Agent, opts ...Option) (*Agent, error) {
	return newSupervisorAgentImpl(ctx, supervisorAgent, subAgents, opts...)
}

// NewPlanExecuteAgent 创建规划-执行 Agent
// planner 规划 -> executor 执行 -> replanner 重规划，循环直到完成
func NewPlanExecuteAgent(ctx context.Context, planner, executor, replanner *Agent, maxIterations int, opts ...Option) (*Agent, error) {
	return newPlanExecuteAgentImpl(ctx, planner, executor, replanner, maxIterations, opts...)
}

// =============================================================================
// 内部实现
// =============================================================================

// newSequentialAgentImpl 顺序执行 Agent 实现
func newSequentialAgentImpl(ctx context.Context, subAgents []*Agent, opts ...Option) (*Agent, error) {
	if len(subAgents) == 0 {
		return nil, ErrNoSubAgents
	}

	adkAgents := extractADKAgents(subAgents)
	if adkAgents == nil {
		return nil, ErrInvalidSubAgent
	}

	agent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        getName(opts),
		Description: getDescription(opts),
		SubAgents:   adkAgents,
	})
	if err != nil {
		return nil, err
	}

	return wrapToAgent(ctx, agent, opts)
}

// newLoopAgentImpl 循环执行 Agent 实现
func newLoopAgentImpl(ctx context.Context, subAgents []*Agent, maxIterations int, opts ...Option) (*Agent, error) {
	if len(subAgents) == 0 {
		return nil, ErrNoSubAgents
	}

	if maxIterations <= 0 {
		maxIterations = 10
	}

	adkAgents := extractADKAgents(subAgents)
	if adkAgents == nil {
		return nil, ErrInvalidSubAgent
	}

	agent, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          getName(opts),
		Description:   getDescription(opts),
		SubAgents:     adkAgents,
		MaxIterations: maxIterations,
	})
	if err != nil {
		return nil, err
	}

	return wrapToAgent(ctx, agent, opts)
}

// newParallelAgentImpl 并行执行 Agent 实现
func newParallelAgentImpl(ctx context.Context, subAgents []*Agent, opts ...Option) (*Agent, error) {
	if len(subAgents) == 0 {
		return nil, ErrNoSubAgents
	}

	adkAgents := extractADKAgents(subAgents)
	if adkAgents == nil {
		return nil, ErrInvalidSubAgent
	}

	agent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        getName(opts),
		Description: getDescription(opts),
		SubAgents:   adkAgents,
	})
	if err != nil {
		return nil, err
	}

	return wrapToAgent(ctx, agent, opts)
}

// newSupervisorAgentImpl 监督者 Agent 实现
func newSupervisorAgentImpl(ctx context.Context, supervisorAgent *Agent, subAgents []*Agent, opts ...Option) (*Agent, error) {
	if supervisorAgent == nil || supervisorAgent.adkAgent == nil {
		return nil, ErrInvalidSupervisor
	}
	if len(subAgents) == 0 {
		return nil, ErrNoSubAgents
	}

	adkSubAgents := extractADKAgents(subAgents)
	if adkSubAgents == nil {
		return nil, ErrInvalidSubAgent
	}

	agent, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: supervisorAgent.adkAgent,
		SubAgents:  adkSubAgents,
	})
	if err != nil {
		return nil, err
	}

	return wrapToAgent(ctx, agent, opts)
}

// newPlanExecuteAgentImpl 规划-执行 Agent 实现
func newPlanExecuteAgentImpl(ctx context.Context, planner, executor, replanner *Agent, maxIterations int, opts ...Option) (*Agent, error) {
	if planner == nil || planner.adkAgent == nil {
		return nil, ErrInvalidPlanner
	}
	if executor == nil || executor.adkAgent == nil {
		return nil, ErrInvalidExecutor
	}
	if replanner == nil || replanner.adkAgent == nil {
		return nil, ErrInvalidReplanner
	}

	if maxIterations <= 0 {
		maxIterations = 10
	}

	agent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner.adkAgent,
		Executor:      executor.adkAgent,
		Replanner:     replanner.adkAgent,
		MaxIterations: maxIterations,
	})
	if err != nil {
		return nil, err
	}

	return wrapToAgent(ctx, agent, opts)
}

// newDeepAgentImpl 深度 Agent 实现
func newDeepAgentImpl(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	maxIter := getMaxIterations(opts)
	if maxIter <= 0 {
		maxIter = 30
	}

	// 收集 handlers
	var handlers []adk.ChatModelAgentMiddleware

	// 确定 backend（文件系统或 skill 需要 backend）
	var backend filesystem.Backend
	if getEnableFileSystem(opts) || getSkillsDir(opts) != "" {
		// 使用 local backend 支持文件系统和 skill
		backend, _ = localbk.NewBackend(ctx, &localbk.Config{})
	}

	// 配置 skill 中间件
	skillsDir := getSkillsDir(opts)
	if skillsDir == "" {
		// 尝试从环境变量获取
		skillsDir = os.Getenv("EINO_EXT_SKILLS_DIR")
	}

	if skillsDir != "" {
		// 验证目录存在
		if fi, err := os.Stat(skillsDir); err == nil && fi.IsDir() {
			skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
				Backend: backend,
				BaseDir: skillsDir,
			})
			if err == nil {
				skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
					Backend: skillBackend,
				})
				if err == nil {
					handlers = append(handlers, skillMiddleware)
				}
			}
		}
	}

	// 添加自定义 handlers
	handlers = append(handlers, getHandlers(opts)...)

	deepCfg := &deep.Config{
		Name:              getName(opts),
		Description:       getDescription(opts),
		ChatModel:         chatModel,
		Instruction:       getInstruction(opts),
		MaxIteration:      maxIter,
		WithoutWriteTodos: !getEnableWriteTodos(opts),
		Backend:           backend,
		Handlers:          handlers,
	}

	agent, err := deep.New(ctx, deepCfg)
	if err != nil {
		return nil, err
	}

	return &Agent{
		name:     getName(opts),
		adkAgent: agent,
		runner:   adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true}),
	}, nil
}

// getHandlers 获取自定义 handlers
func getHandlers(opts []Option) []adk.ChatModelAgentMiddleware {
	var handlers []adk.ChatModelAgentMiddleware
	for _, opt := range opts {
		var cfg options
		opt(&cfg)
		handlers = append(handlers, cfg.handlers...)
	}
	return handlers
}

// =============================================================================
// 辅助函数
// =============================================================================

// extractADKAgents 从 Agent 切片中提取 adk.Agent
func extractADKAgents(agents []*Agent) []adk.Agent {
	if len(agents) == 0 {
		return nil
	}
	result := make([]adk.Agent, len(agents))
	for i, a := range agents {
		if a == nil || a.adkAgent == nil {
			return nil
		}
		result[i] = a.adkAgent
	}
	return result
}

// wrapToAgent 将 adk.Agent 包装为 einox.Agent
func wrapToAgent(ctx context.Context, adkAgent adk.Agent, opts []Option) (*Agent, error) {
	return &Agent{
		name:     getName(opts),
		adkAgent: adkAgent,
		runner:   adk.NewRunner(ctx, adk.RunnerConfig{Agent: adkAgent, EnableStreaming: true}),
	}, nil
}

// getName 获取名称
func getName(opts []Option) string {
	var name string
	for _, opt := range opts {
		opt(&options{name: name})
	}
	return name
}

// getDescription 获取描述
func getDescription(opts []Option) string {
	var desc string
	for _, opt := range opts {
		opt(&options{description: desc})
	}
	return desc
}

// getInstruction 获取指令
func getInstruction(opts []Option) string {
	var inst string
	for _, opt := range opts {
		opt(&options{instruction: inst})
	}
	return inst
}

// getMaxIterations 获取最大迭代次数
func getMaxIterations(opts []Option) int {
	var maxIter int
	for _, opt := range opts {
		opt(&options{maxIter: maxIter})
	}
	return maxIter
}

// getEnableWriteTodos 获取是否启用 WriteTodos
func getEnableWriteTodos(opts []Option) bool {
	var enable bool
	for _, opt := range opts {
		opt(&options{enableWriteTodos: enable})
	}
	return enable
}

// getEnableFileSystem 获取是否启用文件系统
func getEnableFileSystem(opts []Option) bool {
	var enable bool
	for _, opt := range opts {
		opt(&options{enableFileSystem: enable})
	}
	return enable
}

// getSkillsDir 获取 Skills 目录
func getSkillsDir(opts []Option) string {
	var dir string
	for _, opt := range opts {
		opt(&options{skillsDir: dir})
	}
	return dir
}

// =============================================================================
// Extended Option - 支持更多组合型 Agent 配置
// =============================================================================

// WithEnableWriteTodos 启用 WriteTodos（Deep Agent）
func WithEnableWriteTodos(enable bool) Option {
	return func(o *options) {
		o.enableWriteTodos = enable
	}
}

// WithEnableFileSystem 启用文件系统（Deep Agent）
func WithEnableFileSystem(enable bool) Option {
	return func(o *options) {
		o.enableFileSystem = enable
	}
}

// WithMaxIterations 设置最大迭代次数
func WithMaxIterations(max int) Option {
	return func(o *options) {
		o.maxIter = max
	}
}

// WithSkillsDir 设置 Skills 目录（Deep Agent）
//
// 配置后会自动加载该目录下的 SKILL.md 文件，
// Agent 可以通过 skill 工具调用这些技能。
func WithSkillsDir(dir string) Option {
	return func(o *options) {
		o.skillsDir = dir
	}
}

// =============================================================================
// Agent 类型信息
// =============================================================================

// GetEinoAgentTypeInfo 获取 Agent 类型信息
func GetEinoAgentTypeInfo(agentType EinoAgentType) *AgentTypeInfo {
	infos := map[EinoAgentType]*AgentTypeInfo{
		EinoTypeChatModel: {
			Type:         string(EinoTypeChatModel),
			Name:         "ChatModel Agent",
			Description:  "ReAct 工具调用 Agent，LLM 推理 -> 工具调用 -> 循环直到完成",
			Capabilities: []string{"工具调用", "多轮对话", "ReAct 推理"},
		},
		EinoTypeSequential: {
			Type:         string(EinoTypeSequential),
			Name:         "Sequential Agent",
			Description:  "顺序执行 Agent，按顺序依次执行子 Agent",
			Capabilities: []string{"顺序执行", "流水线处理", "多阶段任务"},
		},
		EinoTypeLoop: {
			Type:         string(EinoTypeLoop),
			Name:         "Loop Agent",
			Description:  "循环执行 Agent，循环执行直到条件满足或达到最大迭代次数",
			Capabilities: []string{"循环执行", "迭代优化", "条件退出"},
		},
		EinoTypeParallel: {
			Type:         string(EinoTypeParallel),
			Name:         "Parallel Agent",
			Description:  "并行执行 Agent，并发执行多个子 Agent",
			Capabilities: []string{"并行执行", "并发处理", "多任务同时处理", "结果合并"},
		},
		EinoTypeSupervisor: {
			Type:         string(EinoTypeSupervisor),
			Name:         "Supervisor Agent",
			Description:  "监督者 Agent，协调多个子 Agent 动态分配任务",
			Capabilities: []string{"动态任务分配", "多 Agent 协调", "监督执行"},
		},
		EinoTypePlanExecute: {
			Type:         string(EinoTypePlanExecute),
			Name:         "Plan-Execute Agent",
			Description:  "规划-执行 Agent，Planner -> Executor -> Replanner 循环直到完成",
			Capabilities: []string{"任务规划", "分步执行", "动态调整", "进度评估"},
		},
		EinoTypeDeep: {
			Type:         string(EinoTypeDeep),
			Name:         "Deep Agent",
			Description:  "深度 Agent，内置规划、文件系统、子 Agent 委派",
			Capabilities: []string{"任务规划", "文件操作", "深度思考", "子Agent委派"},
		},
	}

	if info, ok := infos[agentType]; ok {
		return info
	}

	return &AgentTypeInfo{
		Type:        string(agentType),
		Name:        "Unknown Agent",
		Description: "未知 Agent 类型",
	}
}

// AgentTypeInfo Agent 类型信息
type AgentTypeInfo struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}
