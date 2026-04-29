package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcIntervalPhotoSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcIntervalPhotoSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcIntervalPhotoSetLogic {
	return &DrcIntervalPhotoSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcIntervalPhotoSetLogic) DrcIntervalPhotoSet(in *djicloud.DrcIntervalPhotoSetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.DrcIntervalPhotoSetData{PayloadIndex: in.GetPayloadIndex(), Interval: in.GetInterval()}
	tid, err := l.svcCtx.DjiClient.DrcIntervalPhotoSet(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
