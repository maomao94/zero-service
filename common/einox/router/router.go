package router

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// 路由策略类型
// =============================================================================

// Strategy 路由策略
type Strategy string

const (
	StrategyAuto   Strategy = "auto"   // 自动路由（默认）
	StrategySimple Strategy = "simple" // 简单路由（仅关键词）
	StrategyLLM    Strategy = "llm"    // LLM 路由
	StrategyManual Strategy = "manual" // 手动指定
)

// RouterResponse 路由响应
type RouterResponse struct {
	Content    string             // 响应内容
	AgentType  string             // Agent 类型
	ToolCalls  []*schema.ToolCall // 工具调用
	IsComplete bool               // 是否完成
}

// =============================================================================
// Router 接口
// =============================================================================

// Router 路由器接口
type Router interface {
	// Route 执行路由决策
	Route(ctx context.Context, query string) (*IntentResult, error)

	// RouteWithHistory 基于历史消息执行路由
	RouteWithHistory(ctx context.Context, query string, history []*schema.Message) (*IntentResult, error)
}

// =============================================================================
// TwoLevelRouter 两级路由器
// =============================================================================

// TwoLevelRouter 两级路由实现
type TwoLevelRouter struct {
	classifier *IntentClassifier
	strategy   Strategy
	threshold  int // 简单路由阈值（字符数）
}

// RouteOption 路由选项
type RouteOption func(*TwoLevelRouter)

// WithStrategy 设置路由策略
func WithStrategy(s Strategy) RouteOption {
	return func(r *TwoLevelRouter) {
		r.strategy = s
	}
}

// WithSimpleThreshold 设置简单路由阈值
func WithSimpleThreshold(threshold int) RouteOption {
	return func(r *TwoLevelRouter) {
		r.threshold = threshold
	}
}

// NewTwoLevelRouter 创建两级路由器
func NewTwoLevelRouter(opts ...RouteOption) *TwoLevelRouter {
	r := &TwoLevelRouter{
		strategy:  StrategyAuto,
		threshold: 50,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetClassifier 设置分类器
func (r *TwoLevelRouter) SetClassifier(classifier *IntentClassifier) {
	r.classifier = classifier
}

// Route 执行路由
func (r *TwoLevelRouter) Route(ctx context.Context, query string) (*IntentResult, error) {
	return r.RouteWithHistory(ctx, query, nil)
}

// RouteWithHistory 基于历史执行路由
func (r *TwoLevelRouter) RouteWithHistory(ctx context.Context, query string, history []*schema.Message) (*IntentResult, error) {
	switch r.strategy {
	case StrategySimple:
		return r.simpleRoute(query), nil
	case StrategyLLM:
		return r.llmRoute(ctx, query)
	case StrategyAuto:
		return r.autoRoute(ctx, query)
	default:
		return r.simpleRoute(query), nil
	}
}

// simpleRoute 简单路由（仅关键词）
func (r *TwoLevelRouter) simpleRoute(query string) *IntentResult {
	// 长度判断
	if len(query) < r.threshold {
		return &IntentResult{
			Intent:     "fast",
			Confidence: 0.9,
			Reasoning:  "query length < threshold",
		}
	}

	// 关键词判断
	return r.fallbackClassify(query)
}

// llmRoute LLM 路由
func (r *TwoLevelRouter) llmRoute(ctx context.Context, query string) (*IntentResult, error) {
	if r.classifier == nil {
		return r.simpleRoute(query), nil
	}
	return r.classifier.Classify(ctx, query)
}

// autoRoute 自动路由（先简单后 LLM）
func (r *TwoLevelRouter) autoRoute(ctx context.Context, query string) (*IntentResult, error) {
	// 先尝试简单路由
	result := r.simpleRoute(query)
	if result.Confidence >= 0.9 {
		return result, nil
	}

	// 置信度不够，使用 LLM
	if r.classifier != nil {
		return r.classifier.Classify(ctx, query)
	}

	return result, nil
}

// fallbackClassify 降级分类
func (r *TwoLevelRouter) fallbackClassify(query string) *IntentResult {
	// 导入需要避免循环依赖
	classifier := &IntentClassifier{}
	return classifier.fallbackClassify(query)
}
