package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// Agent 类型定义（基于 Eino ADK 官方文档）
// =============================================================================

// AgentType Agent 类型
// 参考: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/
type AgentType = string

const (
	// AgentTypeChatModel ReAct 工具调用 Agent
	// 使用 adk.NewChatModelAgent 创建
	// 特点：LLM 推理 -> 工具调用 -> 循环直到完成
	// 文档: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/
	AgentTypeChatModel AgentType = "chat_model"

	// AgentTypeSequential 顺序执行 Agent (Workflow Agent)
	// 使用 adk.NewSequentialAgent 创建
	// 特点：按顺序依次执行子 Agent，前一个的输出作为后一个的输入
	// 文档: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/workflow/
	AgentTypeSequential AgentType = "sequential"

	// AgentTypeLoop 循环执行 Agent (Workflow Agent)
	// 使用 adk.NewLoopAgent 创建
	// 特点：循环执行子 Agent 序列，直到达到最大迭代次数或产生 ExitAction
	AgentTypeLoop AgentType = "loop"

	// AgentTypeParallel 并行执行 Agent (Workflow Agent)
	// 使用 adk.NewParallelAgent 创建
	// 特点：并发执行多个子 Agent
	AgentTypeParallel AgentType = "parallel"

	// AgentTypeSupervisor 监督者 Agent
	// 使用 adk/prebuilt/supervisor.New 创建
	// 特点：监督者协调多个子 Agent，动态分配任务，子 Agent 完成后自动回调
	// 文档: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/supervisor/
	AgentTypeSupervisor AgentType = "supervisor"

	// AgentTypePlanExecute 规划-执行 Agent
	// 使用 adk/prebuilt/planexecute.New 创建
	// 特点：Planner(规划) -> Executor(执行) -> Replanner(重规划) 循环
	// 文档: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/plan_execute/
	AgentTypePlanExecute AgentType = "plan_execute"

	// AgentTypeDeep 深度 Agent (预构建)
	// 使用 adk/prebuilt/deep.New 创建
	// 特点：内置 WriteTodos 规划、文件系统、Shell、子 Agent 委派
	// 文档: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/deepagents/
	AgentTypeDeep AgentType = "deep"

	// AgentTypeCustom 自定义 Agent
	// 直接实现 adk.Agent 接口
	AgentTypeCustom AgentType = "custom"
)

// AgentInfo Agent 信息
type AgentInfo struct {
	Type         string     `json:"type"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Capabilities []string   `json:"capabilities"`
	Tools        []ToolInfo `json:"tools"`
	Available    bool       `json:"available"`
}

// ToolInfo 工具信息
type ToolInfo struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	ParametersSchema string `json:"parametersSchema"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	Name          string
	Description   string
	Instruction   string
	MaxIterations int
	Tools         []tool.BaseTool
}

// Agent Agent 接口
// 注意：具体实现（如 ChatModelAgent）同时实现 adk.Agent 接口
type Agent interface {
	// Info 获取 Agent 信息
	Info() *AgentInfo
	// RunWithOpts 运行 Agent（带选项）
	RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error)
	// StreamWithOpts 流式运行 Agent
	StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error)
	// GetADKAgent 获取底层的 adk.Agent
	GetADKAgent() adk.Agent
}

// RunOption 运行选项
type RunOption func(*RunOptions)

// RunOptions 运行选项
type RunOptions struct {
	SessionID   string
	UserID      string
	History     []*schema.Message
	Temperature float64
	MaxTokens   int
}

// WithSessionID 设置会话 ID
func WithSessionID(sessionID string) RunOption {
	return func(o *RunOptions) { o.SessionID = sessionID }
}

// WithUserID 设置用户 ID
func WithUserID(userID string) RunOption {
	return func(o *RunOptions) { o.UserID = userID }
}

// WithHistory 设置历史消息
func WithHistory(history []*schema.Message) RunOption {
	return func(o *RunOptions) { o.History = history }
}

// WithTemperature 设置温度
func WithTemperature(temp float64) RunOption {
	return func(o *RunOptions) { o.Temperature = temp }
}

