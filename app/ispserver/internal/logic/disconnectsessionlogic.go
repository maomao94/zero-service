package logic

import (
	"context"
	"fmt"

	"zero-service/app/ispserver/internal/svc"
	"zero-service/app/ispserver/ispserver"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisconnectSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisconnectSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisconnectSessionLogic {
	return &DisconnectSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisconnectSessionLogic) DisconnectSession(in *ispserver.DisconnectSessionReq) (*ispserver.DisconnectSessionRes, error) {
	session := l.svcCtx.IspServer.Manager().Get(in.GetSessionId())
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", in.GetSessionId())
	}
	_ = session.Close()
	return &ispserver.DisconnectSessionRes{}, nil
}
