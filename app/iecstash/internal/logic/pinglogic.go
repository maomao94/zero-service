package logic

import (
	"context"

	"zero-service/app/iecstash/iecstash"
	"zero-service/app/iecstash/internal/svc"

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

func (l *PingLogic) Ping(in *iecstash.Req) (*iecstash.Res, error) {
	return &iecstash.Res{Pong: "pong"}, nil
}
