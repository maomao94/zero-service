package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcCameraApertureValueSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcCameraApertureValueSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcCameraApertureValueSetLogic {
	return &DrcCameraApertureValueSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcCameraApertureValueSetLogic) DrcCameraApertureValueSet(in *djicloud.DrcCameraApertureValueSetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.DrcCameraApertureValueSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), ApertureValue: int(in.GetApertureValue())}
	tid, err := l.svcCtx.DjiClient.DrcCameraApertureValueSet(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
