package fsrestrict

import "context"

type ctxKeySession struct{}

// WithSessionID 将当前 gRPC 会话 ID 注入 context，供 Deep 文件 Backend 解析会话工作区。
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	if sessionID == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeySession{}, sessionID)
}

// SessionIDFrom 读取 WithSessionID 注入的会话 ID；未注入时返回空字符串。
func SessionIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeySession{}).(string)
	return v
}