// WithMaxTokens 设置最大 Token
func WithMaxTokens(maxTokens int) RunOption {
	return func(o *RunOptions) { o.MaxTokens = maxTokens }
}

// RunResult 运行结果
type RunResult struct {
	Message   *schema.Message
	ToolCalls []schema.ToolCall
	Usage     *UsageInfo
	AgentName string
}

// UsageInfo Token 使用量
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamEvent 流式事件
type StreamEvent struct {
	Type     string           // start, chunk, tool_start, tool_result, end, error
	Content  string           // 文本内容
	ToolName string           // 工具名
	ToolCall *schema.ToolCall // 工具调用
	Result   string           // 工具结果
	Error    error            // 错误
	Done     bool             // 是否完成
}

// =============================================================================
// ChatModelAgent - ReAct 工具调用 Agent
// =============================================================================

// ChatModelAgent ReAct 工具调用 Agent
// 使用 adk.NewChatModelAgent 创建
type ChatModelAgent struct {
	config AgentConfig
	agent  adk.Agent
	runner *adk.Runner
}

// NewChatModelAgent 创建 ChatModel Agent
// 这是 Eino 的基础 Agent 类型，实现 ReAct 模式
func NewChatModelAgent(ctx context.Context, model model.BaseChatModel, cfg AgentConfig) (*ChatModelAgent, error) {
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 20
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        cfg.Name,
		Description: cfg.Description,
		Instruction: cfg.Instruction,
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		},
		MaxIterations: cfg.MaxIterations,
	})
	if err != nil {
		return nil, fmt.Errorf("create chat model agent failed: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &ChatModelAgent{
		config: cfg,
		agent:  agent,
		runner: runner,
	}, nil
}

// Info 获取 Agent 信息
func (a *ChatModelAgent) Info() *AgentInfo {
	tools := make([]ToolInfo, 0, len(a.config.Tools))
	for _, t := range a.config.Tools {
		if info, err := t.Info(context.Background()); err == nil {
			tools = append(tools, ToolInfo{
				Name:        info.Name,
				Description: info.Desc,
			})
		}
	}

	return &AgentInfo{
		Type:         AgentTypeChatModel,
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: []string{"工具调用", "多轮对话", "ReAct 推理", "任务执行"},
		Tools:        tools,
		Available:    true,
	}
}

// Run 运行 Agent
func (a *ChatModelAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	var result *RunResult

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				result = &RunResult{
					Message:   msg,
					AgentName: event.AgentName,
				}
			}
		}
	}

	return result, nil
}

// Stream 流式运行 Agent
func (a *ChatModelAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	eventChan := make(chan *StreamEvent, 100)

	go func() {
		defer close(eventChan)

		for {
			event, ok := iter.Next()
			if !ok {
				eventChan <- &StreamEvent{Type: "end", Done: true}
				return
			}

			if event.Err != nil {
				eventChan <- &StreamEvent{Type: "error", Error: event.Err}
				return
			}

			if event.Output != nil && event.Output.MessageOutput != nil {
				if event.Output.MessageOutput.IsStreaming {
					for {
						chunk, err := event.Output.MessageOutput.MessageStream.Recv()
						if err != nil {
							break
						}
						eventChan <- &StreamEvent{
							Type:    "chunk",
							Content: chunk.Content,
						}
					}
				} else {
					msg, err := event.Output.MessageOutput.GetMessage()
					if err == nil {
						eventChan <- &StreamEvent{
							Type:    "chunk",
							Content: msg.Content,
						}
					}
				}
			}
		}
	}()
	return eventChan, nil
}

