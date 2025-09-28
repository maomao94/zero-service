package logic

import (
	"context"

	"zero-service/app/bridgemqtt/bridgemqtt"
	"zero-service/app/bridgemqtt/internal/svc"

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

func (l *PingLogic) Ping(in *bridgemqtt.Req) (*bridgemqtt.Res, error) {
	return &bridgemqtt.Res{
		Pong: "pong",
	}, nil
}
