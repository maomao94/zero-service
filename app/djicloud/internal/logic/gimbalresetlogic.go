package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type GimbalResetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGimbalResetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GimbalResetLogic {
	return &GimbalResetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GimbalResetLogic) GimbalReset(in *djicloud.GimbalResetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.GimbalResetData{
		PayloadIndex: in.PayloadIndex,
		ResetMode:    int(in.ResetMode),
	}
	tid, err := l.svcCtx.DjiClient.GimbalReset(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[camera] gimbal reset failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
