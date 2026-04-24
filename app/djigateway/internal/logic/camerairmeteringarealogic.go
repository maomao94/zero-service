package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraIrMeteringAreaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraIrMeteringAreaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraIrMeteringAreaLogic {
	return &CameraIrMeteringAreaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraIrMeteringAreaLogic) CameraIrMeteringArea(in *djigateway.CameraIrMeteringAreaReq) (*djigateway.CommonRes, error) {
	data := &djisdk.CameraIrMeteringAreaData{
		PayloadIndex: in.PayloadIndex,
		X:            in.X,
		Y:            in.Y,
		Width:        in.Width,
		Height:       in.Height,
	}
	tid, err := l.svcCtx.DjiClient.CameraIrMeteringArea(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera ir metering area failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