// GetADKAgent 获取底层的 adk.Agent
func (a *ChatModelAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// =============================================================================
// SequentialAgent - 顺序执行 Agent (Workflow Agent)
// =============================================================================

// SequentialAgent 顺序执行 Agent
// 使用 adk.NewSequentialAgent 创建
type SequentialAgent struct {
	config    AgentConfig
	agent     adk.Agent
	runner    *adk.Runner
	subAgents []adk.Agent
}

// SequentialAgentConfig SequentialAgent 配置
type SequentialAgentConfig struct {
	Name        string
	Description string
	SubAgents   []adk.Agent // 子 Agent 列表，按顺序执行
}

// NewSequentialAgent 创建顺序执行 Agent
// 流水线模式：Agent1 -> Agent2 -> Agent3 -> ...
func NewSequentialAgent(ctx context.Context, cfg SequentialAgentConfig) (*SequentialAgent, error) {
	if len(cfg.SubAgents) == 0 {
		return nil, fmt.Errorf("sequential agent requires at least one sub agent")
	}

	agent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        cfg.Name,
		Description: cfg.Description,
		SubAgents:   cfg.SubAgents,
	})
	if err != nil {
		return nil, fmt.Errorf("create sequential agent failed: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &SequentialAgent{
		config: AgentConfig{
			Name:        cfg.Name,
			Description: cfg.Description,
		},
		agent:     agent,
		runner:    runner,
		subAgents: cfg.SubAgents,
	}, nil
}

// Info 获取 Agent 信息
func (a *SequentialAgent) Info() *AgentInfo {
	return &AgentInfo{
		Type:         AgentTypeSequential,
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: []string{"顺序执行", "流水线处理", "多阶段任务"},
		Available:    true,
	}
}

// Run 运行 Agent
func (a *SequentialAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	var result *RunResult

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				result = &RunResult{
					Message:   msg,
					AgentName: event.AgentName,
				}
			}
		}
	}

	return result, nil
}

// Stream 流式运行 Agent
func (a *SequentialAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	eventChan := make(chan *StreamEvent, 100)

	go streamAgentEvents(iter, eventChan)
	return eventChan, nil
}

// GetADKAgent 获取底层的 adk.Agent
func (a *SequentialAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// =============================================================================
// LoopAgent - 循环执行 Agent (WorkflowWorkflow Agent)
// =============================================================================

// LoopAgent 循环执行 Agent
// 使用 adk.NewLoopAgent 创建
type LoopAgent struct {
	config    AgentConfig
	agent     adk.Agent
	runner    *adk.Runner
	subAgents []adk.Agent
}

// LoopAgentConfig LoopAgent 配置
type LoopAgentConfig struct {
	Name          string
	Description   string
	SubAgents     []adk.Agent // 子 Agent 列表
	MaxIterations int         // 最大迭代次数，0 表示无限循环
}

// NewLoopAgent 创建循环执行 Agent
func NewLoopAgent(ctx context.Context, cfg LoopAgentConfig) (*LoopAgent, error) {
	if len(cfg.SubAgents) == 0 {
		return nil, fmt.Errorf("loop agent requires at least one sub agent")
	}

	agent, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		SubAgents:     cfg.SubAgents,
		MaxIterations: cfg.MaxIterations,
	})
	if err != nil {
		return nil, fmt.Errorf("create loop agent failed: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &LoopAgent{
		config: AgentConfig{
			Name:        cfg.Name,
			Description: cfg.Description,
		},
		agent:     agent,
		runner:    runner,
		subAgents: cfg.SubAgents,
	}, nil
}

// Info 获取 Agent 信息
func (a *LoopAgent) Info() *AgentInfo {
	return &AgentInfo{
		Type:         AgentTypeLoop,
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: []string{"循环执行", "迭代优化", "条件退出"},
		Available:    true,
	}
}

// Run 运行 Agent
func (a *LoopAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	var result *RunResult

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				result = &RunResult{
					Message:   msg,
					AgentName: event.AgentName,
				}
			}
		}
	}

	return result, nil
}

// Stream 流式运行 Agent
func (a *LoopAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	eventChan := make(chan *StreamEvent, 100)

	go streamAgentEvents(iter, eventChan)
	return eventChan, nil
}

