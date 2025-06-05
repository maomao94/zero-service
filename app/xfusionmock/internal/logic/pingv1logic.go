package logic

import (
	"context"

	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingV1Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPingV1Logic(ctx context.Context, svcCtx *svc.ServiceContext) *PingV1Logic {
	return &PingV1Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PingV1Logic) PingV1(in *xfusionmock.Req) (*xfusionmock.Res, error) {
	// todo: add your logic here and delete this line

	return &xfusionmock.Res{}, nil
}
