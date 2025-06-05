package logic

import (
	"context"

	"zero-service/app/iecagent/iecagent"
	"zero-service/app/iecagent/internal/svc"

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

func (l *PingLogic) Ping(in *iecagent.Req) (*iecagent.Res, error) {
	return &iecagent.Res{Pong: "iecAgent"}, nil
}
