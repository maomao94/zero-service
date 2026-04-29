package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CoverCloseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCoverCloseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CoverCloseLogic {
	return &CoverCloseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CoverClose 关闭机巢舱盖。
func (l *CoverCloseLogic) CoverClose(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.CoverClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] cover close failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
