// Package modes 定义 aisolo 对用户暴露的 Mode, 以及根据 Mode 构造 Agent
// 的 Blueprint。
//
// 设计要点：
//
//  1. 用户只挑 Mode, 不挑 Agent。每个 Mode 在内部可能由多个 Agent 组合而成
//     (比如 Workflow 是 Sequential + 多个子 ChatModelAgent)。
//  2. Blueprint 是轻量配方, 不持有任何状态, 每次 turn 都会按 Blueprint 现场
//     构建 Agent 实例 (实际是通过 AgentPool 做缓存复用)。
//  3. Blueprint 通过 Dependencies 拿到共享资源 (ChatModel / Kit / CheckPoint)。
package modes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"

	"zero-service/aiapp/aisolo/aisolo"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/fsrestrict"
	"zero-service/common/einox/tool"
)

// Dependencies 构造 Agent 所需的全部共享资源。
type Dependencies struct {
	ChatModel       model.BaseChatModel
	Kit             *tool.Kit
	CheckPointStore adk.CheckPointStore
	// SkillsDir 为已解析的 skill 根目录绝对路径；空表示不加载 skill 中间件。
	SkillsDir string
	// DeepEnableLocalFilesystem 为 true 时 Deep 模式挂载 Eino 本地文件系统工具（grep 等）。
	DeepEnableLocalFilesystem bool
	// DeepFSConfig Deep 本地文件沙箱（用户根、会话父目录、策略）；零值表示不限制路径。
	DeepFSConfig fsrestrict.Config
	// PlanMaxIterations PlanExecute 模式最大迭代（默认 10）。
	PlanMaxIterations int
}

// Blueprint 描述一个 Mode 的构造方式。
type Blueprint interface {
	Mode() aisolo.AgentMode
	Info() *aisolo.ModeInfo
	Build(ctx context.Context, deps Dependencies) (*einoxagent.Agent, error)
}

// Registry 聚合所有 Blueprint, 提供 (mode) -> Blueprint 查表。
type Registry struct {
	m       map[aisolo.AgentMode]Blueprint
	ordered []aisolo.AgentMode
	def     aisolo.AgentMode
}

// NewRegistry 构造默认 Registry, 内置全部 Blueprint。
// 默认 Mode 为 AGENT。
func NewRegistry() *Registry {
	r := &Registry{
		m:   make(map[aisolo.AgentMode]Blueprint),
		def: aisolo.AgentMode_AGENT_MODE_AGENT,
	}
	r.Register(&agentBlueprint{})
	for _, bp := range workflowBlueprints() {
		r.Register(bp)
	}
	r.Register(&supervisorBlueprint{})
	r.Register(&planBlueprint{})
	r.Register(&deepBlueprint{})
	return r
}

// Register 注册 Blueprint (覆盖同名)。
func (r *Registry) Register(bp Blueprint) {
	mode := bp.Mode()
	if _, exists := r.m[mode]; !exists {
		r.ordered = append(r.ordered, mode)
	}
	r.m[mode] = bp
}

// Get 取 Blueprint; 未知 mode 回退到默认 mode。
func (r *Registry) Get(mode aisolo.AgentMode) (Blueprint, bool) {
	if mode == aisolo.AgentMode_AGENT_MODE_UNSPECIFIED {
		bp, ok := r.m[r.def]
		return bp, ok
	}
	bp, ok := r.m[mode]
	return bp, ok
}

// Default 返回默认 Mode。
func (r *Registry) Default() aisolo.AgentMode { return r.def }

// List 返回所有 Mode 的 ModeInfo, 按注册顺序。
func (r *Registry) List() []*aisolo.ModeInfo {
	out := make([]*aisolo.ModeInfo, 0, len(r.ordered))
	for _, m := range r.ordered {
		bp := r.m[m]
		info := bp.Info()
		info.Default = m == r.def
		out = append(out, info)
	}
	return out
}

// checkModel 是所有 Blueprint 的通用前置检查。
func checkModel(deps Dependencies) error {
	if deps.ChatModel == nil {
		return fmt.Errorf("modes: ChatModel is required")
	}
	return nil
}

// appendSkillDirOpts 在配置了 Skills.Dir 时追加 einoxagent.WithSkillsDir，从而挂载
// github.com/cloudwego/eino/adk/middlewares/skill：与 chatwitheino 相同，通过 local Backend +
// NewBackendFromFilesystem 从目录读 SKILL.md，模型先见各技能的 name/description，再自行决定是否 launch。
// 技能不由网关 meta 传递；前端标签若存在，仅等价于帮用户拼一句自然语言进输入框。
func appendSkillDirOpts(deps Dependencies, opts ...einoxagent.Option) []einoxagent.Option {
	out := append([]einoxagent.Option{}, opts...)
	if deps.SkillsDir != "" {
		out = append(out, einoxagent.WithSkillsDir(deps.SkillsDir))
	}
	return out
}
