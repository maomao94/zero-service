package modes

import (
	"context"

	"zero-service/aiapp/aisolo/aisolo"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// planBlueprint 走 eino prebuilt/planexecute.
type planBlueprint struct{}

func (*planBlueprint) Mode() aisolo.AgentMode { return aisolo.AgentMode_AGENT_MODE_PLAN }

func (*planBlueprint) Info() *aisolo.ModeInfo {
	return &aisolo.ModeInfo{
		Mode:        aisolo.AgentMode_AGENT_MODE_PLAN,
		Name:        "Plan 模式",
		Description: "Plan-Execute-Replan 循环, 适合可分解的复杂任务。",
		Capabilities: []string{
			"任务分解", "动态重规划", "工具调用",
		},
	}
}

func (*planBlueprint) Build(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error) {
	if err := checkModel(deps); err != nil {
		return nil, err
	}

	tools := tool.NewPolicy().
		AllowCapabilities(tool.CapCompute, tool.CapIO, tool.CapHuman).
		Apply(deps.Kit)

	return einoxagent.NewPlanExecuteAgent(ctx, deps.ChatModel,
		einoxagent.WithName("plan"),
		einoxagent.WithDescription("Plan-Execute-Replan agent."),
		einoxagent.WithInstruction(planPrompt),
		einoxagent.WithTools(tools...),
		einoxagent.WithMaxIterations(10),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
}