// GetADKAgent 获取底层的 adk.Agent
func (a *LoopAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// =============================================================================
// ParallelAgent - 并行执行 Agent (Workflow Agent)
// 参考: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/workflow/
// =============================================================================

// ParallelAgent 并行执行 Agent
// 使用 adk.NewParallelAgent 创建
type ParallelAgent struct {
	config    AgentConfig
	agent     adk.Agent
	runner    *adk.Runner
	subAgents []adk.Agent
}

// ParallelAgentConfig ParallelAgent 配置
type ParallelAgentConfig struct {
	Name        string      // Agent 名称
	Description string      // Agent 描述
	SubAgents   []adk.Agent // 并行执行的子 Agent 列表
}

// NewParallelAgent 创建并行执行 Agent
// 并行模式：Agent1 || Agent2 || Agent3 -> 合并结果
func NewParallelAgent(ctx context.Context, cfg ParallelAgentConfig) (*ParallelAgent, error) {
	if len(cfg.SubAgents) == 0 {
		return nil, fmt.Errorf("parallel agent requires at least one sub agent")
	}

	agent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        cfg.Name,
		Description: cfg.Description,
		SubAgents:   cfg.SubAgents,
	})
	if err != nil {
		return nil, fmt.Errorf("create parallel agent failed: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &ParallelAgent{
		config: AgentConfig{
			Name:        cfg.Name,
			Description: cfg.Description,
		},
		agent:     agent,
		runner:    runner,
		subAgents: cfg.SubAgents,
	}, nil
}

// Info 获取 Agent 信息
func (a *ParallelAgent) Info() *AgentInfo {
	tools := make([]ToolInfo, len(a.subAgents))
	for i, sub := range a.subAgents {
		tools[i] = ToolInfo{
			Name:        sub.Name(context.Background()),
			Description: "并行子 Agent",
		}
	}

	return &AgentInfo{
		Type:         AgentTypeParallel,
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: []string{"并行执行", "并发处理", "多任务同时处理", "结果合并"},
		Tools:        tools,
		Available:    true,
	}
}

// Run 运行 Agent
func (a *ParallelAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	var result *RunResult

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				result = &RunResult{
					Message:   msg,
					AgentName: event.AgentName,
				}
			}
		}
	}

	return result, nil
}

// Stream 流式运行 Agent
func (a *ParallelAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	eventChan := make(chan *StreamEvent, 100)

	go streamAgentEvents(iter, eventChan)
	return eventChan, nil
}

// GetADKAgent 获取底层的 adk.Agent
func (a *ParallelAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// =============================================================================
// SupervisorAgent - 监督者 Agent
// =============================================================================

// SupervisorAgent 监督者 Agent
// 使用 adk/prebuilt/supervisor.New 创建
type SupervisorAgent struct {
	config     AgentConfig
	agent      adk.Agent
	runner     *adk.Runner
	supervisor adk.Agent
	subAgents  []adk.Agent
}

// SupervisorAgentConfig SupervisorAgent 配置
type SupervisorAgentConfig struct {
	Name        string
	Description string
	Supervisor  adk.Agent   // 监督者 Agent
	SubAgents   []adk.Agent // 子 Agent 列表
}

// NewSupervisorAgent 创建监督者 Agent
// 监督者模式：Supervisor 动态分配任务给合适的子 Agent
func NewSupervisorAgent(ctx context.Context, cfg SupervisorAgentConfig) (*SupervisorAgent, error) {
	if cfg.Supervisor == nil {
		return nil, fmt.Errorf("supervisor agent is required")
	}
	if len(cfg.SubAgents) == 0 {
		return nil, fmt.Errorf("supervisor agent requires at least one sub agent")
	}

	agent, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: cfg.Supervisor,
		SubAgents:  cfg.SubAgents,
	})
	if err != nil {
		return nil, fmt.Errorf("create supervisor agent failed: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &SupervisorAgent{
		config: AgentConfig{
			Name:        cfg.Name,
			Description: cfg.Description,
		},
		agent:      agent,
		runner:     runner,
		supervisor: cfg.Supervisor,
		subAgents:  cfg.SubAgents,
	}, nil
}

// Info 获取 Agent 信息
func (a *SupervisorAgent) Info() *AgentInfo {
	tools := make([]ToolInfo, len(a.subAgents))
	for i, sub := range a.subAgents {
		tools[i] = ToolInfo{
			Name:        sub.Name(context.Background()),
			Description: "子 Agent",
		}
	}

	return &AgentInfo{
		Type:         AgentTypeSupervisor,
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: []string{"动态任务分配", "多 Agent 协调", "监督执行"},
		Tools:        tools,
		Available:    true,
	}
}

