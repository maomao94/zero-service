package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcCameraShutterSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcCameraShutterSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcCameraShutterSetLogic {
	return &DrcCameraShutterSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcCameraShutterSetLogic) DrcCameraShutterSet(in *djicloud.DrcCameraShutterSetReq) (*djicloud.DrcCameraShutterSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcCameraShutterSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), ShutterValue: int(in.GetShutterValue())}
	if _, err := l.svcCtx.DjiClient.DrcCameraShutterSet(l.ctx, deviceSn, seq, data); err != nil {
		l.Errorf("[drc] camera shutter set failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcCameraShutterSetRes{Seq: int32(seq)}, nil
}
