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

func (l *DrcCameraDewarpingSetLogic) DrcCameraDewarpingSet(in *djicloud.DrcCameraDewarpingSetReq) (*djicloud.DrcCameraDewarpingSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcCameraDewarpingSetData{PayloadIndex: in.GetPayloadIndex(), CameraType: in.GetCameraType(), DewarpingState: int(in.GetDewarpingState())}
	if _, err := l.svcCtx.DjiClient.DrcCameraDewarpingSet(l.ctx, deviceSn, seq, data); err != nil {
		return nil, err
	}
	return &djicloud.DrcCameraDewarpingSetRes{Seq: int32(seq)}, nil
}
