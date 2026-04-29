package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type FlightAreasUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFlightAreasUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FlightAreasUpdateLogic {
	return &FlightAreasUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// FlightAreasUpdate 触发自定义飞行区文件更新。
func (l *FlightAreasUpdateLogic) FlightAreasUpdate(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, in.GetDeviceSn())
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
