package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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

func (l *DroneEmergencyStopLogic) DroneEmergencyStop(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	if !l.svcCtx.Config.DangerousOps.EnableDroneEmergencyStop {
		l.Errorf("[drc] drone emergency stop blocked: disabled in config (DangerousOps.EnableDroneEmergencyStop=false)")
		return &djicloud.CommonRes{Code: -1, Message: "emergency stop is disabled, enable it in config DangerousOps.EnableDroneEmergencyStop"}, nil
	}

	tid, err := l.svcCtx.DjiClient.DroneEmergencyStop(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] drone emergency stop failed tid=%s: %v", tid, err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
