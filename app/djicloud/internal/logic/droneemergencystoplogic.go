package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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

func (l *DroneEmergencyStopLogic) DroneEmergencyStop(in *djicloud.DroneEmergencyStopReq) (*djicloud.DroneEmergencyStopRes, error) {
	if !l.svcCtx.Config.DangerousOps.EnableDroneEmergencyStop {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_03_FORBIDDEN,
			"emergency stop is disabled, enable it in config DangerousOps.EnableDroneEmergencyStop")
	}

	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "获取序列号失败")
	}
	if _, err := l.svcCtx.DjiClient.DroneEmergencyStop(l.ctx, deviceSn, seq); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "无人机紧急停桨失败")
	}
	return &djicloud.DroneEmergencyStopRes{Seq: int32(seq)}, nil
}
