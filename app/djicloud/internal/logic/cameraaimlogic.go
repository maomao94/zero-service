package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraAimLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraAimLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraAimLogic {
	return &CameraAimLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraAimLogic) CameraAim(in *djicloud.CameraAimReq) (*djicloud.CommonRes, error) {
	data := &djisdk.CameraAimData{
		PayloadIndex: in.PayloadIndex,
		CameraType:   int(in.CameraType),
		Locked:       in.Locked,
		X:            in.X,
		Y:            in.Y,
	}
	tid, err := l.svcCtx.DjiClient.CameraAim(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera aim failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
