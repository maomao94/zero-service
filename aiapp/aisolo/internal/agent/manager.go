package agent

import (
	"context"
	"fmt"
	"sync"

	"zero-service/aiapp/aisolo/internal/config"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// Manager Agent 管理器
// 管理多种类型的 Agent，支持动态创建和获取
type Manager struct {
	mu        sync.RWMutex
	agents    map[string]Agent     // Agent 实例
	model     model.BaseChatModel  // 底座模型
	subAgents map[string]adk.Agent // 子 Agent（用于组合型 Agent）
	config    *config.Config       // 配置
}

// NewManager 创建 Agent 管理器
func NewManager(cfg *config.Config, chatModel model.BaseChatModel) (*Manager, error) {
	m := &Manager{
		agents:    make(map[string]Agent),
		model:     chatModel,
		subAgents: make(map[string]adk.Agent),
		config:    cfg,
	}

	ctx := context.Background()

	// 1. 创建基础 Agent（这些可以作为子 Agent 被其他 Agent 使用）
	m.createBaseAgents(ctx)

	// 2. 创建主要 Agent
	if err := m.initAgents(ctx); err != nil {
		return nil, fmt.Errorf("init agents failed: %w", err)
	}

	return m, nil
}

// createBaseAgents 创建基础 Agent（作为子 Agent）
func (m *Manager) createBaseAgents(ctx context.Context) {
	// 创建默认的简单 Agent 用于子 Agent 组合
	defaultInstruction := "你是一个有帮助的 AI 助手。"

	// Researcher 子 Agent
	m.subAgents["researcher"] = m.createSimpleAgent(ctx, "Researcher", defaultInstruction+"你负责研究和收集信息。")

	// Coder 子 Agent
	m.subAgents["coder"] = m.createSimpleAgent(ctx, "Coder", defaultInstruction+"你负责编写和修改代码。")

	// Writer 子 Agent
	m.subAgents["writer"] = m.createSimpleAgent(ctx, "Writer", defaultInstruction+"你负责撰写文档和内容。")

	// Analyst 子 Agent
	m.subAgents["analyst"] = m.createSimpleAgent(ctx, "Analyst", defaultInstruction+"你负责分析数据和信息。")
}

// createSimpleAgent 创建一个简单的 ChatModelAgent
func (m *Manager) createSimpleAgent(ctx context.Context, name, instruction string) adk.Agent {
	cfg := &adk.ChatModelAgentConfig{
		Name:          name,
		Description:   name + " - AI Assistant",
		Instruction:   instruction,
		Model:         m.model,
		MaxIterations: 10,
	}

	agent, err := adk.NewChatModelAgent(ctx, cfg)
	if err != nil {
		logx.Errorf("create sub agent %s failed: %v", name, err)
		return nil
	}

	return agent
}

// initAgents 初始化所有配置的 Agent
func (m *Manager) initAgents(ctx context.Context) error {
	// 1. ChatModelAgent - 最基础的 ReAct 工具调用 Agent
	if m.config.Agents.ChatModel.Name != "" {
		agent, err := m.createChatModelAgent(ctx)
		if err != nil {
			logx.Errorf("create ChatModelAgent failed: %v", err)
		} else {
			m.agents[AgentTypeChatModel] = agent
			logx.Infof("ChatModelAgent initialized: %s", agent.Info().Name)
		}
	}

	// 2. DeepAgent - 深度规划 Agent
	if m.config.Agents.Deep.Name != "" {
		agent, err := m.createDeepAgent(ctx)
		if err != nil {
			logx.Errorf("create DeepAgent failed: %v", err)
		} else {
			m.agents[AgentTypeDeep] = agent
			logx.Infof("DeepAgent initialized: %s", agent.Info().Name)
		}
	}

	// 3. SequentialAgent - 顺序执行 Agent
	if m.config.Agents.Sequential.Name != "" && len(m.config.Agents.Sequential.SubAgentNames) > 0 {
		agent, err := m.createSequentialAgent(ctx)
		if err != nil {
			logx.Errorf("create SequentialAgent failed: %v", err)
		} else {
			m.agents[AgentTypeSequential] = agent
			logx.Infof("SequentialAgent initialized: %s", agent.Info().Name)
		}
	}

	// 4. ParallelAgent - 并行执行 Agent
	if m.config.Agents.Parallel.Name != "" && len(m.config.Agents.Parallel.SubAgentNames) > 0 {
		agent, err := m.createParallelAgent(ctx)
		if err != nil {
			logx.Errorf("create ParallelAgent failed: %v", err)
		} else {
			m.agents[AgentTypeParallel] = agent
			logx.Infof("ParallelAgent initialized: %s", agent.Info().Name)
		}
	}

	// 5. SupervisorAgent - 监督者 Agent
	if m.config.Agents.Supervisor.Name != "" {
		agent, err := m.createSupervisorAgent(ctx)
		if err != nil {
			logx.Errorf("create SupervisorAgent failed: %v", err)
		} else {
			m.agents[AgentTypeSupervisor] = agent
			logx.Infof("SupervisorAgent initialized: %s", agent.Info().Name)
		}
	}

	// 6. PlanExecuteAgent - 规划-执行 Agent
	if m.config.Agents.PlanExecute.Name != "" {
		agent, err := m.createPlanExecuteAgent(ctx)
		if err != nil {
			logx.Errorf("create PlanExecuteAgent failed: %v", err)
		} else {
			m.agents[AgentTypePlanExecute] = agent
			logx.Infof("PlanExecuteAgent initialized: %s", agent.Info().Name)
		}
	}

	// 7. LoopAgent - 循环执行 Agent
	if m.config.Agents.Loop.Name != "" && len(m.config.Agents.Loop.SubAgentNames) > 0 {
		agent, err := m.createLoopAgent(ctx)
		if err != nil {
			logx.Errorf("create LoopAgent failed: %v", err)
		} else {
			m.agents[AgentTypeLoop] = agent
			logx.Infof("LoopAgent initialized: %s", agent.Info().Name)
		}
	}

	return nil
}

// createChatModelAgent 创建 ChatModelAgent
func (m *Manager) createChatModelAgent(ctx context.Context) (*ChatModelAgent, error) {
	cfg := AgentConfig{
		Name:          m.config.Agents.ChatModel.Name,
		Description:   m.config.Agents.ChatModel.Description,
		Instruction:   m.config.Agents.ChatModel.Instruction,
		MaxIterations: m.config.Agents.ChatModel.MaxIterations,
		Tools:         nil, // 当前配置暂无内置工具
	}

	return NewChatModelAgent(ctx, m.model, cfg)
}

// createDeepAgent 创建 DeepAgent
func (m *Manager) createDeepAgent(ctx context.Context) (*DeepAgent, error) {
	cfg := DeepAgentConfig{
		Name:             m.config.Agents.Deep.Name,
		Description:      m.config.Agents.Deep.Description,
		Instruction:      m.config.Agents.Deep.Instruction,
		MaxIterations:    m.config.Agents.Deep.MaxIterations,
		Model:            m.model,
		Tools:            nil, // 当前配置暂无自定义工具
		EnableWriteTodos: true,
		EnableFileSystem: m.config.Agents.Deep.EnableFileSystem,
	}

	return NewDeepAgent(ctx, cfg)
}

// createSequentialAgent 创建 SequentialAgent
func (m *Manager) createSequentialAgent(ctx context.Context) (*SequentialAgent, error) {
	// 获取子 Agent
	var subAgents []adk.Agent
	for _, name := range m.config.Agents.Sequential.SubAgentNames {
		if sub, ok := m.subAgents[name]; ok && sub != nil {
			subAgents = append(subAgents, sub)
		}
	}

	if len(subAgents) == 0 {
		return nil, fmt.Errorf("sequential agent requires at least one sub agent")
	}

	cfg := SequentialAgentConfig{
		Name:        m.config.Agents.Sequential.Name,
		Description: m.config.Agents.Sequential.Description,
		SubAgents:   subAgents,
	}

	return NewSequentialAgent(ctx, cfg)
}

// createParallelAgent 创建 ParallelAgent
func (m *Manager) createParallelAgent(ctx context.Context) (*ParallelAgent, error) {
	var subAgents []adk.Agent
	for _, name := range m.config.Agents.Parallel.SubAgentNames {
		if sub, ok := m.subAgents[name]; ok && sub != nil {
			subAgents = append(subAgents, sub)
		}
	}

	if len(subAgents) == 0 {
		return nil, fmt.Errorf("parallel agent requires at least one sub agent")
	}

	cfg := ParallelAgentConfig{
		Name:        m.config.Agents.Parallel.Name,
		Description: m.config.Agents.Parallel.Description,
		SubAgents:   subAgents,
	}

	return NewParallelAgent(ctx, cfg)
}

// createSupervisorAgent 创建 SupervisorAgent
func (m *Manager) createSupervisorAgent(ctx context.Context) (*SupervisorAgent, error) {
	// 创建监督者
	supervisorAgent := m.createSimpleAgent(ctx, "Supervisor",
		m.config.Agents.Supervisor.Instruction+"你是任务协调者，负责分配任务给合适的专家。")

	var subAgents []adk.Agent
	for _, name := range m.config.Agents.Supervisor.SubAgentNames {
		if sub, ok := m.subAgents[name]; ok && sub != nil {
			subAgents = append(subAgents, sub)
		}
	}

	if len(subAgents) == 0 {
		return nil, fmt.Errorf("supervisor agent requires at least one sub agent")
	}

	cfg := SupervisorAgentConfig{
		Name:        m.config.Agents.Supervisor.Name,
		Description: m.config.Agents.Supervisor.Description,
		Supervisor:  supervisorAgent,
		SubAgents:   subAgents,
	}

	return NewSupervisorAgent(ctx, cfg)
}

// createPlanExecuteAgent 创建 PlanExecuteAgent
func (m *Manager) createPlanExecuteAgent(ctx context.Context) (*PlanExecuteAgent, error) {
	// 创建 Planner, Executor, Replanner
	planner := m.createSimpleAgent(ctx, "Planner",
		m.config.Agents.PlanExecute.PlannerInstruction)
	executor := m.createSimpleAgent(ctx, "Executor",
		m.config.Agents.PlanExecute.ExecutorInstruction)
	replanner := m.createSimpleAgent(ctx, "Replanner",
		m.config.Agents.PlanExecute.ReplannerInstruction)

	cfg := PlanExecuteAgentConfig{
		Name:          m.config.Agents.PlanExecute.Name,
		Description:   m.config.Agents.PlanExecute.Description,
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: m.config.Agents.PlanExecute.MaxIterations,
	}

	return NewPlanExecuteAgent(ctx, cfg)
}

// createLoopAgent 创建 LoopAgent
func (m *Manager) createLoopAgent(ctx context.Context) (*LoopAgent, error) {
	var subAgents []adk.Agent
	for _, name := range m.config.Agents.Loop.SubAgentNames {
		if sub, ok := m.subAgents[name]; ok && sub != nil {
			subAgents = append(subAgents, sub)
		}
	}

	cfg := LoopAgentConfig{
		Name:          m.config.Agents.Loop.Name,
		Description:   m.config.Agents.Loop.Description,
		SubAgents:     subAgents,
		MaxIterations: m.config.Agents.Loop.MaxIterations,
	}

	return NewLoopAgent(ctx, cfg)
}

// Get 获取指定类型的 Agent
func (m *Manager) Get(agentType string) Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if agent, ok := m.agents[agentType]; ok {
		return agent
	}

	// 如果找不到，返回默认的 ChatModelAgent
	if defaultAgent, ok := m.agents[AgentTypeChatModel]; ok {
		return defaultAgent
	}

	return nil
}

