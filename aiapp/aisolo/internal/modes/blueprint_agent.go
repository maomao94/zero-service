package modes

import (
	"context"

	"zero-service/aiapp/aisolo/aisolo"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// agentBlueprint 默认的 ChatModelAgent + 全部工具 (含 6 种中断工具)。
type agentBlueprint struct{}

func (*agentBlueprint) Mode() aisolo.AgentMode { return aisolo.AgentMode_AGENT_MODE_AGENT }

func (*agentBlueprint) Info() *aisolo.ModeInfo {
	return &aisolo.ModeInfo{
		Mode:        aisolo.AgentMode_AGENT_MODE_AGENT,
		Name:        "Agent 模式",
		Description: "ChatModelAgent + 全工具 ReAct 推理, 默认模式。适合大部分对话场景。",
		Capabilities: []string{
			"工具调用", "ReAct 推理", "人机交互中断",
		},
	}
}

func (*agentBlueprint) Build(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error) {
	if err := checkModel(deps); err != nil {
		return nil, err
	}

	tools := tool.NewPolicy().
		AllowCapabilities(tool.CapCompute, tool.CapIO, tool.CapHuman).
		Apply(deps.Kit)

	// TODO(业务上线前删除): 以下为 Eino 联调演示 —— 将「问卷→echo」顺序流封装为 AgentTool 挂进默认 Agent。
	// 正式业务智能体应移除 NewSurveyEchoAgentTool，仅保留 Kit 内真实业务工具或由配置注入的子 Agent。
	at, err := NewSurveyEchoAgentTool(ctx, deps)
	if err != nil {
		return nil, err
	}
	tools = append(tools, at)

	return einoxagent.NewChatModelAgent(ctx, deps.ChatModel, appendSkillDirOpts(deps,
		einoxagent.WithName("agent"),
		einoxagent.WithDescription("Default ChatModel agent with all built-in tools"),
		einoxagent.WithInstruction(agentPrompt),
		einoxagent.WithTools(tools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)...)
}
