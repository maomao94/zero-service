package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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
func (l *ChargeCloseLogic) ChargeClose(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ChargeClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] charge close failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
