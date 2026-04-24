package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayloadAuthorityGrabLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPayloadAuthorityGrabLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PayloadAuthorityGrabLogic {
	return &PayloadAuthorityGrabLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PayloadAuthorityGrabLogic) PayloadAuthorityGrab(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.PayloadAuthorityGrab(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] payload authority grab failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
