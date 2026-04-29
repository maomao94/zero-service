package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcVideoResolutionSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcVideoResolutionSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcVideoResolutionSetLogic {
	return &DrcVideoResolutionSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcVideoResolutionSetLogic) DrcVideoResolutionSet(in *djicloud.DrcVideoResolutionSetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.DrcVideoResolutionSetData{PayloadIndex: in.GetPayloadIndex(), VideoResolution: in.GetVideoResolution()}
	tid, err := l.svcCtx.DjiClient.DrcVideoResolutionSet(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
