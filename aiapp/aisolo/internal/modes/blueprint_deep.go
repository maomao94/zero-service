package modes

import (
	"context"
	"fmt"

	"zero-service/aiapp/aisolo/aisolo"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/tool"
)

// deepBlueprint 使用 eino prebuilt/deep Agent, 配合本地 FileSystem + WriteTodos。
type deepBlueprint struct{}

func (*deepBlueprint) Mode() aisolo.AgentMode { return aisolo.AgentMode_AGENT_MODE_DEEP }

func (*deepBlueprint) Info() *aisolo.ModeInfo {
	return &aisolo.ModeInfo{
		Mode:        aisolo.AgentMode_AGENT_MODE_DEEP,
		Name:        "Deep 模式",
		Description: "深度 Agent, 内置 WriteTodos + FileSystem + 子 Agent 委派, 适合长程研究。",
		Capabilities: []string{
			"任务规划", "文件操作", "深度思考", "人机交互",
		},
	}
}

func (*deepBlueprint) Build(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error) {
	if err := checkModel(deps); err != nil {
		return nil, err
	}

	tools := mergeTools(tool.NewPolicy().
		AllowCapabilities(tool.CapCompute, tool.CapIO, tool.CapHuman).
		Apply(deps.Kit), deps.KnowledgeTools)

	subs, err := buildDefaultDeepSubAgents(ctx, deps)
	if err != nil {
		return nil, fmt.Errorf("modes: deep subagents: %w", err)
	}

	return einoxagent.NewDeepAgent(ctx, deps.ChatModel,
		einoxagent.WithName("deep"),
		einoxagent.WithDescription("Deep agent: todos, filesystem, tools, and task-delegation to deep_research / deep_synthesis subagents."),
		einoxagent.WithInstruction(deepPrompt),
		einoxagent.WithTools(tools...),
		einoxagent.WithSubAgents(subs...),
		einoxagent.WithMaxIterations(30),
		einoxagent.WithEnableWriteTodos(true),
		einoxagent.WithEnableFileSystem(deps.DeepEnableLocalFilesystem),
		einoxagent.WithDeepFilesystem(deps.DeepFSConfig),
		einoxagent.WithSkillsDir(deps.SkillsDir),
		einoxagent.WithCheckPointStore(deps.CheckPointStore),
	)
}
