package logic

import (
	"context"

	"zero-service/facade/iecstream/iecstream"
	"zero-service/facade/iecstream/internal/svc"

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

func (l *PingLogic) Ping(in *iecstream.Req) (*iecstream.Res, error) {
	return &iecstream.Res{Pong: in.Ping}, nil
}
