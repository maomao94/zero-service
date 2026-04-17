package modes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	ctool "github.com/cloudwego/eino/components/tool"
)

// NewSurveyEchoAgentTool 将「问卷 ask_form_input → echo」整条顺序流封装为 **一个工具**（Eino: adk.NewAgentTool）。
// 父 Agent 通过 function calling 调用时，子流在独立消息上下文中执行。
//
// TODO(业务上线前删除): 与 blueprint_agent 中挂载一并移除；删除 BuildSurveyEchoAgent、prompts 中 surveyEcho*、skills/survey_echo_flow。
func NewSurveyEchoAgentTool(ctx context.Context, deps Dependencies) (ctool.BaseTool, error) {
	ag, err := BuildSurveyEchoAgent(ctx, deps)
	if err != nil {
		return nil, fmt.Errorf("modes: survey echo agent tool: %w", err)
	}
	return adk.NewAgentTool(ctx, ag.GetAgent()), nil
}
