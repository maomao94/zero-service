package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraScreenSplitLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraScreenSplitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraScreenSplitLogic {
	return &CameraScreenSplitLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraScreenSplitLogic) CameraScreenSplit(in *djigateway.CameraScreenSplitReq) (*djigateway.CommonRes, error) {
	data := &djisdk.CameraScreenSplitData{
		PayloadIndex: in.PayloadIndex,
		Enable:       in.Enable,
	}
	tid, err := l.svcCtx.DjiClient.CameraScreenSplit(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera screen split failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
