package logic

import (
	"context"

	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"

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

func (l *PingLogic) Ping(in *xfusionmock.Req) (*xfusionmock.Res, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	l.Logger.Infof("ping >>> %s", in.Ping)
	return &xfusionmock.Res{Pong: "hello"}, nil
}
