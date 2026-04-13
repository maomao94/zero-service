package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/einox/agent/prompts"
)

// =============================================================================
// 内置 Agent 类型
// =============================================================================

// AgentType 内置 Agent 类型
type AgentType string

const (
	AgentTypeAssistant  AgentType = "assistant"  // 通用助手
	AgentTypeResearcher AgentType = "researcher" // 研究助手
	AgentTypeCoder      AgentType = "coder"      // 编程助手
	AgentTypeAnalyst    AgentType = "analyst"    // 分析助手
	AgentTypePlanner    AgentType = "planner"    // 规划助手
	AgentTypeDeep       AgentType = "deep"       // 深度思考
)

// GetPrompt 获取 Agent 类型对应的提示词
func (t AgentType) GetPrompt() string {
	switch t {
	case AgentTypeAssistant:
		return prompts.AssistantPrompt
	case AgentTypeCoder:
		return prompts.CoderPrompt
	case AgentTypeResearcher:
		return prompts.ResearcherPrompt
	case AgentTypeAnalyst:
		return prompts.AnalystPrompt
	case AgentTypePlanner:
		return prompts.PlannerPrompt
	default:
		return prompts.AssistantPrompt
	}
}

// GetDescription 获取 Agent 类型描述
func (t AgentType) GetDescription() string {
	switch t {
	case AgentTypeAssistant:
		return "通用助手 - 回答各类问题，提供帮助"
	case AgentTypeResearcher:
		return "研究助手 - 深入研究，信息收集与分析"
	case AgentTypeCoder:
		return "编程助手 - 代码编写、调试、技术问题"
	case AgentTypeAnalyst:
		return "分析助手 - 数据分析、趋势判断"
	case AgentTypePlanner:
		return "规划助手 - 目标拆解、计划制定"
	case AgentTypeDeep:
		return "深度思考 - 复杂问题的深入分析"
	default:
		return "通用助手"
	}
}

// =============================================================================
// 内置 Agent 工厂函数
// =============================================================================

// NewAssistant 创建通用助手 Agent
func NewAssistant(ctx context.Context, chatModel model.ChatModel, opts ...Option) (*Agent, error) {
	return NewByType(ctx, AgentTypeAssistant, chatModel, opts...)
}

// NewCoder 创建编程助手 Agent
func NewCoder(ctx context.Context, chatModel model.ChatModel, opts ...Option) (*Agent, error) {
	return NewByType(ctx, AgentTypeCoder, chatModel, opts...)
}

// NewResearcher 创建研究助手 Agent
func NewResearcher(ctx context.Context, chatModel model.ChatModel, opts ...Option) (*Agent, error) {
	return NewByType(ctx, AgentTypeResearcher, chatModel, opts...)
}

// NewAnalyst 创建分析助手 Agent
func NewAnalyst(ctx context.Context, chatModel model.ChatModel, opts ...Option) (*Agent, error) {
	return NewByType(ctx, AgentTypeAnalyst, chatModel, opts...)
}

// NewPlanner 创建规划助手 Agent
func NewPlanner(ctx context.Context, chatModel model.ChatModel, opts ...Option) (*Agent, error) {
	return NewByType(ctx, AgentTypePlanner, chatModel, opts...)
}

// NewByType 根据类型创建 Agent
func NewByType(ctx context.Context, agentType AgentType, chatModel model.ChatModel, opts ...Option) (*Agent, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}

	// 构建选项
	agentOpts := []Option{
		WithName(string(agentType)),
		WithDescription(agentType.GetDescription()),
		WithInstruction(agentType.GetPrompt()),
	}
	agentOpts = append(agentOpts, opts...)

	return New(ctx, agentOpts...)
}

// =============================================================================
// 工具注册表
// =============================================================================

// BuiltinTool 内置工具注册表
type BuiltinTool struct {
	Name        string
	Description string
}

// ListBuiltinTools 返回所有内置工具（预留接口）
func ListBuiltinTools() []BuiltinTool {
	return []BuiltinTool{
		{Name: "calculator", Description: "数学计算工具"},
		{Name: "date_time", Description: "日期时间查询"},
		{Name: "web_search", Description: "网页搜索（预留）"},
	}
}

// =============================================================================
// Agent 工厂管理器
// =============================================================================

// FactoryManager Agent 工厂管理器
type FactoryManager struct {
	defaultModel model.ChatModel
}

// NewFactoryManager 创建工厂管理器
func NewFactoryManager(defaultModel model.ChatModel) *FactoryManager {
	return &FactoryManager{
		defaultModel: defaultModel,
	}
}

// CreateAgent 创建指定类型的 Agent
func (fm *FactoryManager) CreateAgent(ctx context.Context, agentType AgentType, opts ...Option) (*Agent, error) {
	return NewByType(ctx, agentType, fm.defaultModel, opts...)
}

// ListAgentTypes 返回所有可用的 Agent 类型
func ListAgentTypes() []AgentType {
	return []AgentType{
		AgentTypeAssistant,
		AgentTypeCoder,
		AgentTypeResearcher,
		AgentTypeAnalyst,
		AgentTypePlanner,
	}
}

// ParseAgentType 解析字符串为 AgentType
func ParseAgentType(s string) (AgentType, bool) {
	t := AgentType(strings.ToLower(s))
	switch t {
	case AgentTypeAssistant, AgentTypeCoder, AgentTypeResearcher,
		AgentTypeAnalyst, AgentTypePlanner, AgentTypeDeep:
		return t, true
	default:
		return AgentTypeAssistant, false
	}
}

// MustParseAgentType 解析字符串为 AgentType，失败返回默认
func MustParseAgentType(s string) AgentType {
	t, ok := ParseAgentType(s)
	if !ok {
		logx.Errorf("unknown agent type: %s, using default: assistant", s)
		return AgentTypeAssistant
	}
	return t
}
