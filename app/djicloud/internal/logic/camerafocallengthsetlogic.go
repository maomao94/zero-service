package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraFocalLengthSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraFocalLengthSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraFocalLengthSetLogic {
	return &CameraFocalLengthSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraFocalLengthSetLogic) CameraFocalLengthSet(in *djicloud.CameraFocalLengthSetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.CameraFocalLengthSetData{
		PayloadIndex: in.PayloadIndex,
		CameraType:   int(in.CameraType),
		ZoomFactor:   in.ZoomFactor,
	}
	tid, err := l.svcCtx.DjiClient.CameraFocalLengthSet(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera focal length set failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
