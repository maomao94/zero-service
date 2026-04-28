package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendDrcStickControlLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendDrcStickControlLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendDrcStickControlLogic {
	return &SendDrcStickControlLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendDrcStickControlLogic) SendDrcStickControl(in *djigateway.DrcStickControlReq) (*djigateway.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
	seq := int(in.GetSeq())
	data := buildDrcStickControlData(in)
	err := l.svcCtx.DjiClient.SendDrcStickControl(l.ctx, deviceSn, seq, data)
	if err != nil {
		l.Errorf("[drc] send stick control failed device_sn=%s seq=%d: %v", deviceSn, seq, err)
		return errRes("", err), nil
	}
	return okRes(""), nil
}

func buildDrcStickControlData(in *djigateway.DrcStickControlReq) *djisdk.DrcStickControlData {
	return &djisdk.DrcStickControlData{
		Roll:        in.GetRoll(),
		Pitch:       in.GetPitch(),
		Throttle:    in.GetThrottle(),
		Yaw:         in.GetYaw(),
		GimbalPitch: in.GetGimbalPitch(),
	}
}
