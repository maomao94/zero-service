package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type LiveStopPushLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLiveStopPushLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LiveStopPushLogic {
	return &LiveStopPushLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LiveStopPushLogic) LiveStopPush(in *djigateway.LiveStopPushReq) (*djigateway.CommonRes, error) {
	data := &djisdk.LiveStopPushData{
		VideoID: in.VideoId,
	}
	tid, err := l.svcCtx.DjiClient.LiveStopPush(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[live] live stop push failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
