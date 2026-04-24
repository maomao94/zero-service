package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChargeCloseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChargeCloseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChargeCloseLogic {
	return &ChargeCloseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ChargeClose 关闭机巢充电功能。
func (l *ChargeCloseLogic) ChargeClose(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ChargeClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] charge close failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
