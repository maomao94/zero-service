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

func (l *DrcCameraApertureValueSetLogic) DrcCameraApertureValueSet(in *djicloud.DrcCameraApertureValueSetReq) (*djicloud.DrcCameraApertureValueSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DrcManager.GetNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcCameraApertureValueSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), ApertureValue: int(in.GetApertureValue())}
	if _, err := l.svcCtx.DjiClient.DrcCameraApertureValueSet(l.ctx, deviceSn, seq, data); err != nil {
		l.Errorf("[drc] camera aperture value set failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcCameraApertureValueSetRes{Seq: int32(seq)}, nil
}