// Run 运行 Agent
func (a *SupervisorAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	var result *RunResult

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				result = &RunResult{
					Message:   msg,
					AgentName: event.AgentName,
				}
			}
		}
	}

	return result, nil
}

// Stream 流式运行 Agent
func (a *SupervisorAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	eventChan := make(chan *StreamEvent, 100)

	go streamAgentEvents(iter, eventChan)
	return eventChan, nil
}

// GetADKAgent 获取底层的 adk.Agent
func (a *SupervisorAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// =============================================================================
// PlanExecuteAgent - 规划-执行 Agent
// =============================================================================

// PlanExecuteAgent 规划-执行 Agent
// 使用 adk/prebuilt/planexecute.New 创建
type PlanExecuteAgent struct {
	config    AgentConfig
	agent     adk.Agent
	runner    *adk.Runner
	planner   adk.Agent
	executor  adk.Agent
	replanner adk.Agent
}

// PlanExecuteAgentConfig PlanExecuteAgent 配置
type PlanExecuteAgentConfig struct {
	Name          string
	Description   string
	Planner       adk.Agent // 规划器
	Executor      adk.Agent // 执行器
	Replanner     adk.Agent // 重规划器
	MaxIterations int       // 最大迭代次数
}

// NewPlanExecuteAgent 创建规划-执行 Agent
// Planner(规划) -> Executor(执行) -> Replanner(重规划) 循环
func NewPlanExecuteAgent(ctx context.Context, cfg PlanExecuteAgentConfig) (*PlanExecuteAgent, error) {
	if cfg.Planner == nil || cfg.Executor == nil || cfg.Replanner == nil {
		return nil, fmt.Errorf("planner, executor and replanner are required")
	}

	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 10
	}

	agent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       cfg.Planner,
		Executor:      cfg.Executor,
		Replanner:     cfg.Replanner,
		MaxIterations: cfg.MaxIterations,
	})
	if err != nil {
		return nil, fmt.Errorf("create plan execute agent failed: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &PlanExecuteAgent{
		config: AgentConfig{
			Name:        cfg.Name,
			Description: cfg.Description,
		},
		agent:     agent,
		runner:    runner,
		planner:   cfg.Planner,
		executor:  cfg.Executor,
		replanner: cfg.Replanner,
	}, nil
}

// Info 获取 Agent 信息
func (a *PlanExecuteAgent) Info() *AgentInfo {
	return &AgentInfo{
		Type:         AgentTypePlanExecute,
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: []string{"任务规划", "分步执行", "动态调整", "进度评估"},
		Tools: []ToolInfo{
			{Name: "PlanTool", Description: "生成任务计划"},
			{Name: "RespondTool", Description: "返回最终结果"},
		},
		Available: true,
	}
}

// Run 运行 Agent
func (a *PlanExecuteAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	var result *RunResult

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				result = &RunResult{
					Message:   msg,
					AgentName: event.AgentName,
				}
			}
		}
	}

	return result, nil
}

// Stream 流式运行 Agent
func (a *PlanExecuteAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	eventChan := make(chan *StreamEvent, 100)

	go streamAgentEvents(iter, eventChan)
	return eventChan, nil
}

