package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcChannelDroneEmergencyStopLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcChannelDroneEmergencyStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcChannelDroneEmergencyStopLogic {
	return &DrcChannelDroneEmergencyStopLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcChannelDroneEmergencyStopLogic) DrcChannelDroneEmergencyStop(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
	if !l.svcCtx.Config.DangerousOps.EnableDroneEmergencyStop {
		l.Errorf("[drc] channel emergency stop blocked device_sn=%s: disabled in config DangerousOps.EnableDroneEmergencyStop=false", deviceSn)
		return &djigateway.CommonRes{Code: -1, Message: "emergency stop is disabled, enable it in config DangerousOps.EnableDroneEmergencyStop"}, nil
	}
	err := l.svcCtx.DjiClient.SendDrcDroneEmergencyStop(l.ctx, deviceSn)
	if err != nil {
		l.Errorf("[drc] channel emergency stop failed device_sn=%s: %v", deviceSn, err)
		return errRes("", err), nil
	}
	return okRes(""), nil
}
