package logic

import (
	"context"

	"zero-service/app/logdump/internal/svc"
	"zero-service/app/logdump/logdump"

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

func (l *PingLogic) Ping(in *logdump.PingReq) (*logdump.PingRes, error) {
	return &logdump.PingRes{
		Pong: "pong",
	}, nil
}
