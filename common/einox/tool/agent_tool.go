package tool

import (
	"context"

	"zero-service/common/einox/agent"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
)

// NewAgentTool 将 einox Agent 包装为工具，供父 Agent 调用
//
// 示例：
//
//	// 创建子 Agent
//	subAgent, _ := agent.New(ctx,
//	    agent.WithName("researcher"),
//	    agent.WithInstruction("你是一个研究助手..."),
//	    agent.WithModel(model),
//	)
//
//	// 包装为工具
//	agentTool := tool.NewAgentTool(ctx, subAgent)
//
//	// 在父 Agent 中使用
//	parent, _ := agent.New(ctx,
//	    agent.WithTools(agentTool),
//	)
func NewAgentTool(ctx context.Context, a *agent.Agent) tool.BaseTool {
	return adk.NewAgentTool(ctx, a.GetAgent())
}
