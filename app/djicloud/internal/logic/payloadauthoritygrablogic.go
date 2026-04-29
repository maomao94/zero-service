package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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

func (l *PayloadAuthorityGrabLogic) PayloadAuthorityGrab(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.PayloadAuthorityGrab(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] payload authority grab failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
