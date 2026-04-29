package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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
func (l *DebugModeCloseLogic) DebugModeClose(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DebugModeClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] debug mode close failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
