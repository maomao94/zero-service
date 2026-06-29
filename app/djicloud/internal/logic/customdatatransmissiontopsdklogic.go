package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CustomDataTransmissionToPsdkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCustomDataTransmissionToPsdkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CustomDataTransmissionToPsdkLogic {
	return &CustomDataTransmissionToPsdkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CustomDataTransmissionToPsdkLogic) CustomDataTransmissionToPsdk(in *djicloud.CustomDataTransmissionToPsdkReq) (*djicloud.CommonRes, error) {
	l.Infof("[psdk-transmit] sn=%s value_len=%d", in.DeviceSn, len(in.Value))

	if len(in.Value) >= 256 {
		return &djicloud.CommonRes{
			Code:    -1,
			Message: "value length must be less than 256",
		}, nil
	}

	tid, err := l.svcCtx.DjiClient.CustomDataTransmissionToPsdk(l.ctx, in.DeviceSn, in.Value)
	if err != nil {
		return errRes(tid, err), nil
	}

	return okRes(tid), nil
}
