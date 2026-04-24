package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatteryStoreModeSwitchSwitchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatteryStoreModeSwitchSwitchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatteryStoreModeSwitchSwitchLogic {
	return &BatteryStoreModeSwitchSwitchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatteryStoreModeSwitchSwitch 切换电池保养存储模式。
func (l *BatteryStoreModeSwitchSwitchLogic) BatteryStoreModeSwitchSwitch(in *djigateway.BatteryStoreModeReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.BatteryStoreModeSwitchSwitch(l.ctx, in.DeviceSn, int(in.Enable))
	if err != nil {
		l.Errorf("[remote-debug] battery store mode switch failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
