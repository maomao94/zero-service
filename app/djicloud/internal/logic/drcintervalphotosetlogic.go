package logic

import (
	"context"
	"strconv"

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

func (l *DrcIntervalPhotoSetLogic) DrcIntervalPhotoSet(in *djicloud.DrcIntervalPhotoSetReq) (*djicloud.DrcIntervalPhotoSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	interval, _ := strconv.ParseFloat(in.GetInterval(), 64)
	data := &djisdk.DrcIntervalPhotoSetData{PayloadIndex: in.GetPayloadIndex(), Interval: interval}
	if _, err := l.svcCtx.DjiClient.DrcIntervalPhotoSet(l.ctx, deviceSn, seq, data); err != nil {
		l.Errorf("[drc] interval photo set failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcIntervalPhotoSetRes{Seq: int32(seq)}, nil
}
