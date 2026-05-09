package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (l *DroneEmergencyStopLogic) DroneEmergencyStop(in *djicloud.DroneEmergencyStopReq) (*djicloud.DroneEmergencyStopRes, error) {
	if !l.svcCtx.Config.DangerousOps.EnableDroneEmergencyStop {
		l.Errorf("[drc] drone emergency stop blocked: disabled in config")
		return nil, status.Errorf(codes.PermissionDenied,
			"emergency stop is disabled, enable it in config DangerousOps.EnableDroneEmergencyStop")
	}

	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DrcManager.GetNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.DjiClient.DroneEmergencyStop(l.ctx, deviceSn, seq); err != nil {
		l.Errorf("[drc] drone emergency stop failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DroneEmergencyStopRes{Seq: int32(seq)}, nil
}
