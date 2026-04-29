package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcCameraIsoSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcCameraIsoSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcCameraIsoSetLogic {
	return &DrcCameraIsoSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcCameraIsoSetLogic) DrcCameraIsoSet(in *djicloud.DrcCameraIsoSetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.DrcCameraIsoSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), ISOValue: int(in.GetIsoValue())}
	tid, err := l.svcCtx.DjiClient.DrcCameraIsoSet(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
