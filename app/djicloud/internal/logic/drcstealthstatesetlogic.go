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

func (l *DrcStealthStateSetLogic) DrcStealthStateSet(in *djicloud.DrcStealthStateSetReq) (*djicloud.CommonRes, error) {
	data := &djisdk.DrcStealthStateSetData{StealthState: int(in.GetStealthState())}
	tid, err := l.svcCtx.DjiClient.DrcStealthStateSet(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
