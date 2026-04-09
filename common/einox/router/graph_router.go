package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"

	"zero-service/common/einox"
)

// =============================================================================
// SmartGraphRouter - 基于 eino Graph 的智能路由
// =============================================================================

// SmartGraphRouter 基于 eino Graph 编排的智能路由
type SmartGraphRouter struct {
	graph        compose.Runnable[string, string]
	agents       map[string]einox.AgentInterface
	store        CheckPointStore
	classifier   *IntentClassifier
	agentNames   []string
	defaultAgent string
}

// GraphRouterConfig Graph 路由配置
type GraphRouterConfig struct {
	EnableLLM       bool
	CheckPointStore CheckPointStore
	Classifier      *IntentClassifier
	Agents          map[string]einox.AgentInterface
	DefaultAgent    string // 默认 Agent 类型
}

// NewSmartGraphRouter 创建基于 Graph 的智能路由
func NewSmartGraphRouter(cfg *GraphRouterConfig) (*SmartGraphRouter, error) {
	r := &SmartGraphRouter{
		agents:       make(map[string]einox.AgentInterface),
		store:        cfg.CheckPointStore,
		classifier:   cfg.Classifier,
		defaultAgent: cfg.DefaultAgent,
	}

	if r.defaultAgent == "" {
		r.defaultAgent = "chat_model"
	}

	// 复制 Agent 配置
	for k, v := range cfg.Agents {
		r.agents[k] = v
		r.agentNames = append(r.agentNames, k)
	}

	// 构建 Graph
	g, err := r.buildGraph()
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	r.graph = g
	return r, nil
}

// RegisterAgent 注册 Agent
func (r *SmartGraphRouter) RegisterAgent(name string, agent einox.AgentInterface) {
	r.agents[name] = agent
	r.agentNames = append(r.agentNames, name)
}

// Route 执行路由决策，返回意图结果
func (r *SmartGraphRouter) Route(ctx context.Context, query string) (*IntentResult, error) {
	result := r.classify(query)
	agentType := r.selectAgent(result)

	result.SelectedAgent = agentType
	return result, nil
}

// selectAgent 根据意图选择 Agent
func (r *SmartGraphRouter) selectAgent(result *IntentResult) string {
	agentType := IntentToAgentType(result.Intent)

	// 检查 Agent 是否存在
	if _, ok := r.agents[agentType]; ok {
		return agentType
	}

	// 回退到默认 Agent
	if _, ok := r.agents[r.defaultAgent]; ok {
		return r.defaultAgent
	}

	// 回退到第一个注册的 Agent
	for name := range r.agents {
		return name
	}

	return r.defaultAgent
}

// classify 意图分类
func (r *SmartGraphRouter) classify(query string) *IntentResult {
	if r.classifier != nil {
		result, err := r.classifier.Classify(context.Background(), query)
		if err == nil {
			return result
		}
	}
	return r.simpleClassify(query)
}

// simpleClassify 简单分类
func (r *SmartGraphRouter) simpleClassify(query string) *IntentResult {
	query = strings.ToLower(query)

	complexKeywords := []string{
		"规划", "计划", "分析", "报告", "文档", "设计", "架构",
		"研究", "详细", "深度", "复杂", "多个", "协作", "团队",
		"并行", "同时", "执行", "操作", "修改", "创建", "开发",
	}

	for _, kw := range complexKeywords {
		if strings.Contains(query, kw) {
			return &IntentResult{
				Intent:     "deep",
				Confidence: 0.8,
				Reasoning:  "检测到复杂任务关键词",
			}
		}
	}

	return &IntentResult{
		Intent:     "fast",
		Confidence: 0.9,
		Reasoning:  "简单问答",
	}
}

// Run 使用 Graph 执行
func (r *SmartGraphRouter) Run(ctx context.Context, query string) (string, error) {
	if r.graph == nil {
		return "", fmt.Errorf("graph not initialized")
	}

	output, err := r.graph.Invoke(ctx, query)
	if err != nil {
		return "", fmt.Errorf("graph run: %w", err)
	}

	return output, nil
}

// routeDecision 路由决策 Lambda
func (r *SmartGraphRouter) routeDecision(ctx context.Context, query string) (string, error) {
	result, err := r.Route(ctx, query)
	if err != nil {
		return "", err
	}
	return result.SelectedAgent, nil
}

// executeAgent 执行指定 Agent 的 Lambda
func (r *SmartGraphRouter) executeAgent(agentName string) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
		agent, ok := r.agents[agentName]
		if !ok {
			return "", fmt.Errorf("agent %s not found", agentName)
		}

		result, err := agent.Run(ctx, input)
		if err != nil {
			return "", fmt.Errorf("agent run: %w", err)
		}

		return result.Response, nil
	})
}

