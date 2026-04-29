package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReturnHomeCancelAutoReturnLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReturnHomeCancelAutoReturnLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReturnHomeCancelAutoReturnLogic {
	return &ReturnHomeCancelAutoReturnLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ReturnHomeCancelAutoReturnLogic) ReturnHomeCancelAutoReturn(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ReturnHomeCancelAutoReturn(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] return home cancel failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
