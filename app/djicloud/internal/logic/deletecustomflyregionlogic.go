package logic

import (
	"context"
	"fmt"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCustomFlyRegionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCustomFlyRegionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCustomFlyRegionLogic {
	return &DeleteCustomFlyRegionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteCustomFlyRegion 删除自定义飞行区。
// 清除目标设备下所有飞行区配置，然后触发设备同步清空。
func (l *DeleteCustomFlyRegionLogic) DeleteCustomFlyRegion(in *djicloud.DeleteCustomFlyRegionReq) (*djicloud.DeleteCustomFlyRegionRes, error) {
	gatewaySn := in.GetDeviceSn()

	var regions []gormmodel.DjiFlyRegion
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("gateway_sn = ?", gatewaySn).Find(&regions).Error; err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		return nil, fmt.Errorf("未找到飞行区配置")
	}

	if err := l.svcCtx.DB.WithContext(l.ctx).Where("gateway_sn = ?", gatewaySn).Delete(&gormmodel.DjiFlyRegion{}).Error; err != nil {
		return nil, err
	}

	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, gatewaySn)
	if err != nil {
		return &djicloud.DeleteCustomFlyRegionRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}

	return &djicloud.DeleteCustomFlyRegionRes{Code: 0, Message: "success", Tid: tid}, nil
}
