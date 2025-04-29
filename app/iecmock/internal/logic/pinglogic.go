package logic

import (
	"context"

	"zero-service/app/iecmock/iecmock"
	"zero-service/app/iecmock/internal/svc"

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

func (l *PingLogic) Ping(in *iecmock.Req) (*iecmock.Res, error) {
	// todo: add your logic here and delete this line

	return &iecmock.Res{}, nil
}
