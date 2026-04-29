package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type MediaUploadFlighttaskMediaPrioritizeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMediaUploadFlighttaskMediaPrioritizeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MediaUploadFlighttaskMediaPrioritizeLogic {
	return &MediaUploadFlighttaskMediaPrioritizeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// MediaUploadFlighttaskMediaPrioritize 优先上传指定航线任务媒体。
func (l *MediaUploadFlighttaskMediaPrioritizeLogic) MediaUploadFlighttaskMediaPrioritize(in *djicloud.MediaFlighttaskReq) (*djicloud.CommonRes, error) {
	data := &djisdk.MediaUploadFlighttaskMediaPrioritizeData{FlightID: in.FlightId}
	tid, err := l.svcCtx.DjiClient.MediaUploadFlighttaskMediaPrioritize(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[media] upload flighttask media prioritize failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
