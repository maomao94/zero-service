package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DebugModeCloseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDebugModeCloseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DebugModeCloseLogic {
	return &DebugModeCloseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DebugModeClose 关闭机巢调试模式。
func (l *DebugModeCloseLogic) DebugModeClose(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DebugModeClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] debug mode close failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
