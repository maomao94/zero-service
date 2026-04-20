package knowledge

import (
	"context"
	"strings"
)

type turnCtxKey struct{}

// turnValues 由 aisolo turn 在每次 Run/Resume 前注入，供检索工具读取。
type turnValues struct {
	UserID          string
	KnowledgeBaseID string
}

// WithAgentTurn 注入当前轮次的用户 ID 与会话绑定的知识库 ID（空表示未绑定）。
func WithAgentTurn(ctx context.Context, userID, knowledgeBaseID string) context.Context {
	return context.WithValue(ctx, turnCtxKey{}, &turnValues{
		UserID:          strings.TrimSpace(userID),
		KnowledgeBaseID: strings.TrimSpace(knowledgeBaseID),
	})
}

// UserIDFrom 读取 WithAgentTurn 注入的用户 ID。
func UserIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(turnCtxKey{}).(*turnValues)
	if v == nil {
		return ""
	}
	return v.UserID
}

// KnowledgeBaseIDFrom 读取当前会话绑定的知识库 ID。
func KnowledgeBaseIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(turnCtxKey{}).(*turnValues)
	if v == nil {
		return ""
	}
	return v.KnowledgeBaseID
}
