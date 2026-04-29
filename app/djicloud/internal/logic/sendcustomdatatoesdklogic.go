package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendCustomDataToEsdkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendCustomDataToEsdkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCustomDataToEsdkLogic {
	return &SendCustomDataToEsdkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SendCustomDataToEsdk 自定义数据透传至 ESDK 设备。
func (l *SendCustomDataToEsdkLogic) SendCustomDataToEsdk(in *djicloud.CustomDataToEsdkReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.SendCustomDataToEsdk(l.ctx, in.GetDeviceSn(), in.GetValue())
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
