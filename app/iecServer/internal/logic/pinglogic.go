package logic

import (
	"context"

	"zero-service/app/iecServer/iecServer"
	"zero-service/app/iecServer/internal/svc"

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

func (l *PingLogic) Ping(in *iecServer.Req) (*iecServer.Res, error) {
	return &iecServer.Res{
		Pong: "iecServer",
	}, nil
}