// List 获取所有 Agent 信息
func (m *Manager) List() []*AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []*AgentInfo
	for _, agent := range m.agents {
		infos = append(infos, agent.Info())
	}

	// 如果没有初始化任何 Agent，返回默认信息
	if len(infos) == 0 {
		infos = append(infos, GetDefaultAgentTypes()...)
	}

	return infos
}

// GetByType 根据类型字符串获取 Agent
func (m *Manager) GetByType(agentType string) Agent {
	switch agentType {
	case AgentTypeChatModel:
		return m.Get(AgentTypeChatModel)
	case AgentTypeDeep:
		return m.Get(AgentTypeDeep)
	case AgentTypeSequential:
		return m.Get(AgentTypeSequential)
	case AgentTypeParallel:
		return m.Get(AgentTypeParallel)
	case AgentTypeSupervisor:
		return m.Get(AgentTypeSupervisor)
	case AgentTypePlanExecute:
		return m.Get(AgentTypePlanExecute)
	case AgentTypeLoop:
		return m.Get(AgentTypeLoop)
	case AgentTypeCustom:
		return m.Get(AgentTypeCustom)
	default:
		// 默认返回 ChatModelAgent
		return m.Get(AgentTypeChatModel)
	}
}

// GetDefaultAgentTypes 返回默认的 Agent 类型信息
func GetDefaultAgentTypes() []*AgentInfo {
	return []*AgentInfo{
		{
			Type:         AgentTypeChatModel,
			Name:         "ChatModel Agent",
			Description:  "ReAct 工具调用 Agent，LLM 推理 -> 工具调用 -> 循环",
			Capabilities: []string{"工具调用", "多轮对话", "ReAct 推理"},
			Available:    true,
		},
		{
			Type:         AgentTypeDeep,
			Name:         "Deep Agent",
			Description:  "深度规划 Agent，支持任务规划、文件系统和子 Agent 委派",
			Capabilities: []string{"任务规划", "文件操作", "子Agent委派"},
			Available:    true,
		},
		{
			Type:         AgentTypeSequential,
			Name:         "Sequential Agent",
			Description:  "顺序执行 Agent，按顺序依次执行子 Agent",
			Capabilities: []string{"顺序执行", "流水线处理"},
			Available:    true,
		},
		{
			Type:         AgentTypeParallel,
			Name:         "Parallel Agent",
			Description:  "并行执行 Agent，并发执行多个子 Agent",
			Capabilities: []string{"并行执行", "并发处理"},
			Available:    true,
		},
		{
			Type:         AgentTypeSupervisor,
			Name:         "Supervisor Agent",
			Description:  "监督者 Agent，协调多个子 Agent 动态分配任务",
			Capabilities: []string{"动态任务分配", "多 Agent 协调"},
			Available:    true,
		},
		{
			Type:         AgentTypePlanExecute,
			Name:         "Plan-Execute Agent",
			Description:  "规划-执行 Agent，Planner -> Executor -> Replanner 循环",
			Capabilities: []string{"任务规划", "分步执行", "动态调整"},
			Available:    true,
		},
		{
			Type:         AgentTypeLoop,
			Name:         "Loop Agent",
			Description:  "循环执行 Agent，循环执行直到条件满足",
			Capabilities: []string{"循环执行", "迭代优化"},
			Available:    true,
		},
		{
			Type:         AgentTypeCustom,
			Name:         "Custom Agent",
			Description:  "自定义 Agent，实现扩展接口",
			Capabilities: []string{"自定义能力"},
			Available:    false,
		},
	}
}
