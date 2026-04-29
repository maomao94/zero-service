package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CoverForceCloseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCoverForceCloseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CoverForceCloseLogic {
	return &CoverForceCloseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CoverForceClose 强制关闭机巢舱盖。
func (l *CoverForceCloseLogic) CoverForceClose(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.CoverForceClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] cover force close failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
