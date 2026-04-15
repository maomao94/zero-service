package logic

import (
	"context"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/einox/agent"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/internal/svc"
)

type ListAgentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAgentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentsLogic {
	return &ListAgentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListAgents 列出可用 Agent
func (l *ListAgentsLogic) ListAgents(in *aisolo.ListAgentsReq) (*aisolo.ListAgentsResp, error) {
	// 返回默认 Agent 类型列表（来自 Builtins）
	agents := getDefaultAgents()

	return &aisolo.ListAgentsResp{
		Agents: agents,
	}, nil
}

// getDefaultAgents 返回默认 Agent 列表
func getDefaultAgents() []*aisolo.AgentInfo {
	return []*aisolo.AgentInfo{
		{
			Id:           string(agent.EinoTypeChatModel),
			Name:         "ChatModelAgent",
			Description:  "ReAct 工具调用 Agent，支持多轮对话和工具调用",
			Capabilities: []string{"工具调用", "多轮对话", "ReAct 推理"},
			Available:    true,
			Tools: []*aisolo.ToolInfo{
				{Name: "calculator", Description: "数学计算器"},
				{Name: "datetime", Description: "日期时间查询"},
				{Name: "echo", Description: "消息回显"},
			},
		},
		{
			Id:           string(agent.EinoTypeDeep),
			Name:         "DeepAgent",
			Description:  "深度规划 Agent，支持任务规划、文件系统和子 Agent 委派",
			Capabilities: []string{"任务规划", "文件操作", "子Agent委派"},
			Available:    true,
			Tools: []*aisolo.ToolInfo{
				{Name: "filesystem", Description: "文件系统操作"},
				{Name: "subagent", Description: "子Agent调用"},
				{Name: "planner", Description: "任务规划"},
			},
		},
		{
			Id:           string(agent.EinoTypeSequential),
			Name:         "SequentialAgent",
			Description:  "顺序执行 Agent，按顺序依次执行子 Agent",
			Capabilities: []string{"顺序执行", "流水线处理"},
			Available:    true,
			Tools: []*aisolo.ToolInfo{
				{Name: "pipeline", Description: "流水线执行"},
				{Name: "step", Description: "步骤控制"},
			},
		},
		{
			Id:           string(agent.EinoTypeParallel),
			Name:         "ParallelAgent",
			Description:  "并行执行 Agent，并发执行多个子 Agent",
			Capabilities: []string{"并行执行", "并发处理"},
			Available:    true,
			Tools: []*aisolo.ToolInfo{
				{Name: "concurrent", Description: "并发执行"},
				{Name: "wait", Description: "等待完成"},
			},
		},
		{
			Id:           string(agent.EinoTypeSupervisor),
			Name:         "SupervisorAgent",
			Description:  "监督者 Agent，协调多个子 Agent 动态分配任务",
			Capabilities: []string{"动态任务分配", "多 Agent 协调"},
			Available:    true,
			Tools: []*aisolo.ToolInfo{
				{Name: "delegate", Description: "任务委托"},
				{Name: "coordinate", Description: "协调调度"},
			},
		},
		{
			Id:           string(agent.EinoTypePlanExecute),
			Name:         "PlanExecuteAgent",
			Description:  "规划-执行 Agent，Planner -> Executor -> Replanner 循环",
			Capabilities: []string{"任务规划", "分步执行", "动态调整"},
			Available:    true,
			Tools: []*aisolo.ToolInfo{
				{Name: "plan", Description: "规划任务"},
				{Name: "execute", Description: "执行任务"},
				{Name: "replan", Description: "重新规划"},
			},
		},
		{
			Id:           string(agent.EinoTypeLoop),
			Name:         "LoopAgent",
			Description:  "循环执行 Agent，循环执行直到条件满足",
			Capabilities: []string{"循环执行", "迭代优化"},
			Available:    true,
			Tools: []*aisolo.ToolInfo{
				{Name: "iterate", Description: "迭代执行"},
				{Name: "condition", Description: "条件判断"},
			},
		},
	}
}
