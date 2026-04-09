package logic

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

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
func (l *ListAgentsLogic) ListAgents(in *aisolo.ListAgentsRequest) (*aisolo.ListAgentsResponse, error) {
	var agents []*aisolo.AgentInfo

	// 从 AgentManager 获取
	if l.svcCtx.AgentManager != nil {
		agentInfos := l.svcCtx.AgentManager.List()
		for _, info := range agentInfos {
			agents = append(agents, &aisolo.AgentInfo{
				Id:           info.Type,
				Name:         info.Name,
				Description:  info.Description,
				Capabilities: info.Capabilities,
				Available:    info.Available,
			})
		}
	} else {
		// 返回默认 Agent 类型
		agents = getDefaultAgents()
	}

	return &aisolo.ListAgentsResponse{
		Agents: agents,
	}, nil
}

// getDefaultAgents 返回默认 Agent 列表
func getDefaultAgents() []*aisolo.AgentInfo {
	return []*aisolo.AgentInfo{
		{
			Id:           "chat_model",
			Name:         "ChatModelAgent",
			Description:  "ReAct 工具调用 Agent，支持多轮对话和工具调用",
			Capabilities: []string{"工具调用", "多轮对话", "ReAct 推理"},
			Available:    true,
		},
		{
			Id:           "deep",
			Name:         "DeepAgent",
			Description:  "深度规划 Agent，支持任务规划、文件系统和子 Agent 委派",
			Capabilities: []string{"任务规划", "文件操作", "子Agent委派"},
			Available:    true,
		},
		{
			Id:           "sequential",
			Name:         "SequentialAgent",
			Description:  "顺序执行 Agent，按顺序依次执行子 Agent",
			Capabilities: []string{"顺序执行", "流水线处理"},
			Available:    true,
		},
		{
			Id:           "parallel",
			Name:         "ParallelAgent",
			Description:  "并行执行 Agent，并发执行多个子 Agent",
			Capabilities: []string{"并行执行", "并发处理"},
			Available:    true,
		},
		{
			Id:           "supervisor",
			Name:         "SupervisorAgent",
			Description:  "监督者 Agent，协调多个子 Agent 动态分配任务",
			Capabilities: []string{"动态任务分配", "多 Agent 协调"},
			Available:    true,
		},
		{
			Id:           "plan_execute",
			Name:         "PlanExecuteAgent",
			Description:  "规划-执行 Agent，Planner -> Executor -> Replanner 循环",
			Capabilities: []string{"任务规划", "分步执行", "动态调整"},
			Available:    true,
		},
		{
			Id:           "loop",
			Name:         "LoopAgent",
			Description:  "循环执行 Agent，循环执行直到条件满足",
			Capabilities: []string{"循环执行", "迭代优化"},
			Available:    true,
		},
	}
}
