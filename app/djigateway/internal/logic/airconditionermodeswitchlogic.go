package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AirConditionerModeSwitchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAirConditionerModeSwitchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AirConditionerModeSwitchLogic {
	return &AirConditionerModeSwitchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// AirConditionerModeSwitch 切换机巢空调工作模式。
func (l *AirConditionerModeSwitchLogic) AirConditionerModeSwitch(in *djigateway.AirConditionerModeSwitchReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.AirConditionerModeSwitch(l.ctx, in.DeviceSn, int(in.Action))
	if err != nil {
		l.Errorf("[remote-debug] air conditioner mode switch failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
