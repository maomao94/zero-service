package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CustomDataTransmissionToEsdkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCustomDataTransmissionToEsdkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CustomDataTransmissionToEsdkLogic {
	return &CustomDataTransmissionToEsdkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CustomDataTransmissionToEsdkLogic) CustomDataTransmissionToEsdk(in *djicloud.CustomDataTransmissionToEsdkReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.CustomDataTransmissionToEsdk(l.ctx, in.GetDeviceSn(), in.GetValue())
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
