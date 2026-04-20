package modes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/modeweb"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// workflowBlueprint 三种 ADK Workflow（Sequential / Parallel / Loop），子 Agent 均为 planner + summarizer。
type workflowBlueprint struct {
	mode     aisolo.AgentMode
	topology string // sequential | parallel | loop
}

func (b *workflowBlueprint) Mode() aisolo.AgentMode { return b.mode }

func (b *workflowBlueprint) Info() *aisolo.ModeInfo {
	switch b.topology {
	case "parallel":
		return &aisolo.ModeInfo{
			Mode:        b.mode,
			Name:        "Workflow（并行）",
			Description: "Parallel 编排: Planner 与 Summarizer 并行执行后由框架合并；子任务应相互独立。",
			Capabilities: []string{
				"并行编排", "任务拆解", "结论汇总",
			},
		}
	case "loop":
		return &aisolo.ModeInfo{
			Mode:        b.mode,
			Name:        "Workflow（循环）",
			Description: "Loop 编排: Planner → Summarizer 反复迭代直至终止；注意迭代成本与 Checkpoint。",
			Capabilities: []string{
				"循环编排", "多轮 refine", "结论汇总",
			},
		}
	default:
		return &aisolo.ModeInfo{
			Mode:        b.mode,
			Name:        "Workflow（顺序）",
			Description: "Sequential 编排: 先由 Planner 拆分任务, 再由 Summarizer 汇总成最终回复。",
			Capabilities: []string{
				"顺序编排", "任务拆解", "结论汇总",
			},
		}
	}
}

func (b *workflowBlueprint) Build(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error) {
	if err := checkModel(deps); err != nil {
		return nil, err
	}

	computeTools := tool.NewPolicy().AllowCapabilities(tool.CapCompute).Apply(deps.Kit)
	allTools := mergeTools(tool.NewPolicy().
		AllowCapabilities(tool.CapCompute, tool.CapIO, tool.CapHuman).
		Apply(deps.Kit), deps.KnowledgeTools)

	planner, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel, appendSkillDirOpts(deps,
		einoxagent.WithName("workflow-planner"),
		einoxagent.WithDescription("Plan the task into steps, no external side effects."),
		einoxagent.WithInstruction(workflowPrompt),
		einoxagent.WithTools(computeTools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)...)
	if err != nil {
		return nil, fmt.Errorf("modes: build workflow planner: %w", err)
	}

	summarizer, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel, appendSkillDirOpts(deps,
		einoxagent.WithName("workflow-summarizer"),
		einoxagent.WithDescription("Summarize previous steps into the final answer for the user."),
		einoxagent.WithInstruction(workflowSummarizerPrompt),
		einoxagent.WithTools(allTools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)...)
	if err != nil {
		return nil, fmt.Errorf("modes: build workflow summarizer: %w", err)
	}

	subs := []adk.Agent{planner.GetAgent(), summarizer.GetAgent()}
	common := []einoxagent.Option{
		einoxagent.WithSubAgents(subs...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	}

	switch b.topology {
	case "parallel":
		return einoxagent.NewParallelAgent(ctx, append(common,
			einoxagent.WithName("workflow-parallel"),
			einoxagent.WithDescription("Parallel Workflow: planner || summarizer"),
		)...)
	case "loop":
		return einoxagent.NewLoopAgent(ctx, append(common,
			einoxagent.WithName("workflow-loop"),
			einoxagent.WithDescription("Loop Workflow: planner -> summarizer (iterative)"),
			einoxagent.WithMaxIterations(8),
		)...)
	default:
		return einoxagent.NewSequentialAgent(ctx, append(common,
			einoxagent.WithName("workflow-sequential"),
			einoxagent.WithDescription("Sequential Workflow: planner -> summarizer"),
		)...)
	}
}

func workflowBlueprints() []Blueprint {
	modes := modeweb.WorkflowAgentModes()
	out := make([]Blueprint, 0, len(modes))
	for _, m := range modes {
		top := modeweb.Topology(m)
		if top == "" {
			continue
		}
		out = append(out, &workflowBlueprint{mode: m, topology: top})
	}
	return out
}
