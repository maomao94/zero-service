package router

import (
	"context"
	"strings"

	"zero-service/common/einox"
)

// =============================================================================
// SmartRouter 智能路由（基于 eino Graph 编排）
// =============================================================================

// SmartRouter 智能路由编排器
// 使用 eino Graph 实现动态路由决策
type SmartRouter struct {
	classifier *IntentClassifier
	agents     map[string]einox.AgentInterface // 可用的 Agent
	store      CheckPointStore                 // 检查点存储
}

// RouterConfig 路由配置
type RouterConfig struct {
	EnableLLM       bool            // 是否启用 LLM 分类
	SimpleThreshold int             // 简单路由阈值
	CheckPointStore CheckPointStore // 检查点存储
}

// NewSmartRouter 创建智能路由
func NewSmartRouter(cfg *RouterConfig) (*SmartRouter, error) {
	r := &SmartRouter{
		agents: make(map[string]einox.AgentInterface),
		store:  cfg.CheckPointStore,
	}

	if cfg.EnableLLM {
		// LLM 分类器需要外部设置
	}

	return r, nil
}

// RegisterAgent 注册 Agent
func (r *SmartRouter) RegisterAgent(name string, a einox.AgentInterface) {
	r.agents[name] = a
}

// SetClassifier 设置分类器
func (r *SmartRouter) SetClassifier(classifier *IntentClassifier) {
	r.classifier = classifier
}

// Route 执行路由决策
func (r *SmartRouter) Route(ctx context.Context, query string) (*IntentResult, error) {
	// 使用分类器进行意图识别
	if r.classifier != nil {
		return r.classifier.Classify(ctx, query)
	}

	// 降级：使用简单分类
	return r.simpleClassify(query), nil
}

// simpleClassify 简单意图分类
func (r *SmartRouter) simpleClassify(query string) *IntentResult {
	query = strings.ToLower(query)

	// 复杂任务关键词
	complexKeywords := []string{
		"规划", "计划", "分析", "报告", "文档", "设计", "架构",
		"研究", "详细", "深度", "复杂", "多个", "协作", "团队",
		"并行", "同时", "执行", "操作", "修改", "创建", "开发",
	}

	hasComplex := false
	for _, kw := range complexKeywords {
		if strings.Contains(query, kw) {
			hasComplex = true
			break
		}
	}

	if hasComplex {
		return &IntentResult{
			Intent:     "deep",
			Confidence: 0.8,
			Reasoning:  "检测到复杂任务关键词",
		}
	}

	return &IntentResult{
		Intent:     "fast",
		Confidence: 0.9,
		Reasoning:  "简单问答",
	}
}

// GetAgent 获取指定类型的 Agent
func (r *SmartRouter) GetAgent(agentType string) (einox.AgentInterface, bool) {
	if a, ok := r.agents[agentType]; ok {
		return a, true
	}
	// 返回默认 Agent
	if a, ok := r.agents["chat_model"]; ok {
		return a, true
	}
	return nil, false
}

// GetCheckPointStore 获取检查点存储
func (r *SmartRouter) GetCheckPointStore() CheckPointStore {
	return r.store
}

// =============================================================================
// 路由辅助函数
// =============================================================================

// IntentToAgentType 意图转换为 Agent 类型
func IntentToAgentType(intent string) string {
	switch intent {
	case "fast":
		return "chat_model"
	case "deep":
		return "deep"
	case "multi":
		return "parallel"
	default:
		return "chat_model"
	}
}

// AgentTypeToIntent Agent 类型转换为意图
func AgentTypeToIntent(agentType string) string {
	switch agentType {
	case "chat_model":
		return "fast"
	case "deep":
		return "deep"
	case "parallel", "sequential", "supervisor":
		return "multi"
	default:
		return "fast"
	}
}

// IsComplexQuery 判断是否为复杂查询
func IsComplexQuery(query string) bool {
	query = strings.ToLower(query)

	complexPatterns := []string{
		"?", "怎么", "如何", "为什么", "什么",
		"请", "帮我", "帮我分析", "帮我规划",
		"详细", "完整", "具体",
	}

	for _, pattern := range complexPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}

	// 长度超过阈值也视为复杂
	return len(query) > 100
}
