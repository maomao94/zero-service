package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReturnSpecificHomeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReturnSpecificHomeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReturnSpecificHomeLogic {
	return &ReturnSpecificHomeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ReturnSpecificHome 返航至指定备降点。
func (l *ReturnSpecificHomeLogic) ReturnSpecificHome(in *djigateway.ReturnSpecificHomeReq) (*djigateway.CommonRes, error) {
	data := &djisdk.ReturnSpecificHomeData{
		Latitude:  in.Latitude,
		Longitude: in.Longitude,
		Height:    in.Height,
	}
	tid, err := l.svcCtx.DjiClient.ReturnSpecificHome(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[flight-control] return specific home failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
