package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcCameraMechanicalShutterSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcCameraMechanicalShutterSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcCameraMechanicalShutterSetLogic {
	return &DrcCameraMechanicalShutterSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcCameraMechanicalShutterSetLogic) DrcCameraMechanicalShutterSet(in *djicloud.DrcCameraMechanicalShutterSetReq) (*djicloud.DrcCameraMechanicalShutterSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcCameraMechanicalShutterSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), MechanicalShutterState: int(in.GetDewarpingState())}
	if _, err := l.svcCtx.DjiClient.DrcCameraMechanicalShutterSet(l.ctx, deviceSn, seq, data); err != nil {
		return nil, err
	}
	return &djicloud.DrcCameraMechanicalShutterSetRes{Seq: int32(seq)}, nil
}
