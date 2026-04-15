package router

import (
	"context"
	"strings"

	"zero-service/aiapp/aisolo/aisolo"
	einoagent "zero-service/aiapp/aisolo/internal/agent"
	einox "zero-service/common/einox/agent"
)

// RequestType 请求类型
type RequestType string

const (
	RequestTypeChat    RequestType = "chat"    // 普通对话
	RequestTypeCode    RequestType = "code"    // 编码任务
	RequestTypeSearch  RequestType = "search"  // 搜索任务
	RequestTypeTool    RequestType = "tool"    // 工具调用
	RequestTypeComplex RequestType = "complex" // 复杂任务
)

// Router 请求路由器
type Router struct {
	agentPool *einoagent.AgentPool
	rules     map[RequestType]string // 请求类型到角色的映射
}

// NewRouter 创建路由器
func NewRouter(agentPool *einoagent.AgentPool) *Router {
	r := &Router{
		agentPool: agentPool,
		rules: map[RequestType]string{
			RequestTypeChat:    "assistant",
			RequestTypeCode:    "coder",
			RequestTypeSearch:  "searcher",
			RequestTypeTool:    "toolmaster",
			RequestTypeComplex: "deepthink",
		},
	}
	return r
}

// Route 根据请求分配Agent
func (r *Router) Route(ctx context.Context, req *aisolo.AskReq) (*einox.Agent, func(), error) {
	reqType := r.classifyRequest(req)

	// 优先使用用户指定的模式
	roleID := r.getRoleIDByMode(req.AgentMode)
	if roleID == "" {
		roleID = r.rules[reqType]
	}

	agent, err := r.agentPool.GetAgent(ctx, roleID, req.AgentMode)
	if err != nil {
		return nil, nil, err
	}

	// 返回归还函数
	cleanup := func() {
		r.agentPool.PutAgent(roleID, req.AgentMode, agent)
	}

	return agent, cleanup, nil
}

// classifyRequest 智能分类请求类型
func (r *Router) classifyRequest(req *aisolo.AskReq) RequestType {
	msg := strings.ToLower(req.Message)

	// 简单规则判断，后续可接入einox智能路由
	switch {
	case strings.Contains(msg, "代码") || strings.Contains(msg, "编程") || strings.Contains(msg, "写个") || strings.Contains(msg, "实现"):
		return RequestTypeCode
	case strings.Contains(msg, "搜索") || strings.Contains(msg, "查找") || strings.Contains(msg, "查询"):
		return RequestTypeSearch
	case strings.Contains(msg, "工具") || strings.Contains(msg, "调用") || strings.Contains(msg, "执行"):
		return RequestTypeTool
	case strings.Contains(msg, "分析") || strings.Contains(msg, "规划") || strings.Contains(msg, "解决") || len(msg) > 200:
		return RequestTypeComplex
	default:
		return RequestTypeChat
	}
}

// getRoleIDByMode 根据AgentMode获取角色ID
func (r *Router) getRoleIDByMode(mode aisolo.AgentMode) string {
	switch mode {
	case aisolo.AgentMode_AGENT_MODE_FAST:
		return "assistant"
	case aisolo.AgentMode_AGENT_MODE_DEEP:
		return "deepthink"
	default:
		return ""
	}
}

// AddRule 添加路由规则
func (r *Router) AddRule(reqType RequestType, roleID string) {
	r.rules[reqType] = roleID
}
