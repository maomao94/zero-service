package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChargeOpenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChargeOpenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChargeOpenLogic {
	return &ChargeOpenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ChargeOpen 开启机巢充电功能。
func (l *ChargeOpenLogic) ChargeOpen(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ChargeOpen(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] charge open failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
