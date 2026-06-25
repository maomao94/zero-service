package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcLinkageZoomSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcLinkageZoomSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcLinkageZoomSetLogic {
	return &DrcLinkageZoomSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcLinkageZoomSetLogic) DrcLinkageZoomSet(in *djicloud.DrcLinkageZoomSetReq) (*djicloud.DrcLinkageZoomSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcLinkageZoomSetData{PayloadIndex: in.GetPayloadIndex(), State: in.GetState()}
	if _, err := l.svcCtx.DjiClient.DrcLinkageZoomSet(l.ctx, deviceSn, seq, data); err != nil {
		l.Errorf("[drc] linkage zoom set failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcLinkageZoomSetRes{Seq: int32(seq)}, nil
}
