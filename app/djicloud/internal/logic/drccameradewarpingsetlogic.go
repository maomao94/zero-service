package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcCameraDewarpingSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcCameraDewarpingSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcCameraDewarpingSetLogic {
	return &DrcCameraDewarpingSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcCameraDewarpingSetLogic) DrcCameraDewarpingSet(in *djicloud.DrcCameraDewarpingSetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.DrcCameraDewarpingSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), DewarpingState: int(in.GetDewarpingState())}
	tid, err := l.svcCtx.DjiClient.DrcCameraDewarpingSet(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
