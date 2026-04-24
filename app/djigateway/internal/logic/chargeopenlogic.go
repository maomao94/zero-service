package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

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
func (l *ChargeOpenLogic) ChargeOpen(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ChargeOpen(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] charge open failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
