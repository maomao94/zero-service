package modes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	"zero-service/aiapp/aisolo/aisolo"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// supervisorBlueprint 使用 adk supervisor:
//
//	supervisor (LLM, 负责分派)
//	 ├── researcher (compute + io 工具, 检索/计算)
//	 └── interactor (human 工具, 与用户交互)
type supervisorBlueprint struct{}

func (*supervisorBlueprint) Mode() aisolo.AgentMode { return aisolo.AgentMode_AGENT_MODE_SUPERVISOR }

func (*supervisorBlueprint) Info() *aisolo.ModeInfo {
	return &aisolo.ModeInfo{
		Mode:        aisolo.AgentMode_AGENT_MODE_SUPERVISOR,
		Name:        "Supervisor 模式",
		Description: "多 Agent 协作: 研究员负责检索计算, 交互员负责与用户确认, Supervisor 调度。",
		Capabilities: []string{
			"多 Agent 协作", "任务委派", "人机交互",
		},
	}
}

func (*supervisorBlueprint) Build(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error) {
	if err := checkModel(deps); err != nil {
		return nil, err
	}

	computeIO := tool.NewPolicy().AllowCapabilities(tool.CapCompute, tool.CapIO).Apply(deps.Kit)
	humanTools := tool.NewPolicy().AllowCapabilities(tool.CapHuman).Apply(deps.Kit)

	researcher, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel, appendSkillDirOpts(deps,
		einoxagent.WithName("researcher"),
		einoxagent.WithDescription("Do calculation, fetch time, or any compute/io task."),
		einoxagent.WithInstruction(supervisorWorkerPrompt),
		einoxagent.WithTools(computeIO...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)...)
	if err != nil {
		return nil, fmt.Errorf("modes: build supervisor.researcher: %w", err)
	}

	interactor, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel, appendSkillDirOpts(deps,
		einoxagent.WithName("interactor"),
		einoxagent.WithDescription("Interact with the user via approval/select/text/form/info tools."),
		einoxagent.WithInstruction(supervisorWorkerPrompt),
		einoxagent.WithTools(humanTools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)...)
	if err != nil {
		return nil, fmt.Errorf("modes: build supervisor.interactor: %w", err)
	}

	return einoxagent.NewSupervisorAgent(ctx, deps.ChatModel,
		[]adk.Agent{researcher.GetAgent(), interactor.GetAgent()},
		appendSkillDirOpts(deps,
			einoxagent.WithName("supervisor"),
			einoxagent.WithDescription("Supervisor that delegates to researcher / interactor."),
			einoxagent.WithInstruction(supervisorPrompt),
			einoxagent.WithCheckPointStore(deps.CheckPointStore),
		)...,
	)
}
