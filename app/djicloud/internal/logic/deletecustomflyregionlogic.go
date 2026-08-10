package logic

import (
	"context"
	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询飞行区配置失败")
	}
	if len(regions) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "未找到飞行区配置")
	}

	if err := l.svcCtx.DB.WithContext(l.ctx).Where("gateway_sn = ?", gatewaySn).Delete(&gormmodel.DjiFlyRegion{}).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "删除飞行区配置失败")
	}

	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, gatewaySn)
	if err != nil {
		message, reasonCode, err := commandError(err)
		if err != nil {
			return nil, err
		}
		return &djicloud.DeleteCustomFlyRegionRes{Code: -1, Message: message, Tid: tid, ReasonCode: reasonCode}, nil
	}

	return &djicloud.DeleteCustomFlyRegionRes{Code: 0, Message: "success", Tid: tid}, nil
}
