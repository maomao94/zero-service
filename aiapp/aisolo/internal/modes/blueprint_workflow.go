package modes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	"zero-service/aiapp/aisolo/aisolo"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// workflowBlueprint 使用 Sequential Workflow 串联 2 个 ChatModelAgent:
//
//	planner  (只能用 compute 工具, 负责拆解任务)
//	summarizer (全部工具, 负责最终汇总)
//
// 之所以选 Sequential 而不是 Parallel/Loop 作为默认 Workflow, 是因为 Sequential
// 在用户视角最可解释; Parallel/Loop 可在上层 policy 里替换。
type workflowBlueprint struct{}

func (*workflowBlueprint) Mode() aisolo.AgentMode { return aisolo.AgentMode_AGENT_MODE_WORKFLOW }

func (*workflowBlueprint) Info() *aisolo.ModeInfo {
	return &aisolo.ModeInfo{
		Mode:        aisolo.AgentMode_AGENT_MODE_WORKFLOW,
		Name:        "Workflow 模式",
		Description: "Sequential 编排: 先由 Planner 拆分任务, 再由 Summarizer 汇总成最终回复。",
		Capabilities: []string{
			"顺序编排", "任务拆解", "结论汇总",
		},
	}
}

func (*workflowBlueprint) Build(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error) {
	if err := checkModel(deps); err != nil {
		return nil, err
	}

	computeTools := tool.NewPolicy().AllowCapabilities(tool.CapCompute).Apply(deps.Kit)
	allTools := tool.NewPolicy().
		AllowCapabilities(tool.CapCompute, tool.CapIO, tool.CapHuman).
		Apply(deps.Kit)

	planner, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel,
		einoxagent.WithName("workflow-planner"),
		einoxagent.WithDescription("Plan the task into steps, no external side effects."),
		einoxagent.WithInstruction(workflowPrompt),
		einoxagent.WithTools(computeTools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
	if err != nil {
		return nil, fmt.Errorf("modes: build workflow planner: %w", err)
	}

	summarizer, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel,
		einoxagent.WithName("workflow-summarizer"),
		einoxagent.WithDescription("Summarize previous steps into the final answer for the user."),
		einoxagent.WithInstruction(workflowSummarizerPrompt),
		einoxagent.WithTools(allTools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
	if err != nil {
		return nil, fmt.Errorf("modes: build workflow summarizer: %w", err)
	}

	return einoxagent.NewSequentialAgent(ctx,
		einoxagent.WithName("workflow"),
		einoxagent.WithDescription("Sequential Workflow: planner -> summarizer"),
		einoxagent.WithSubAgents([]adk.Agent{planner.GetAgent(), summarizer.GetAgent()}...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
}
