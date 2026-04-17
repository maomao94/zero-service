package agent

import (
	"context"
	"fmt"
	"os"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// EinoAgentType 标识本封装支持的 Eino Agent 实现类型。
//
// aisolo 的 Mode 会映射到其中某一个（或组合出一个由多个 Agent 组合成的 Workflow）。
type EinoAgentType string

const (
	// EinoTypeChatModel ReAct 风格, ChatModelAgent + tool calling。
	EinoTypeChatModel EinoAgentType = "chat_model"
	// EinoTypeDeep prebuilt/deep, 带 WriteTodos / FileSystem / 子 Agent 委派。
	EinoTypeDeep EinoAgentType = "deep"
	// EinoTypePlan prebuilt/planexecute, 计划-执行-重新计划循环。
	EinoTypePlan EinoAgentType = "plan"
	// EinoTypeSupervisor prebuilt/supervisor, 多 Agent 协作。
	EinoTypeSupervisor EinoAgentType = "supervisor"
	// EinoTypeSequential adk.NewSequentialAgent, 子 Agent 顺序执行。
	EinoTypeSequential EinoAgentType = "sequential"
	// EinoTypeParallel adk.NewParallelAgent, 子 Agent 并行执行。
	EinoTypeParallel EinoAgentType = "parallel"
	// EinoTypeLoop adk.NewLoopAgent, 子 Agent 循环执行到终止条件。
	EinoTypeLoop EinoAgentType = "loop"
)

// AgentTypeInfo Agent 类型元信息, 供 ListModes 等接口展示。
type AgentTypeInfo struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

var einoAgentTypeInfos = map[EinoAgentType]AgentTypeInfo{
	EinoTypeChatModel: {
		Type:         string(EinoTypeChatModel),
		Name:         "ChatModel Agent",
		Description:  "ReAct 工具调用 Agent, LLM 推理 → 工具调用 → 循环直到完成",
		Capabilities: []string{"工具调用", "多轮对话", "ReAct 推理"},
	},
	EinoTypeDeep: {
		Type:         string(EinoTypeDeep),
		Name:         "Deep Agent",
		Description:  "深度 Agent, 内置规划、文件系统、子 Agent 委派",
		Capabilities: []string{"任务规划", "文件操作", "深度思考", "子 Agent 委派"},
	},
	EinoTypePlan: {
		Type:         string(EinoTypePlan),
		Name:         "PlanExecute Agent",
		Description:  "计划-执行-重新计划循环 Agent，适合可分解的复杂任务",
		Capabilities: []string{"任务分解", "计划执行", "动态重规划", "工具调用"},
	},
	EinoTypeSupervisor: {
		Type:         string(EinoTypeSupervisor),
		Name:         "Supervisor Agent",
		Description:  "监督者多 Agent 协作，Supervisor 调度多个子 Agent",
		Capabilities: []string{"多 Agent 协作", "任务委派", "集中调度"},
	},
	EinoTypeSequential: {
		Type:         string(EinoTypeSequential),
		Name:         "Sequential Workflow",
		Description:  "Workflow Agent, 多个子 Agent 顺序执行, 上一个输出作为下一个输入",
		Capabilities: []string{"Workflow", "顺序编排"},
	},
	EinoTypeParallel: {
		Type:         string(EinoTypeParallel),
		Name:         "Parallel Workflow",
		Description:  "Workflow Agent, 多个子 Agent 并行执行, 最终汇总",
		Capabilities: []string{"Workflow", "并行编排"},
	},
	EinoTypeLoop: {
		Type:         string(EinoTypeLoop),
		Name:         "Loop Workflow",
		Description:  "Workflow Agent, 子 Agent 循环执行到最大迭代或 Exit",
		Capabilities: []string{"Workflow", "循环编排"},
	},
}

// LookupEinoAgentType 返回类型元信息, 不存在时返回 false。
func LookupEinoAgentType(t EinoAgentType) (AgentTypeInfo, bool) {
	info, ok := einoAgentTypeInfos[t]
	return info, ok
}

// NewAgent 是上层唯一的 Agent 构造入口 —— 按类型分发到具体的构造函数。
//
// Workflow 类（sequential / parallel / loop / supervisor）必须通过 WithSubAgents
// 提供子 Agent；ChatModel / Deep / Plan 则需要通过 WithModel 提供 ChatModel。
func NewAgent(ctx context.Context, t EinoAgentType, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	switch t {
	case EinoTypeChatModel:
		if cfg.model == nil {
			return nil, fmt.Errorf("agent: chat_model requires WithModel")
		}
		m, ok := cfg.model.(model.BaseChatModel)
		if !ok {
			return nil, fmt.Errorf("agent: chat_model requires model.BaseChatModel")
		}
		return NewChatModelAgent(ctx, m, opts...)
	case EinoTypeDeep:
		if cfg.model == nil {
			return nil, fmt.Errorf("agent: deep requires WithModel")
		}
		m, ok := cfg.model.(model.BaseChatModel)
		if !ok {
			return nil, fmt.Errorf("agent: deep requires model.BaseChatModel")
		}
		return NewDeepAgent(ctx, m, opts...)
	case EinoTypePlan:
		if cfg.model == nil {
			return nil, fmt.Errorf("agent: plan requires WithModel")
		}
		m, ok := cfg.model.(model.BaseChatModel)
		if !ok {
			return nil, fmt.Errorf("agent: plan requires model.BaseChatModel")
		}
		return NewPlanExecuteAgent(ctx, m, opts...)
	case EinoTypeSupervisor:
		if cfg.model == nil {
			return nil, fmt.Errorf("agent: supervisor requires WithModel")
		}
		m, ok := cfg.model.(model.BaseChatModel)
		if !ok {
			return nil, fmt.Errorf("agent: supervisor requires model.BaseChatModel")
		}
		return NewSupervisorAgent(ctx, m, cfg.subAgents, opts...)
	case EinoTypeSequential:
		return NewSequentialAgent(ctx, opts...)
	case EinoTypeParallel:
		return NewParallelAgent(ctx, opts...)
	case EinoTypeLoop:
		return NewLoopAgent(ctx, opts...)
	default:
		return nil, fmt.Errorf("agent: unknown type %q", string(t))
	}
}

// NewChatModelAgent 创建 ReAct 工具调用 Agent。
func NewChatModelAgent(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	return New(ctx, append(opts, WithModel(chatModel))...)
}

// NewDeepAgent 创建 Deep Agent (预构建, 带 WriteTodos、文件系统、子 Agent 委派)。
func NewDeepAgent(ctx context.Context, chatModel model.BaseChatModel, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	cfg.model = chatModel

	maxIter := cfg.maxIter
	if maxIter <= 0 {
		maxIter = 30
	}

	skillHandlers := buildSkillHandlers(ctx, cfg)
	cfg.handlers = append(skillHandlers, cfg.handlers...)

	var backend filesystem.Backend
	if cfg.enableFileSystem || cfg.skillsDir != "" || os.Getenv("EINO_EXT_SKILLS_DIR") != "" {
		backend, _ = localbk.NewBackend(ctx, &localbk.Config{})
	}

	deepAgent, err := deep.New(ctx, &deep.Config{
		Name:              cfg.name,
		Description:       cfg.description,
		ChatModel:         chatModel,
		Instruction:       cfg.instruction,
		MaxIteration:      maxIter,
		WithoutWriteTodos: !cfg.enableWriteTodos,
		Backend:           backend,
		Handlers:          cfg.handlers,
	})
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

// NewSequentialAgent 创建 adk Sequential Workflow Agent。
//
// 子 Agent 通过 WithSubAgents 传入, 至少一个。
func NewSequentialAgent(ctx context.Context, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	if len(cfg.subAgents) == 0 {
		return nil, fmt.Errorf("agent: sequential requires at least one sub agent")
	}

	seqAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        cfg.name,
		Description: cfg.description,
		SubAgents:   cfg.subAgents,
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
func NewParallelAgent(ctx context.Context, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	if len(cfg.subAgents) == 0 {
		return nil, fmt.Errorf("agent: parallel requires at least one sub agent")
	}

	parAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        cfg.name,
		Description: cfg.description,
		SubAgents:   cfg.subAgents,
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
func NewLoopAgent(ctx context.Context, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	if len(cfg.subAgents) == 0 {
		return nil, fmt.Errorf("agent: loop requires at least one sub agent")
	}

	loopAgent, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          cfg.name,
		Description:   cfg.description,
		SubAgents:     cfg.subAgents,
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
