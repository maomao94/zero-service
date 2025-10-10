package logic

import (
	"context"

	"zero-service/app/alarm/alarm"
	"zero-service/app/alarm/internal/svc"

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

func (l *PingLogic) Ping(in *alarm.Req) (*alarm.Res, error) {
	return &alarm.Res{
		Pong: "pong",
	}, nil
}
