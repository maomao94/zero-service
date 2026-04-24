package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DebugModeOpenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDebugModeOpenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DebugModeOpenLogic {
	return &DebugModeOpenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DebugModeOpen 开启机巢调试模式。
func (l *DebugModeOpenLogic) DebugModeOpen(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DebugModeOpen(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] debug mode open failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
