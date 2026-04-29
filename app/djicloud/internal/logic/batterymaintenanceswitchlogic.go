package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatteryMaintenanceSwitchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatteryMaintenanceSwitchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatteryMaintenanceSwitchLogic {
	return &BatteryMaintenanceSwitchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatteryMaintenanceSwitch 切换电池保养功能开关。
func (l *BatteryMaintenanceSwitchLogic) BatteryMaintenanceSwitch(in *djicloud.BatteryStoreModeReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.BatteryMaintenanceSwitch(l.ctx, in.DeviceSn, int(in.Enable))
	if err != nil {
		l.Errorf("[remote-debug] battery maintenance switch failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
