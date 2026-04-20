package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
)

type DeleteSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSessionLogic {
	return &DeleteSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteSessionLogic) DeleteSession(in *aisolo.DeleteSessionReq) (*aisolo.DeleteSessionResp, error) {
	if err := l.svcCtx.Sessions.DeleteSession(l.ctx, in.UserId, in.SessionId); err != nil {
		return &aisolo.DeleteSessionResp{Success: false}, err
	}
	if l.svcCtx.Messages != nil {
		_ = l.svcCtx.Messages.DeleteSession(l.ctx, in.UserId, in.SessionId)
	}
	return &aisolo.DeleteSessionResp{Success: true}, nil
}
