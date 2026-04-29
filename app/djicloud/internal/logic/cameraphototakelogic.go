package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraPhotoTakeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraPhotoTakeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraPhotoTakeLogic {
	return &CameraPhotoTakeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraPhotoTakeLogic) CameraPhotoTake(in *djicloud.CameraPhotoTakeReq) (*djicloud.CommonRes, error) {
	data := &djisdk.CameraPhotoTakeData{
		PayloadIndex: in.PayloadIndex,
	}
	tid, err := l.svcCtx.DjiClient.CameraPhotoTake(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera photo take failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
