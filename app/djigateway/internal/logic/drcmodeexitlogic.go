package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

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

func (l *DrcModeExitLogic) DrcModeExit(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
	tid, err := l.svcCtx.DjiClient.DrcModeExit(l.ctx, deviceSn)
	if err != nil {
		l.Errorf("[drc] mode exit failed device_sn=%s tid=%s: %v", deviceSn, tid, err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
