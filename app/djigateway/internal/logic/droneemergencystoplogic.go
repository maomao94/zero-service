package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DroneEmergencyStopLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDroneEmergencyStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DroneEmergencyStopLogic {
	return &DroneEmergencyStopLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DroneEmergencyStopLogic) DroneEmergencyStop(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	if !l.svcCtx.Config.DangerousOps.EnableDroneEmergencyStop {
		l.Errorf("[drc] drone emergency stop blocked: disabled in config (DangerousOps.EnableDroneEmergencyStop=false)")
		return &djigateway.CommonRes{Code: -1, Message: "emergency stop is disabled, enable it in config DangerousOps.EnableDroneEmergencyStop"}, nil
	}

	tid, err := l.svcCtx.DjiClient.DroneEmergencyStop(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] drone emergency stop failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
