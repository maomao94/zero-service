package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type MediaHighestPriorityUploadFlighttaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMediaHighestPriorityUploadFlighttaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MediaHighestPriorityUploadFlighttaskLogic {
	return &MediaHighestPriorityUploadFlighttaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// MediaHighestPriorityUploadFlighttask 最高优先级上传指定航线任务媒体。
func (l *MediaHighestPriorityUploadFlighttaskLogic) MediaHighestPriorityUploadFlighttask(in *djicloud.MediaFlighttaskReq) (*djicloud.CommonRes, error) {
	data := &djisdk.MediaHighestPriorityUploadFlighttaskData{FlightID: in.FlightId}
	tid, err := l.svcCtx.DjiClient.MediaHighestPriorityUploadFlighttask(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[media] highest priority upload flighttask failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
