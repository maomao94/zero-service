package logic

import (
	"context"
	"errors"

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
	sess, err := l.svcCtx.Sessions.GetSession(l.ctx, in.UserId, in.SessionId)
	if err != nil {
		return &aisolo.DeleteSessionResp{Success: false}, err
	}
	if sess.Status == aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		return &aisolo.DeleteSessionResp{Success: false}, errors.New("cannot delete running session")
	}
	if l.svcCtx.Messages != nil {
		if err := l.svcCtx.Messages.DeleteSession(l.ctx, in.UserId, in.SessionId); err != nil {
			return &aisolo.DeleteSessionResp{Success: false}, err
		}
	}
	if err := l.svcCtx.Sessions.DeleteSession(l.ctx, in.UserId, in.SessionId); err != nil {
		return &aisolo.DeleteSessionResp{Success: false}, err
	}
	return &aisolo.DeleteSessionResp{Success: true}, nil
}
