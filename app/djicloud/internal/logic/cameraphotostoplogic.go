package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraPhotoStopLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraPhotoStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraPhotoStopLogic {
	return &CameraPhotoStopLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraPhotoStopLogic) CameraPhotoStop(in *djicloud.CameraPhotoStopReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.CameraPhotoStop(l.ctx, in.DeviceSn, in.PayloadIndex)
	if err != nil {
		l.Errorf("[camera] camera photo stop failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
