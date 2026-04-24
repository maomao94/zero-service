package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SupplementLightCloseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSupplementLightCloseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SupplementLightCloseLogic {
	return &SupplementLightCloseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SupplementLightClose 关闭机巢补光灯。
func (l *SupplementLightCloseLogic) SupplementLightClose(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.SupplementLightClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] supplement light close failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
