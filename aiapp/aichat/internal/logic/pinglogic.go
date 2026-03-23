package logic

import (
	"context"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PingLogic) Ping(in *aichat.PingReq) (*aichat.PingRes, error) {
	return &aichat.PingRes{Pong: "pong"}, nil
}
