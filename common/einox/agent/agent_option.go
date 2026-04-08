package agent

import (
	"time"

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
	name        string
	description string
	instruction string
	model       any // 支持 BaseChatModel/ChatModel/ToolCallingChatModel
	tools       []tool.BaseTool
	storage     memory.Storage
	stream      bool
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

// =============================================================================
// Agent 运行时选项
// =============================================================================

// RunOption 运行时选项
type RunOption func(*runOptions)

type runOptions struct {
	sessionID string
	userID    string
	system    string
	messages  []*schema.Message
	tools     []tool.BaseTool
	timeout   time.Duration
}

// WithSessionID 设置会话 ID
func WithSessionID(sessionID string) RunOption {
	return func(o *runOptions) {
		o.sessionID = sessionID
	}
}

// WithUserID 设置用户 ID
func WithUserID(userID string) RunOption {
	return func(o *runOptions) {
		o.userID = userID
	}
}

// WithSystem 设置系统提示
func WithSystem(system string) RunOption {
	return func(o *runOptions) {
		o.system = system
	}
}

// WithMessages 设置初始消息
func WithMessages(msgs ...*schema.Message) RunOption {
	return func(o *runOptions) {
		o.messages = msgs
	}
}

// WithDynamicTools 设置动态工具
func WithDynamicTools(tools ...tool.BaseTool) RunOption {
	return func(o *runOptions) {
		o.tools = tools
	}
}

// WithTimeout 设置超时
func WithTimeout(d time.Duration) RunOption {
	return func(o *runOptions) {
		o.timeout = d
	}
}
