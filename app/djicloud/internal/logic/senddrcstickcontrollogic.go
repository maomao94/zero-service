package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
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

func (l *SendDrcStickControlLogic) SendDrcStickControl(in *djicloud.DrcStickControlReq) (*djicloud.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
	seq := int(in.GetSeq())
	data := buildDrcStickControlData(in)
	tid, err := l.svcCtx.DjiClient.SendDrcStickControl(l.ctx, deviceSn, seq, data)
	if err != nil {
		l.Errorf("[drc] send stick control failed device_sn=%s seq=%d tid=%s: %v", deviceSn, seq, tid, err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}

func buildDrcStickControlData(in *djicloud.DrcStickControlReq) *djisdk.DrcStickControlData {
	return &djisdk.DrcStickControlData{
		Roll:        in.GetRoll(),
		Pitch:       in.GetPitch(),
		Throttle:    in.GetThrottle(),
		Yaw:         in.GetYaw(),
		GimbalPitch: in.GetGimbalPitch(),
	}
}
