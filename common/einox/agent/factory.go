package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"

	"zero-service/common/einox/fsrestrict"

	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// NewChatModelAgent 创建 ReAct 工具调用 Agent。
func NewChatModelAgent(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	return New(ctx, append(opts, WithModel(chatModel))...)
}

// NewDeepAgent 创建 Deep Agent (预构建: WriteTodos、可选 FileSystem、WithTools 合并进 ToolsConfig、WithSubAgents 写入 SubAgents)。
// Eino 会为子 Agent 注册 task 编排工具；主模型根据对话上下文决定何时委派。
func NewDeepAgent(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	cfg.model = chatModel

	maxIter := cfg.maxIter
	if maxIter <= 0 {
		maxIter = 30
	}

	skillHandlers := buildSkillHandlers(ctx, cfg)
	cfg.handlers = append(skillHandlers, cfg.handlers...)

	// 仅当启用 Deep 本地文件系统时挂载 Backend（grep 等工具依赖本机 ripgrep）。
	// Skill 中间件在 buildSkillHandlers 内自行创建 local backend，不因 skills 目录而向 Deep 重复挂载。
	var backend filesystem.Backend
	if cfg.enableFileSystem {
		inner, err := localbk.NewBackend(ctx, &localbk.Config{})
		if err != nil {
			return nil, fmt.Errorf("agent: local filesystem backend: %w", err)
		}
		backend = fsrestrict.WrapConfigured(inner, cfg.deepFS)
	}

	deepCfg := &deep.Config{
		Name:              cfg.name,
		Description:       cfg.description,
		ChatModel:         chatModel,
		Instruction:       cfg.instruction,
		MaxIteration:      maxIter,
		WithoutWriteTodos: !cfg.enableWriteTodos,
		Backend:           backend,
		Handlers:          cfg.handlers,
		SubAgents:         cfg.subAgents,
	}
	if len(cfg.tools) > 0 {
		deepCfg.ToolsConfig = adk.ToolsConfig{ToolsNodeConfig: toolsNodeConfig(cfg)}
	}
	if len(cfg.middlewares) > 0 {
		deepCfg.Middlewares = cfg.middlewares
	}

	deepAgent, err := deep.New(ctx, deepCfg)
	if err != nil {
		return nil, err
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           deepAgent,
		EnableStreaming: true,
		CheckPointStore: cfg.checkpointStore,
	})

	return &Agent{
		name:     cfg.name,
		adkAgent: deepAgent,
		runner:   runner,
		opts:     *cfg,
	}, nil
}

// NewPlanExecuteAgent 创建 prebuilt/planexecute 计划-执行 Agent。
func NewPlanExecuteAgent(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	cfg.model = chatModel

	tc, ok := chatModel.(model.ToolCallingChatModel)
	if !ok {
		return nil, fmt.Errorf("agent: planexecute requires model.ToolCallingChatModel")
	}

	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: tc,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new planner: %w", err)
	}

	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: toolsNodeConfig(cfg),
		},
		MaxIterations: cfg.maxIter,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new executor: %w", err)
	}

	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: tc,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new replanner: %w", err)
	}

	planAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: cfg.maxIter,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new plan-execute: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           planAgent,
		EnableStreaming: true,
		CheckPointStore: cfg.checkpointStore,
	})

	return &Agent{
		name:     cfg.name,
		adkAgent: planAgent,
		runner:   runner,
		opts:     *cfg,
	}, nil
}

// NewSupervisorAgent 创建 prebuilt/supervisor 多 Agent 协作。
func NewSupervisorAgent(ctx context.Context, chatModel model.BaseChatModel, subAgents []adk.Agent, opts ...Option) (*Agent, error) {
	if len(subAgents) == 0 {
		return nil, fmt.Errorf("agent: supervisor requires at least one sub agent")
	}

	cfg := newOptions(opts...)
	cfg.model = chatModel

	supAgent, err := buildChatModelAgent(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("agent: build supervisor inner: %w", err)
	}

	container, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: supAgent,
		SubAgents:  subAgents,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new supervisor: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           container,
		EnableStreaming: true,
		CheckPointStore: cfg.checkpointStore,
	})

	return &Agent{
		name:     cfg.name,
		adkAgent: container,
		runner:   runner,
		opts:     *cfg,
	}, nil
}

// needsWorkflowCoordinator 为 true 时，Workflow 需挂 ChatModel 才能挂载 tools / skills / handlers / middlewares。
func needsWorkflowCoordinator(cfg *options) bool {
	if len(cfg.tools) > 0 || len(cfg.handlers) > 0 || len(cfg.middlewares) > 0 {
		return true
	}
	d := strings.TrimSpace(cfg.skillsDir)
	if d == "" {
		d = strings.TrimSpace(os.Getenv("EINO_EXT_SKILLS_DIR"))
	}
	return d != ""
}

