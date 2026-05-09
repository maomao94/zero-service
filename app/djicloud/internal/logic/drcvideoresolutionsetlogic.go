package logic

import (
	"context"
	"strconv"

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

func (l *DrcVideoResolutionSetLogic) DrcVideoResolutionSet(in *djicloud.DrcVideoResolutionSetReq) (*djicloud.DrcVideoResolutionSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DrcManager.GetNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	videoRes, _ := strconv.Atoi(in.GetVideoResolution())
	data := &djisdk.DrcVideoResolutionSetData{PayloadIndex: in.GetPayloadIndex(), VideoResolution: videoRes}
	if _, err := l.svcCtx.DjiClient.DrcVideoResolutionSet(l.ctx, deviceSn, seq, data); err != nil {
		l.Errorf("[drc] video resolution set failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcVideoResolutionSetRes{Seq: int32(seq)}, nil
}
