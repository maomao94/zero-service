package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
)

type ListMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMessagesLogic {
	return &ListMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListMessagesLogic) ListMessages(in *aisolo.ListMessagesReq) (*aisolo.ListMessagesResp, error) {
	if l.svcCtx.Messages == nil {
		return &aisolo.ListMessagesResp{}, nil
	}
	limit := int(in.Limit)
	msgs, err := l.svcCtx.Messages.GetMessages(l.ctx, in.UserId, in.SessionId, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*aisolo.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, &aisolo.Message{
			Id:         m.ID,
			SessionId:  m.SessionID,
			UserId:     m.UserID,
			Role:       m.Role,
			Content:    m.Content,
			CreatedAt:  m.CreatedAt.Unix(),
			ToolCallId: m.ToolCallID,
			ToolName:   m.ToolName,
		})
	}
	return &aisolo.ListMessagesResp{Messages: out, Total: int32(len(out))}, nil
}
