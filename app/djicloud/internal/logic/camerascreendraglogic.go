package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type CameraScreenDragLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCameraScreenDragLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CameraScreenDragLogic {
	return &CameraScreenDragLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CameraScreenDragLogic) CameraScreenDrag(in *djicloud.CameraScreenDragReq) (*djicloud.CommonRes, error) {
	data := &djisdk.CameraScreenDragData{
		PayloadIndex: in.PayloadIndex,
		X:            in.X,
		Y:            in.Y,
	}
	tid, err := l.svcCtx.DjiClient.CameraScreenDrag(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] camera screen drag failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
