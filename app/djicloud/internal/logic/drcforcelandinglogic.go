package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcForceLandingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcForceLandingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcForceLandingLogic {
	return &DrcForceLandingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcForceLandingLogic) DrcForceLanding(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DrcForceLanding(l.ctx, in.GetDeviceSn())
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
