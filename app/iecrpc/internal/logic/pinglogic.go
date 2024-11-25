package logic

import (
	"context"

	"zero-service/app/iecrpc/iecrpc"
	"zero-service/app/iecrpc/internal/svc"

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

func (l *PingLogic) Ping(in *iecrpc.Req) (*iecrpc.Res, error) {
	return &iecrpc.Res{
		Pong: "iecrpc",
	}, nil
}
