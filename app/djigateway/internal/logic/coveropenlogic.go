package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CoverOpenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCoverOpenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CoverOpenLogic {
	return &CoverOpenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CoverOpen 打开机巢舱盖。
func (l *CoverOpenLogic) CoverOpen(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.CoverOpen(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] cover open failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
