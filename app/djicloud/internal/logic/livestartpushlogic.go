package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type LiveStartPushLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLiveStartPushLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LiveStartPushLogic {
	return &LiveStartPushLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LiveStartPushLogic) LiveStartPush(in *djicloud.LiveStartPushReq) (*djicloud.CommonRes, error) {
	data := &djisdk.LiveStartPushData{
		URLType:      int(in.UrlType),
		URL:          in.Url,
		VideoID:      in.VideoId,
		VideoQuality: int(in.VideoQuality),
	}
	tid, err := l.svcCtx.DjiClient.LiveStartPush(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[live] live start push failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
