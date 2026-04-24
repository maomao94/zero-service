package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraLookAtLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraLookAtLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraLookAtLogic {
	return &CameraLookAtLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraLookAtLogic) CameraLookAt(in *djigateway.CameraLookAtReq) (*djigateway.CommonRes, error) {
	data := &djisdk.CameraLookAtData{
		PayloadIndex: in.PayloadIndex,
		Latitude:     in.Latitude,
		Longitude:    in.Longitude,
		Height:       in.Height,
	}
	tid, err := l.svcCtx.DjiClient.CameraLookAt(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera look at failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
