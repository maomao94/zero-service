package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendDrcCommandLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendDrcCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendDrcCommandLogic {
	return &SendDrcCommandLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendDrcCommandLogic) SendDrcCommand(in *djigateway.DroneControlReq) (*djigateway.CommonRes, error) {
	data := &djisdk.DroneControlData{
		X:   in.X,
		Y:   in.Y,
		H:   in.H,
		W:   in.W,
		Seq: int(in.Seq),
	}
	err := l.svcCtx.DjiClient.SendDrcCommand(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[drc] send drc command failed: %v", err)
		return errRes("", err), nil
	}
	return okRes(""), nil
}
