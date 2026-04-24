package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type LiveSetQualityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLiveSetQualityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LiveSetQualityLogic {
	return &LiveSetQualityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LiveSetQualityLogic) LiveSetQuality(in *djigateway.LiveSetQualityReq) (*djigateway.CommonRes, error) {
	data := &djisdk.LiveSetQualityData{
		VideoID:      in.VideoId,
		VideoQuality: int(in.VideoQuality),
	}
	tid, err := l.svcCtx.DjiClient.LiveSetQuality(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[live] live set quality failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
