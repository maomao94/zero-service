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

// FlightAreasUpdate 触发自定义飞行区文件更新（仅通知信号）。
// flight_areas_update 为触发信号，不含文件数据；设备收到通知后通过 flight_areas_get 拉取文件。
func (l *FlightAreasUpdateLogic) FlightAreasUpdate(in *djicloud.FlightAreasUpdateReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, in.GetDeviceSn())
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
