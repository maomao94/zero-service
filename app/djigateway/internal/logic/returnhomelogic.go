package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReturnHomeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReturnHomeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReturnHomeLogic {
	return &ReturnHomeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ReturnHomeLogic) ReturnHome(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ReturnHome(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] return home failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
