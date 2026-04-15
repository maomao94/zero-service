package einox

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// Agent 接口
// =============================================================================

// AgentInterface Agent 接口定义
// 定义 Agent 必须实现的方法
type AgentInterface interface {
	// Run 运行 Agent（单轮对话）
	Run(ctx context.Context, input string, opts ...RunOption) (*AgentResult, error)

	// RunWithHistory 运行 Agent（带历史消息）
	RunWithHistory(ctx context.Context, input string, opts ...RunOption) (*AgentResult, error)

	// RunStream 流式运行
	RunStream(ctx context.Context, input string, opts ...RunOption) (<-chan *AgentResult, error)

	// Stream 返回 Agent 事件的原始流
	Stream(ctx context.Context, input string, opts ...RunOption) (*adk.AsyncIterator[*adk.AgentEvent], error)

	// ClearMemory 清除记忆
	ClearMemory(ctx context.Context, userID, sessionID string) error
}

// AgentResult Agent 执行结果
type AgentResult struct {
	Response string // 响应内容
	Err      error  // 错误信息（仅在发生错误时设置）
}

// RunOption 运行选项
type RunOption func(*RunOptions)

// RunOptions 运行选项
type RunOptions struct {
	UserID    string
	SessionID string
	System    string
}

// WithUserID 设置用户 ID
func WithUserID(userID string) RunOption {
	return func(o *RunOptions) {
		o.UserID = userID
	}
}

// WithSessionID 设置会话 ID
func WithSessionID(sessionID string) RunOption {
	return func(o *RunOptions) {
		o.SessionID = sessionID
	}
}

// WithSystem 设置系统消息
func WithSystem(system string) RunOption {
	return func(o *RunOptions) {
		o.System = system
	}
}

// ToSchemaMessages 转换为 schema.Message 切片
func (r *RunOptions) ToSchemaMessages() []*schema.Message {
	var msgs []*schema.Message
	if r.System != "" {
		msgs = append(msgs, &schema.Message{
			Role:    schema.System,
			Content: r.System,
		})
	}
	return msgs
}

type sessionIDKey struct{}

// WithSessionIDContext 将会话ID存入上下文
func WithSessionIDContext(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

// GetSessionID 从上下文获取会话ID
func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDKey{}).(string); ok {
		return v
	}
	return ""
}
