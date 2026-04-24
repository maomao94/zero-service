package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendCustomDataToPsdkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendCustomDataToPsdkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCustomDataToPsdkLogic {
	return &SendCustomDataToPsdkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendCustomDataToPsdkLogic) SendCustomDataToPsdk(in *djigateway.CustomDataToPsdkReq) (*djigateway.CommonRes, error) {
	l.Infof("[psdk-transmit] sn=%s value_len=%d", in.DeviceSn, len(in.Value))

	if len(in.Value) >= 256 {
		return &djigateway.CommonRes{
			Code:    -1,
			Message: "value length must be less than 256",
		}, nil
	}

	tid, err := l.svcCtx.DjiClient.SendCustomDataToPsdk(l.ctx, in.DeviceSn, in.Value)
	if err != nil {
		l.Errorf("[psdk-transmit] send failed: %v", err)
		return &djigateway.CommonRes{
			Code:    -1,
			Message: err.Error(),
			Tid:     tid,
		}, nil
	}

	return &djigateway.CommonRes{
		Code:    0,
		Message: "success",
		Tid:     tid,
	}, nil
}