// GetADKAgent 获取底层的 adk.Agent
func (a *PlanExecuteAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// =============================================================================
// DeepAgent - 深度 Agent (预构建)
// =============================================================================

// DeepAgent 深度 Agent
// 使用 adk/prebuilt/deep.New 创建
type DeepAgent struct {
	config AgentConfig
	agent  adk.Agent
	runner *adk.Runner
}

// DeepAgentConfig DeepAgent 配置
type DeepAgentConfig struct {
	Name             string
	Description      string
	Instruction      string
	Model            model.BaseChatModel
	Tools            []tool.BaseTool // 自定义工具
	SubAgents        []adk.Agent     // 子 Agent
	MaxIterations    int             // 最大迭代次数
	EnableWriteTodos bool            // 是否启用 WriteTodos
	EnableFileSystem bool            // 是否启用文件系统
}

// NewDeepAgent 创建 Deep Agent
// DeepAgent 是预构建的 Agent，开箱即用地提供：
// 1. WriteTodos 工具 - 任务规划
// 2. 文件系统工具 - read_file, write_file, edit_file, glob, grep, execute
// 3. 子 Agent 委派 - TaskTool
func NewDeepAgent(ctx context.Context, cfg DeepAgentConfig) (*DeepAgent, error) {
	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 30
	}

	deepCfg := &deep.Config{
		Name:              cfg.Name,
		Description:       cfg.Description,
		ChatModel:         cfg.Model,
		Instruction:       cfg.Instruction,
		SubAgents:         cfg.SubAgents,
		WithoutWriteTodos: !cfg.EnableWriteTodos,
		MaxIteration:      maxIter,
	}

	// 如果启用了文件系统，创建内存后端
	if cfg.EnableFileSystem {
		deepCfg.Backend = filesystem.NewInMemoryBackend()
	}

	agent, err := deep.New(ctx, deepCfg)
	if err != nil {
		return nil, fmt.Errorf("create deep agent failed: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &DeepAgent{
		config: AgentConfig{
			Name:        cfg.Name,
			Description: cfg.Description,
			Instruction: cfg.Instruction,
		},
		agent:  agent,
		runner: runner,
	}, nil
}

// Info 获取 Agent 信息
func (a *DeepAgent) Info() *AgentInfo {
	return &AgentInfo{
		Type:         AgentTypeDeep,
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: []string{"任务规划", "文件操作", "多步骤执行", "深度思考", "子Agent委派"},
		Tools: []ToolInfo{
			{Name: "write_todos", Description: "任务规划工具"},
			{Name: "read_file", Description: "读取文件"},
			{Name: "write_file", Description: "写入文件"},
			{Name: "edit_file", Description: "编辑文件"},
			{Name: "glob", Description: "文件搜索"},
			{Name: "grep", Description: "内容搜索"},
			{Name: "execute", Description: "执行命令"},
			{Name: "task", Description: "子Agent委派"},
		},
		Available: true,
	}
}

// Run 运行 Agent
func (a *DeepAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	var result *RunResult

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				result = &RunResult{
					Message:   msg,
					AgentName: event.AgentName,
				}
			}
		}
	}

	return result, nil
}

// Stream 流式运行 Agent
func (a *DeepAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	options := &RunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	iter := a.runner.Query(ctx, input)
	eventChan := make(chan *StreamEvent, 100)

	go streamAgentEvents(iter, eventChan)
	return eventChan, nil
}

