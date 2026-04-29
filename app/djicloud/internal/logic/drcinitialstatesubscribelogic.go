package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcInitialStateSubscribeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcInitialStateSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcInitialStateSubscribeLogic {
	return &DrcInitialStateSubscribeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcInitialStateSubscribeLogic) DrcInitialStateSubscribe(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DrcInitialStateSubscribe(l.ctx, in.GetDeviceSn())
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
