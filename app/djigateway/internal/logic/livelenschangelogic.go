package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type LiveLensChangeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLiveLensChangeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LiveLensChangeLogic {
	return &LiveLensChangeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// LiveLensChange 切换直播镜头。
func (l *LiveLensChangeLogic) LiveLensChange(in *djigateway.LiveLensChangeReq) (*djigateway.CommonRes, error) {
	data := &djisdk.LiveLensChangeData{
		VideoID:   in.VideoId,
		VideoType: int(in.VideoType),
	}
	tid, err := l.svcCtx.DjiClient.LiveLensChange(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[live] live lens change failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
