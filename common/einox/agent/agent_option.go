package agent

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"zero-service/common/einox/memory"
)

// =============================================================================
// Agent 配置选项（构造时）
// =============================================================================

// Option Agent 构造选项
type Option func(*options)

type options struct {
	name         string
	description  string
	instruction  string
	model        any // 支持 BaseChatModel/ChatModel/ToolCallingChatModel
	tools        []tool.BaseTool
	storage      memory.Storage
	stream       bool
	memoryConfig *memory.MemoryConfig
	handlers     []adk.ChatModelAgentMiddleware // 使用 Handlers (接口类型)
	middlewares  []adk.AgentMiddleware          // 简单场景用 Middlewares (结构体类型)
	modelOptions []model.Option
}

// WithName 设置 Agent 名称
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithDescription 设置 Agent 描述
func WithDescription(desc string) Option {
	return func(o *options) {
		o.description = desc
	}
}

// WithInstruction 设置系统指令
func WithInstruction(instruction string) Option {
	return func(o *options) {
		o.instruction = instruction
	}
}

// WithModel 设置模型（支持 BaseChatModel/ChatModel/ToolCallingChatModel）
//
// 推荐使用 ToolCallingChatModel（如 ark.ChatModel），因为它支持并发安全的 WithTools
func WithModel(m any) Option {
	return func(o *options) {
		o.model = m
	}
}

// WithTools 设置工具列表
func WithTools(tools ...tool.BaseTool) Option {
	return func(o *options) {
		o.tools = tools
	}
}

// WithStorage 设置记忆存储
func WithStorage(storage memory.Storage) Option {
	return func(o *options) {
		o.storage = storage
	}
}

// WithMemoryStorage 创建并设置记忆存储（使用默认内存存储）
func WithMemoryStorage(opts ...memory.StorageOption) Option {
	return func(o *options) {
		o.storage = memory.NewMemoryStorage(opts...)
	}
}

// WithStream 启用流式输出
func WithStream(enable bool) Option {
	return func(o *options) {
		o.stream = enable
	}
}

// WithMemoryConfig 设置记忆配置
//
// 配置用户记忆、会话摘要等功能。
// 传 nil 使用默认配置，传 memory.DisableMemory() 禁用记忆功能。
func WithMemoryConfig(cfg *memory.MemoryConfig) Option {
	return func(o *options) {
		o.memoryConfig = cfg
	}
}

// WithHandler 添加 Handler（ChatModelAgentMiddleware 接口实现）
//
// 推荐使用此方法添加复杂中间件，如：
// - ApprovalMiddleware: 工具调用审批
// - ChoiceMiddleware: 单选/多选交互
// - HumanConfirmMiddleware: 人工确认
func WithHandler(mw adk.ChatModelAgentMiddleware) Option {
	return func(o *options) {
		o.handlers = append(o.handlers, mw)
	}
}

// WithHandlers 添加多个 Handler
func WithHandlers(mws ...adk.ChatModelAgentMiddleware) Option {
	return func(o *options) {
		o.handlers = append(o.handlers, mws...)
	}
}

// WithMiddleware 添加 Middleware（AgentMiddleware 结构体，简化场景）
//
// 适用于简单的静态扩展，如添加额外指令或工具
func WithMiddleware(mw adk.AgentMiddleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, mw)
	}
}

// WithMiddlewares 添加多个 Middleware
func WithMiddlewares(mws ...adk.AgentMiddleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, mws...)
	}
}

// WithModelOption 添加模型选项
//
// 可用选项参考 model 包的 WithTemperature, WithTopP, WithMaxTokens 等
func WithModelOption(opt model.Option) Option {
	return func(o *options) {
		o.modelOptions = append(o.modelOptions, opt)
	}
}

// =============================================================================
// 辅助函数
// =============================================================================

// schemaMsgFromMessages 转换消息列表
func schemaMsgFromMessages(msgs []*memory.ConversationMessage) []*schema.Message {
	var schemaMsgs []*schema.Message
	for _, msg := range msgs {
		schemaMsgs = append(schemaMsgs, msg.ToSchemaMessage())
	}
	return schemaMsgs
}
