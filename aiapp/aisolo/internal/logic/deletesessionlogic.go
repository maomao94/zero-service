package logic

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"github.com/zeromicro/go-zero/core/logx"

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

// DeleteSession 删除会话
func (l *DeleteSessionLogic) DeleteSession(in *aisolo.DeleteSessionReq) (*aisolo.DeleteSessionResp, error) {
	err := GlobalSessionStore.Delete(l.ctx, in.SessionId)
	if err != nil {
		l.Errorf("delete session failed: %v", err)
		return nil, err
	}

	l.Infof("Session deleted: %s", in.SessionId)

	return &aisolo.DeleteSessionResp{Success: true}, nil
}
