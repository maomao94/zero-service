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

func (l *DrcCameraIsoSetLogic) DrcCameraIsoSet(in *djicloud.DrcCameraIsoSetReq) (*djicloud.DrcCameraIsoSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DrcManager.GetNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcCameraIsoSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), ISOValue: int(in.GetIsoValue())}
	if _, err := l.svcCtx.DjiClient.DrcCameraIsoSet(l.ctx, deviceSn, seq, data); err != nil {
		l.Errorf("[drc] camera iso set failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcCameraIsoSetRes{Seq: int32(seq)}, nil
}
