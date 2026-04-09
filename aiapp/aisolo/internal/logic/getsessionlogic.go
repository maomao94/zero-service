package logic

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/internal/svc"
)

type GetSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSessionLogic {
	return &GetSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetSession 获取会话
func (l *GetSessionLogic) GetSession(in *aisolo.SessionRequest) (*aisolo.Session, error) {
	session, err := GlobalSessionStore.Get(l.ctx, in.SessionId)
	if err != nil {
		l.Errorf("get session failed: %v", err)
		return nil, err
	}

	return session, nil
}
