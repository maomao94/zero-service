package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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

func (l *ReturnHomeLogic) ReturnHome(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ReturnHome(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] return home failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
