package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type FlightTaskUndoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFlightTaskUndoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FlightTaskUndoLogic {
	return &FlightTaskUndoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FlightTaskUndoLogic) FlightTaskUndo(in *djicloud.FlightTaskUndoReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.FlightTaskUndo(l.ctx, in.DeviceSn, in.FlightIds)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