// buildGraph 构建 Graph
func (r *SmartGraphRouter) buildGraph() (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string]()

	// 入口节点：接收用户输入
	g.AddPassthroughNode("input")

	// 意图分类节点
	classifyLambda := compose.InvokableLambda(func(ctx context.Context, query string) (*IntentResult, error) {
		return r.classify(query), nil
	})
	err := g.AddLambdaNode("classifier", classifyLambda,
		compose.WithInputKey("query"),
		compose.WithOutputKey("intent"))
	if err != nil {
		return nil, fmt.Errorf("add classifier node: %w", err)
	}

	// 路由决策节点：基于意图选择 Agent
	routeLambda := compose.InvokableLambda(func(ctx context.Context, intent *IntentResult) (string, error) {
		agentType := r.selectAgent(intent)
		return agentType, nil
	})
	err = g.AddLambdaNode("router", routeLambda,
		compose.WithInputKey("intent"),
		compose.WithOutputKey("agent"))
	if err != nil {
		return nil, fmt.Errorf("add router node: %w", err)
	}

	// 添加分支：基于路由决策动态选择 Agent 节点
	branch := compose.NewGraphBranch(
		func(ctx context.Context, agentName string) (string, error) {
			// 检查 Agent 是否存在
			if _, ok := r.agents[agentName]; !ok {
				return r.defaultAgent, nil
			}
			return "agent_" + agentName, nil
		},
		r.buildEndNodes(),
	)
	err = g.AddBranch("router", branch)
	if err != nil {
		return nil, fmt.Errorf("add branch: %w", err)
	}

	// 动态添加 Agent 节点
	for agentType := range r.agents {
		nodeName := "agent_" + agentType
		ag := agentType // 捕获变量

		agentLambda := compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
			agent, ok := r.agents[ag]
			if !ok {
				return "", fmt.Errorf("agent %s not found", ag)
			}

			result, err := agent.Run(ctx, input)
			if err != nil {
				return "", fmt.Errorf("agent run: %w", err)
			}

			return result.Response, nil
		})

		err = g.AddLambdaNode(nodeName, agentLambda,
			compose.WithInputKey("query"),
			compose.WithOutputKey("response"))
		if err != nil {
			return nil, fmt.Errorf("add agent node %s: %w", nodeName, err)
		}

		// 连接 Agent 到输出
		g.AddEdge(nodeName, "output")
	}

	// 输出节点
	outputLambda := compose.InvokableLambda(func(ctx context.Context, response string) (string, error) {
		return response, nil
	})
	err = g.AddLambdaNode("output", outputLambda,
		compose.WithInputKey("response"))
	if err != nil {
		return nil, fmt.Errorf("add output node: %w", err)
	}

	// 连接边
	g.AddEdge(compose.START, "input")
	g.AddEdge("input", "classifier")
	g.AddEdge("classifier", "router")
	g.AddEdge("output", compose.END)

	// 编译 Graph
	return g.Compile(context.Background())
}

// buildEndNodes 构建分支终点节点映射
func (r *SmartGraphRouter) buildEndNodes() map[string]bool {
	endNodes := make(map[string]bool)
	for agentType := range r.agents {
		endNodes["agent_"+agentType] = true
	}
	// 确保默认节点也在终点中
	if defaultNode := "agent_" + r.defaultAgent; endNodes[defaultNode] {
		endNodes[defaultNode] = true
	} else {
		// 如果默认 Agent 不存在，使用第一个注册的 Agent
		for name := range r.agents {
			endNodes["agent_"+name] = true
			break
		}
	}
	return endNodes
}

// =============================================================================
// StreamGraphRouter - 支持流式输出的 Graph 路由
// =============================================================================

// StreamGraphRouter 支持流式输出的智能路由
type StreamGraphRouter struct {
	*SmartGraphRouter
	streamAgents map[string]StreamAgent
}

// StreamAgent 支持流式输出的 Agent
type StreamAgent interface {
	RunStream(ctx context.Context, query string, opts ...einox.RunOption) (<-chan *einox.AgentResult, error)
}

// NewStreamGraphRouter 创建流式 Graph 路由
func NewStreamGraphRouter(cfg *GraphRouterConfig) (*StreamGraphRouter, error) {
	sr, err := NewSmartGraphRouter(cfg)
	if err != nil {
		return nil, err
	}

	return &StreamGraphRouter{
		SmartGraphRouter: sr,
		streamAgents:     make(map[string]StreamAgent),
	}, nil
}

// RegisterStreamAgent 注册流式 Agent
func (r *StreamGraphRouter) RegisterStreamAgent(name string, agent StreamAgent) {
	r.streamAgents[name] = agent
}

// RunStream 流式执行
func (r *StreamGraphRouter) RunStream(ctx context.Context, query string, opts ...einox.RunOption) (<-chan *einox.AgentResult, error) {
	// 路由决策
	intent, err := r.Route(ctx, query)
	if err != nil {
		return nil, err
	}

	agentType := intent.SelectedAgent

	// 优先使用流式 Agent
	streamAgent, ok := r.streamAgents[agentType]
	if ok {
		return streamAgent.RunStream(ctx, query, opts...)
	}

	// 回退到普通 Agent
	agent, ok := r.agents[agentType]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentType)
	}
	return agent.RunStream(ctx, query, opts...)
}