// coordinatorOptionsClone 复制 options 供独立 ChatModel 子 Agent 使用，避免 buildChatModelAgent 改写原 cfg 的 handlers。
func coordinatorOptionsClone(cfg *options) *options {
	cc := *cfg
	cc.subAgents = nil
	cc.tools = append([]tool.BaseTool(nil), cfg.tools...)
	cc.handlers = append([]adk.ChatModelAgentMiddleware(nil), cfg.handlers...)
	cc.middlewares = append([]adk.AgentMiddleware(nil), cfg.middlewares...)
	if strings.TrimSpace(cc.name) != "" {
		cc.name = cc.name + "/coordinator"
	} else {
		cc.name = "workflow-coordinator"
	}
	if strings.TrimSpace(cc.description) == "" {
		cc.description = "Sub-agent with tools/skills; runs as first step of the workflow"
	}
	return &cc
}

// workflowSubAgents 合并 WithSubAgents 与可选的协调子 Agent（ADK Workflow 本身无 Tools 字段）。
func workflowSubAgents(ctx context.Context, cfg *options) ([]adk.Agent, error) {
	base := append([]adk.Agent(nil), cfg.subAgents...)
	if !needsWorkflowCoordinator(cfg) {
		return base, nil
	}
	if cfg.model == nil {
		return nil, fmt.Errorf("agent: workflow with tools/skills/handlers/middlewares requires WithModel (coordinator ChatModel)")
	}
	cc := coordinatorOptionsClone(cfg)
	coord, err := buildChatModelAgent(ctx, cc)
	if err != nil {
		return nil, fmt.Errorf("agent: workflow coordinator: %w", err)
	}
	return append([]adk.Agent{coord}, base...), nil
}

// NewSequentialAgent 创建 adk Sequential Workflow Agent。
//
// 子 Agent 通过 WithSubAgents 传入，至少一个（若配置了 tools/skills/handlers/middlewares 且提供 WithModel，
// 会自动前置一个带工具与 skill 的 ChatModel 协调子 Agent）。
func NewSequentialAgent(ctx context.Context, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	subs, err := workflowSubAgents(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, fmt.Errorf("agent: sequential requires at least one sub agent")
	}

	seqAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        cfg.name,
		Description: cfg.description,
		SubAgents:   subs,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new sequential: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           seqAgent,
		EnableStreaming: true,
		CheckPointStore: cfg.checkpointStore,
	})

	return &Agent{
		name:     cfg.name,
		adkAgent: seqAgent,
		runner:   runner,
		opts:     *cfg,
	}, nil
}

// NewParallelAgent 创建 adk Parallel Workflow Agent。
//
// 若配置了 tools/skills/handlers/middlewares 且提供 WithModel，会前置一个协调子 Agent（与其它子 Agent 并行）。
func NewParallelAgent(ctx context.Context, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	subs, err := workflowSubAgents(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, fmt.Errorf("agent: parallel requires at least one sub agent")
	}

	parAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        cfg.name,
		Description: cfg.description,
		SubAgents:   subs,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new parallel: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           parAgent,
		EnableStreaming: true,
		CheckPointStore: cfg.checkpointStore,
	})

	return &Agent{
		name:     cfg.name,
		adkAgent: parAgent,
		runner:   runner,
		opts:     *cfg,
	}, nil
}

// NewLoopAgent 创建 adk Loop Workflow Agent。
//
// 最大迭代次数通过 WithMaxIterations 设置, 不设置则由 adk 默认值决定。
// 若配置了 tools/skills/handlers/middlewares 且提供 WithModel，会前置协调子 Agent（参与每轮循环中的子 Agent 序列）。
func NewLoopAgent(ctx context.Context, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	subs, err := workflowSubAgents(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, fmt.Errorf("agent: loop requires at least one sub agent")
	}

	loopAgent, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          cfg.name,
		Description:   cfg.description,
		SubAgents:     subs,
		MaxIterations: cfg.maxIter,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new loop: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           loopAgent,
		EnableStreaming: true,
		CheckPointStore: cfg.checkpointStore,
	})

	return &Agent{
		name:     cfg.name,
		adkAgent: loopAgent,
		runner:   runner,
		opts:     *cfg,
	}, nil
}

// toolsNodeConfig 把 options.tools 转成 compose.ToolsNodeConfig。
func toolsNodeConfig(cfg *options) compose.ToolsNodeConfig {
	if len(cfg.tools) == 0 {
		return compose.ToolsNodeConfig{}
	}
	return compose.ToolsNodeConfig{Tools: cfg.tools}
}
