package router

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"

	"zero-service/aiapp/aisolo/internal/agent"
)

// =============================================================================
// Agent 类型（使用字符串类型，与 agent 包保持一致）
// =============================================================================

// AgentType Agent 类型（别名）
type AgentType = agent.AgentType

// 预定义的 Agent 类型常量
const (
	AgentTypeChatModel   = agent.AgentTypeChatModel
	AgentTypeSequential  = agent.AgentTypeSequential
	AgentTypeLoop        = agent.AgentTypeLoop
	AgentTypeParallel    = agent.AgentTypeParallel
	AgentTypeSupervisor  = agent.AgentTypeSupervisor
	AgentTypePlanExecute = agent.AgentTypePlanExecute
	AgentTypeDeep        = agent.AgentTypeDeep
	AgentTypeCustom      = agent.AgentTypeCustom
)

// ParseAgentType 解析 Agent 类型字符串
func ParseAgentType(s string) AgentType {
	switch s {
	case "chat_model", "AGENT_TYPE_CHAT_MODEL":
		return AgentTypeChatModel
	case "sequential", "AGENT_TYPE_SEQUENTIAL":
		return AgentTypeSequential
	case "loop", "AGENT_TYPE_LOOP":
		return AgentTypeLoop
	case "parallel", "AGENT_TYPE_PARALLEL":
		return AgentTypeParallel
	case "supervisor", "AGENT_TYPE_SUPERVISOR":
		return AgentTypeSupervisor
	case "plan_execute", "AGENT_TYPE_PLAN_EXECUTE":
		return AgentTypePlanExecute
	case "deep", "AGENT_TYPE_DEEP":
		return AgentTypeDeep
	case "custom", "AGENT_TYPE_CUSTOM":
		return AgentTypeCustom
	default:
		return AgentTypeChatModel
	}
}

// =============================================================================
// 路由策略
// =============================================================================

// RouterStrategy 路由策略
type RouterStrategy string

const (
	RouterStrategyAuto   RouterStrategy = "auto"   // 自动路由（两级）
	RouterStrategySimple RouterStrategy = "simple" // 简单路由
	RouterStrategyLLM    RouterStrategy = "llm"    // LLM 意图路由
	RouterStrategyManual RouterStrategy = "manual" // 手动指定
)

// =============================================================================
// 路由决策
// =============================================================================

// Decision 路由决策
type Decision struct {
	Strategy      RouterStrategy `json:"strategy"`
	SelectedAgent AgentType      `json:"selectedAgent"`
	Reason        string         `json:"reason"`
	Confidence    float64        `json:"confidence"`
	Candidates    []AgentType    `json:"candidates"`
}

// Router 智能路由器接口
type Router interface {
	Route(ctx context.Context, query string, opts ...RouteOption) (*Decision, error)
}

// RouteOption 路由选项
type RouteOption func(*RouteOptions)

// RouteOptions 路由选项
type RouteOptions struct {
	UserID        string
	SessionID     string
	History       []*schema.Message
	ForceAgent    AgentType
	ForceStrategy RouterStrategy
}

// WithUserID 设置用户 ID
func WithUserID(userID string) RouteOption {
	return func(o *RouteOptions) { o.UserID = userID }
}

// WithSessionID 设置会话 ID
func WithSessionID(sessionID string) RouteOption {
	return func(o *RouteOptions) { o.SessionID = sessionID }
}

// WithHistory 设置历史消息
func WithHistory(history []*schema.Message) RouteOption {
	return func(o *RouteOptions) { o.History = history }
}

// WithForceAgent 强制指定 Agent
func WithForceAgent(agent AgentType) RouteOption {
	return func(o *RouteOptions) { o.ForceAgent = agent }
}

// WithForceStrategy 强制指定策略
func WithForceStrategy(strategy RouterStrategy) RouteOption {
	return func(o *RouteOptions) { o.ForceStrategy = strategy }
}

// =============================================================================
// ChatModel LLM 接口
// =============================================================================

// ChatModel LLM 接口
type ChatModel interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...interface{}) (*schema.Message, error)
}

// =============================================================================
// TwoLevelRouter 两级路由器
// =============================================================================

// TwoLevelRouter 两级路由器
// 简单问题 -> ChatModel，复杂问题 -> LLM 意图路由
type TwoLevelRouter struct {
	simpleThreshold int
	llm             ChatModel
	llmPrompt       string
}

