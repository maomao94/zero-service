package agent

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
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

// WithSubAgents 设置子 Agent（Workflow / Supervisor 使用）
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

// WithCheckPointStore 设置 adk Runner 的 CheckPointStore（用于中断/恢复）
func WithCheckPointStore(s adk.CheckPointStore) Option {
	return func(o *options) { o.checkpointStore = s }
}
