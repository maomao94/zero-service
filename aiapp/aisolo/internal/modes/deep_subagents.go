package modes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// buildDefaultDeepSubAgents 构造 Deep 模式默认的两个示例子 Agent：
//   - deep_research: compute + io，承担可自动化的事实与数据步骤；
//   - deep_synthesis: 无工具，承担终稿整合（主控通过 task 工具按上下文委派）。
//
// 与 supervisor 蓝图分工一致：人机交互保留在主 Deep（WithTools 含 human），
// 子 Agent 收窄能力边界，便于模型在 task 描述中稳定选型。
func buildDefaultDeepSubAgents(ctx context.Context, deps Dependencies) ([]adk.Agent, error) {
	computeIO := mergeTools(tool.NewPolicy().AllowCapabilities(tool.CapCompute, tool.CapIO).Apply(deps.Kit), deps.KnowledgeTools)

	research, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel, appendSkillDirOpts(deps,
		einoxagent.WithName("deep_research"),
		einoxagent.WithDescription("Runs compute/io steps: calculator, time, http_get, echo checks. No user-facing interrupts."),
		einoxagent.WithInstruction(deepSubResearchPrompt),
		einoxagent.WithTools(computeIO...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)...)
	if err != nil {
		return nil, fmt.Errorf("modes: deep sub deep_research: %w", err)
	}

	synthesis, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel, appendSkillDirOpts(deps,
		einoxagent.WithName("deep_synthesis"),
		einoxagent.WithDescription("Turns gathered notes into a polished user-facing answer. No tools."),
		einoxagent.WithInstruction(deepSubSynthesisPrompt),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)...)
	if err != nil {
		return nil, fmt.Errorf("modes: deep sub deep_synthesis: %w", err)
	}

	return []adk.Agent{research.GetAgent(), synthesis.GetAgent()}, nil
}
