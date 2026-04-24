package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraPhotoStorageSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraPhotoStorageSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraPhotoStorageSetLogic {
	return &CameraPhotoStorageSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraPhotoStorageSetLogic) CameraPhotoStorageSet(in *djigateway.CameraPhotoStorageSetReq) (*djigateway.CommonRes, error) {
	data := &djisdk.CameraStorageSetData{
		PayloadIndex: in.PayloadIndex,
		StorageType:  int(in.PhotoStorageType),
	}
	tid, err := l.svcCtx.DjiClient.CameraPhotoStorageSet(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera photo storage set failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
