package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraRecordingStartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraRecordingStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraRecordingStartLogic {
	return &CameraRecordingStartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraRecordingStartLogic) CameraRecordingStart(in *djigateway.CameraRecordingStartReq) (*djigateway.CommonRes, error) {
	data := &djisdk.CameraRecordingStartData{
		PayloadIndex: in.PayloadIndex,
	}
	tid, err := l.svcCtx.DjiClient.CameraRecordingStart(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera recording start failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
