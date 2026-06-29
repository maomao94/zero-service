package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcModeExitLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcModeExitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcModeExitLogic {
	return &DrcModeExitLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcModeExitLogic) DrcModeExit(in *djicloud.DrcModeExitReq) (*djicloud.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
	tid, err := l.svcCtx.DjiClient.DrcModeExit(l.ctx, deviceSn)
	if err != nil {
		return errRes(tid, err), nil
	}
	if err := l.svcCtx.DjiClient.DisableDrc(l.ctx, deviceSn); err != nil {
		return nil, err
	}
	return okRes(tid), nil
}
