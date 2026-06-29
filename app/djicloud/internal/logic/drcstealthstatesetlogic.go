package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcStealthStateSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcStealthStateSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcStealthStateSetLogic {
	return &DrcStealthStateSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcStealthStateSetLogic) DrcStealthStateSet(in *djicloud.DrcStealthStateSetReq) (*djicloud.DrcStealthStateSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcStealthStateSetData{StealthState: int(in.GetStealthState())}
	if _, err := l.svcCtx.DjiClient.DrcStealthStateSet(l.ctx, deviceSn, seq, data); err != nil {
		return nil, err
	}
	return &djicloud.DrcStealthStateSetRes{Seq: int32(seq)}, nil
}
