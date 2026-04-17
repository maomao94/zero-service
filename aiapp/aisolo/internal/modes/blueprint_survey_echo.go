package modes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// BuildSurveyEchoAgent 构建「问卷 ask_form_input → echo」顺序工作流（联调 / AgentTool 复用）。
// 已不作为独立 AgentMode 暴露；上线前若不再需要联调，可删除本构建与 prompts 中 surveyEcho* 常量。
func BuildSurveyEchoAgent(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error) {
	if err := checkModel(deps); err != nil {
		return nil, err
	}

	humanTools := tool.NewPolicy().AllowCapabilities(tool.CapHuman).Apply(deps.Kit)
	echoTools := tool.NewPolicy().AllowNames("echo").Apply(deps.Kit)

	planner, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel,
		einoxagent.WithName("survey-echo-planner"),
		einoxagent.WithDescription("Collect user answers via human-interrupt tools."),
		einoxagent.WithInstruction(surveyEchoPlannerPrompt),
		einoxagent.WithTools(humanTools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
	if err != nil {
		return nil, fmt.Errorf("modes: build survey-echo planner: %w", err)
	}

	echoer, err := einoxagent.NewChatModelAgent(ctx, deps.ChatModel,
		einoxagent.WithName("survey-echo-echoer"),
		einoxagent.WithDescription("Echo survey summary via echo tool only."),
		einoxagent.WithInstruction(surveyEchoEchoPrompt),
		einoxagent.WithTools(echoTools...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
	if err != nil {
		return nil, fmt.Errorf("modes: build survey-echo echoer: %w", err)
	}

	return einoxagent.NewSequentialAgent(ctx,
		einoxagent.WithName("survey-echo"),
		einoxagent.WithDescription("Survey form interrupt then echo tool"),
		einoxagent.WithSubAgents([]adk.Agent{planner.GetAgent(), echoer.GetAgent()}...),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
}
