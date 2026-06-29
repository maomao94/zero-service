package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type StickControlLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStickControlLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StickControlLogic {
	return &StickControlLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StickControlLogic) StickControl(in *djicloud.StickControlReq) (*djicloud.StickControlRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := buildDrcStickControlData(in)
	if _, err := l.svcCtx.DjiClient.StickControl(l.ctx, deviceSn, int(seq), data); err != nil {
		return nil, err
	}
	return &djicloud.StickControlRes{Seq: int32(seq)}, nil
}

func buildDrcStickControlData(in *djicloud.StickControlReq) *djisdk.DrcStickControlData {
	return &djisdk.DrcStickControlData{
		Roll:        in.GetRoll(),
		Pitch:       in.GetPitch(),
		Throttle:    in.GetThrottle(),
		Yaw:         in.GetYaw(),
		GimbalPitch: in.GetGimbalPitch(),
	}
}
