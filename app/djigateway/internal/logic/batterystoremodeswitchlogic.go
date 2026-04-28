package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatteryStoreModeSwitchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatteryStoreModeSwitchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatteryStoreModeSwitchLogic {
	return &BatteryStoreModeSwitchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatteryStoreModeSwitch 切换电池保养存储模式。
func (l *BatteryStoreModeSwitchLogic) BatteryStoreModeSwitch(in *djigateway.BatteryStoreModeReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.BatteryStoreModeSwitch(l.ctx, in.DeviceSn, int(in.Enable))
	if err != nil {
		l.Errorf("[remote-debug] battery store mode switch failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