// GetADKAgent 获取底层的 adk.Agent
func (a *DeepAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// =============================================================================
// CustomAgent - 自定义 Agent
// =============================================================================

// CustomAgent 自定义 Agent
// 直接实现 adk.Agent 接口
type CustomAgent struct {
	name        string
	description string
	runFunc     func(ctx context.Context, input string, opts ...RunOption) (*RunResult, error)
	streamFunc  func(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error)
}

// NewCustomAgent 创建自定义 Agent
func NewCustomAgent(name, description string, runFunc func(ctx context.Context, input string, opts ...RunOption) (*RunResult, error), streamFunc func(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error)) *CustomAgent {
	return &CustomAgent{
		name:        name,
		description: description,
		runFunc:     runFunc,
		streamFunc:  streamFunc,
	}
}

// Info 获取 Agent 信息
func (a *CustomAgent) Info() *AgentInfo {
	return &AgentInfo{
		Type:         AgentTypeCustom,
		Name:         a.name,
		Description:  a.description,
		Capabilities: []string{"自定义能力"},
		Available:    true,
	}
}

// Run 运行 Agent
func (a *CustomAgent) RunWithOpts(ctx context.Context, input string, opts ...RunOption) (*RunResult, error) {
	if a.runFunc == nil {
		return nil, fmt.Errorf("run function not implemented")
	}
	return a.runFunc(ctx, input, opts...)
}

// Stream 流式运行 Agent
func (a *CustomAgent) StreamWithOpts(ctx context.Context, input string, opts ...RunOption) (<-chan *StreamEvent, error) {
	if a.streamFunc == nil {
		return nil, fmt.Errorf("stream function not implemented")
	}
	return a.streamFunc(ctx, input, opts...)
}

// GetADKAgent 获取底层的 adk.Agent
func (a *CustomAgent) GetADKAgent() adk.Agent {
	return nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// streamAgentEvents 通用的流式事件处理
func streamAgentEvents(iter *adk.AsyncIterator[*adk.AgentEvent], eventChan chan<- *StreamEvent) {
	defer close(eventChan)

	for {
		event, ok := iter.Next()
		if !ok {
			eventChan <- &StreamEvent{Type: "end", Done: true}
			return
		}

		if event.Err != nil {
			eventChan <- &StreamEvent{Type: "error", Error: event.Err}
			return
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			if event.Output.MessageOutput.IsStreaming {
				for {
					chunk, err := event.Output.MessageOutput.MessageStream.Recv()
					if err != nil {
						break
					}
					eventChan <- &StreamEvent{
						Type:    "chunk",
						Content: chunk.Content,
					}
				}
			} else {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err == nil {
					eventChan <- &StreamEvent{
						Type:    "chunk",
						Content: msg.Content,
					}
				}
			}
		}
	}
}

// GetAgentTypes 获取所有 Agent 类型
func GetAgentTypes() []AgentType {
	return []AgentType{
		AgentTypeChatModel,
		AgentTypeSequential,
		AgentTypeLoop,
		AgentTypeParallel,
		AgentTypeSupervisor,
		AgentTypePlanExecute,
		AgentTypeDeep,
		AgentTypeCustom,
	}
}

// GetAgentTypeInfo 获取 Agent 类型信息
func GetAgentTypeInfo(agentType AgentType) *AgentInfo {
	infos := map[AgentType]*AgentInfo{
		AgentTypeChatModel: {
			Type:         AgentTypeChatModel,
			Name:         "ChatModel Agent",
			Description:  "ReAct 工具调用 Agent，LLM 推理 -> 工具调用 -> 循环",
			Capabilities: []string{"工具调用", "多轮对话", "ReAct 推理"},
		},
		AgentTypeSequential: {
			Type:         AgentTypeSequential,
			Name:         "Sequential Agent",
			Description:  "顺序执行 Agent，按顺序依次执行子 Agent",
			Capabilities: []string{"顺序执行", "流水线处理"},
		},
		AgentTypeLoop: {
			Type:         AgentTypeLoop,
			Name:         "Loop Agent",
			Description:  "循环执行 Agent，循环执行直到条件满足",
			Capabilities: []string{"循环执行", "迭代优化"},
		},
		AgentTypeParallel: {
			Type:         AgentTypeParallel,
			Name:         "Parallel Agent",
			Description:  "并行执行 Agent，并发执行多个子 Agent",
			Capabilities: []string{"并行执行", "并发处理"},
		},
		AgentTypeSupervisor: {
			Type:         AgentTypeSupervisor,
			Name:         "Supervisor Agent",
			Description:  "监督者 Agent，协调多个子 Agent 动态分配任务",
			Capabilities: []string{"动态任务分配", "多 Agent 协调"},
		},
		AgentTypePlanExecute: {
			Type:         AgentTypePlanExecute,
			Name:         "Plan-Execute Agent",
			Description:  "规划-执行 Agent，Planner -> Executor -> Replanner 循环",
			Capabilities: []string{"任务规划", "分步执行", "动态调整"},
		},
		AgentTypeDeep: {
			Type:         AgentTypeDeep,
			Name:         "Deep Agent",
			Description:  "深度 Agent，内置规划、文件系统、子 Agent 委派",
			Capabilities: []string{"任务规划", "文件操作", "子Agent委派"},
		},
		AgentTypeCustom: {
			Type:         AgentTypeCustom,
			Name:         "Custom Agent",
			Description:  "自定义 Agent，实现 adk.Agent 接口",
			Capabilities: []string{"自定义能力"},
		},
	}

	if info, ok := infos[agentType]; ok {
		info.Available = true
		return info
	}

	return &AgentInfo{
		Type:        agentType,
		Name:        "Unknown Agent",
		Description: "未知 Agent 类型",
		Available:   false,
	}
}
