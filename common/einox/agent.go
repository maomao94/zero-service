// Package einox 提供 AI Agent 相关的公共类型与工具。
//
// 本包仅保留被其他服务（aiapp/aisolo 等）依赖的少量公共符号：
//   - Agent 相关错误（ErrAgentNotFound 等）
//   - Session ID 的 context 读写辅助
//
// 复杂抽象已拆分到 common/einox/agent、common/einox/memory 等子包。
package einox

import "context"

// sessionIDKey 用于在 context 中携带会话 ID。
type sessionIDKey struct{}

// WithSessionIDContext 将会话 ID 存入 context。
func WithSessionIDContext(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

// GetSessionID 从 context 获取会话 ID，没有则返回空串。
func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDKey{}).(string); ok {
		return v
	}
	return ""
}
