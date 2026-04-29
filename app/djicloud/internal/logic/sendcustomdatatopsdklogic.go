package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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

func (l *SendCustomDataToPsdkLogic) SendCustomDataToPsdk(in *djicloud.CustomDataToPsdkReq) (*djicloud.CommonRes, error) {
	l.Infof("[psdk-transmit] sn=%s value_len=%d", in.DeviceSn, len(in.Value))

	if len(in.Value) >= 256 {
		return &djicloud.CommonRes{
			Code:    -1,
			Message: "value length must be less than 256",
		}, nil
	}

	tid, err := l.svcCtx.DjiClient.SendCustomDataToPsdk(l.ctx, in.DeviceSn, in.Value)
	if err != nil {
		l.Errorf("[psdk-transmit] send failed: %v", err)
		return errRes(tid, err), nil
	}

	return okRes(tid), nil
}