// NewTwoLevelRouter 创建两级路由器
func NewTwoLevelRouter(opts ...RouterOption) *TwoLevelRouter {
	r := &TwoLevelRouter{
		simpleThreshold: 50,
		llmPrompt:       DefaultLLMRouterPrompt,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RouterOption 路由器选项
type RouterOption func(*TwoLevelRouter)

// WithSimpleThreshold 设置简单路由阈值
func WithSimpleThreshold(threshold int) RouterOption {
	return func(r *TwoLevelRouter) { r.simpleThreshold = threshold }
}

// WithLLM 设置 LLM 模型
func WithLLM(llm ChatModel) RouterOption {
	return func(r *TwoLevelRouter) { r.llm = llm }
}

// WithLLMPrompt 设置 LLM 路由 Prompt
func WithLLMPrompt(prompt string) RouterOption {
	return func(r *TwoLevelRouter) { r.llmPrompt = prompt }
}

// Route 路由决策
func (r *TwoLevelRouter) Route(ctx context.Context, query string, opts ...RouteOption) (*Decision, error) {
	options := &RouteOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 手动指定
	if options.ForceStrategy == RouterStrategyManual && options.ForceAgent != "" {
		return &Decision{
			Strategy:      RouterStrategyManual,
			SelectedAgent: options.ForceAgent,
			Reason:        "用户手动指定",
			Confidence:    1.0,
		}, nil
	}

	// 强制简单路由
	if options.ForceStrategy == RouterStrategySimple {
		return r.simpleRoute(query), nil
	}

	// 强制 LLM 路由
	if options.ForceStrategy == RouterStrategyLLM {
		return r.llmRoute(ctx, query)
	}

	// 自动路由（两级）
	// 第一级：简单判断
	if len(query) < r.simpleThreshold && !isComplexQuery(query) {
		return r.simpleRoute(query), nil
	}

	// 第二级：LLM 意图路由
	if r.llm != nil {
		return r.llmRoute(ctx, query)
	}

	// 降级到简单路由
	return r.simpleRoute(query), nil
}

// simpleRoute 简单路由
func (r *TwoLevelRouter) simpleRoute(query string) *Decision {
	agentType := AgentTypeChatModel
	reason := "简单查询，使用 ChatModel Agent"

	// 基于关键词的简单判断
	if isComplexQuery(query) {
		agentType = AgentTypeDeep
		reason = "检测到复杂任务关键词，使用 Deep Agent"
	}

	return &Decision{
		Strategy:      RouterStrategySimple,
		SelectedAgent: agentType,
		Reason:        reason,
		Confidence:    0.8,
	}
}

// llmRoute LLM 意图路由
func (r *TwoLevelRouter) llmRoute(ctx context.Context, query string) (*Decision, error) {
	if r.llm == nil {
		return r.simpleRoute(query), nil
	}

	prompt := r.llmPrompt
	prompt = strings.ReplaceAll(prompt, "{query}", query)

	// 调用 LLM
	resp, err := r.llm.Generate(ctx, []*schema.Message{
		schema.UserMessage(prompt),
	})
	if err != nil {
		// 降级到简单路由
		return r.simpleRoute(query), nil
	}

	// 解析结果
	return r.parseLLMResponse(resp.Content), nil
}

// parseLLMResponse 解析 LLM 路由响应
func (r *TwoLevelRouter) parseLLMResponse(content string) *Decision {
	var result struct {
		AgentType  string  `json:"agent_type"`
		Reason     string  `json:"reason"`
		Confidence float64 `json:"confidence"`
	}

	// 提取 JSON
	jsonStr := extractJSON(content)
	if jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return &Decision{
				SelectedAgent: ParseAgentType(result.AgentType),
				Reason:        result.Reason,
				Confidence:    result.Confidence,
			}
		}
	}

	// 默认返回
	return &Decision{
		SelectedAgent: AgentTypeChatModel,
		Reason:        "无法解析 LLM 路由结果，使用默认 Agent",
		Confidence:    0.5,
	}
}

// isComplexQuery 判断是否是复杂查询
func isComplexQuery(query string) bool {
	complexKeywords := []string{
		"规划", "计划", "步骤", "分析", "报告", "文档",
		"文件", "读取", "写入", "创建", "修改",
		"多个", "协作", "专家", "团队",
		"复杂", "详细", "完整", "帮我", "请帮我",
	}

	queryLower := strings.ToLower(query)
	for _, keyword := range complexKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}

	// 多个问号
	if strings.Count(query, "?") > 1 {
		return true
	}

	return false
}

// extractJSON 从文本中提取 JSON
func extractJSON(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(text, "}")
	if end == -1 || end < start {
		return ""
	}
	return text[start : end+1]
}

// DefaultLLMRouterPrompt 默认 LLM 路由 Prompt
const DefaultLLMRouterPrompt = `分析用户请求，选择最合适的 Agent 类型。

可选 Agent 类型：
- chat_model: 简单问答、工具调用、单步任务
- deep: 复杂任务规划、文件操作、多步骤任务
- sequential: 顺序执行多个步骤
- parallel: 并行执行多个任务
- supervisor: 多专家协作，动态分配任务
- plan_execute: 规划-执行-重规划循环

用户请求：{query}

请返回 JSON 格式结果：
{"agent_type": "xxx", "reason": "选择原因", "confidence": 0.9}`
