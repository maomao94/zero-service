package agent

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"

	"zero-service/common/einox/fsrestrict"
)

// Option Agent 构造选项
type Option func(*options)

// newOptions 应用选项, 给出可读的默认值。
func newOptions(opts ...Option) *options {
	o := &options{}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

type options struct {
	name            string
	description     string
	instruction     string
	model           any
	tools           []tool.BaseTool
	subAgents       []adk.Agent
	handlers        []adk.ChatModelAgentMiddleware
	middlewares     []adk.AgentMiddleware
	modelOptions    []model.Option
	skillsDir       string
	maxIter         int
	checkpointStore adk.CheckPointStore

	enableWriteTodos bool
	enableFileSystem bool

	// deepFS 控制 Deep 本地文件 Backend 的用户根、会话父目录与读写策略；零值表示不限制路径。
	deepFS fsrestrict.Config
}

// WithName 设置 Agent 名称
func WithName(name string) Option {
	return func(o *options) { o.name = name }
}

// WithDescription 设置 Agent 描述
func WithDescription(desc string) Option {
	return func(o *options) { o.description = desc }
}

// WithInstruction 设置系统指令
func WithInstruction(s string) Option {
	return func(o *options) { o.instruction = s }
}

// WithModel 设置模型
func WithModel(m any) Option {
	return func(o *options) { o.model = m }
}

// WithTools 设置工具列表
func WithTools(tools ...tool.BaseTool) Option {
	return func(o *options) { o.tools = append(o.tools, tools...) }
}

// WithSubAgents 设置子 Agent（Workflow / Supervisor / Deep 使用）。
// Deep 模式下会写入 deep.Config.SubAgents，由 prebuilt/deep 的 task 工具按上下文调度。
func WithSubAgents(subs ...adk.Agent) Option {
	return func(o *options) { o.subAgents = append(o.subAgents, subs...) }
}

// WithHandler 添加 ChatModelAgentMiddleware
func WithHandler(mw adk.ChatModelAgentMiddleware) Option {
	return func(o *options) { o.handlers = append(o.handlers, mw) }
}

// WithHandlers 添加多个 ChatModelAgentMiddleware
func WithHandlers(mws ...adk.ChatModelAgentMiddleware) Option {
	return func(o *options) { o.handlers = append(o.handlers, mws...) }
}

// WithMiddleware 添加 AgentMiddleware
func WithMiddleware(mw adk.AgentMiddleware) Option {
	return func(o *options) { o.middlewares = append(o.middlewares, mw) }
}

// WithMiddlewares 添加多个 AgentMiddleware
func WithMiddlewares(mws ...adk.AgentMiddleware) Option {
	return func(o *options) { o.middlewares = append(o.middlewares, mws...) }
}

// WithModelOption 添加模型参数选项
func WithModelOption(opt model.Option) Option {
	return func(o *options) { o.modelOptions = append(o.modelOptions, opt) }
}

// WithSkillsDir 设置 Skills 目录
func WithSkillsDir(dir string) Option {
	return func(o *options) { o.skillsDir = dir }
}

// WithMaxIterations 设置最大迭代次数（Deep / PlanExecute / Loop 使用）
func WithMaxIterations(n int) Option {
	return func(o *options) { o.maxIter = n }
}

// WithEnableWriteTodos 启用 Deep Agent 的 WriteTodos
func WithEnableWriteTodos(b bool) Option {
	return func(o *options) { o.enableWriteTodos = b }
}

// WithEnableFileSystem 启用 Deep Agent 的文件系统
func WithEnableFileSystem(b bool) Option {
	return func(o *options) { o.enableFileSystem = b }
}

// WithDeepFilesystem 设置 Deep 本地文件系统沙箱（用户根、会话父目录、策略）。
func WithDeepFilesystem(c fsrestrict.Config) Option {
	return func(o *options) { o.deepFS = c }
}

// WithFilesystemAllowedRoots 仅设置用户可见根目录；策略为 PermissivePolicy（与旧 Wrap 行为一致）。
func WithFilesystemAllowedRoots(absRoots []string) Option {
	return func(o *options) {
		if len(absRoots) == 0 {
			o.deepFS.UserRoots = nil
			return
		}
		o.deepFS.UserRoots = append([]string(nil), absRoots...)
		o.deepFS.Policy = fsrestrict.PermissivePolicy()
	}
}

// WithCheckPointStore 设置 adk Runner 的 CheckPointStore（用于中断/恢复）
func WithCheckPointStore(s adk.CheckPointStore) Option {
	return func(o *options) { o.checkpointStore = s }
}
