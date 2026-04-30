package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetDeviceDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDeviceDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceDetailLogic {
	return &GetDeviceDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetDeviceDetail 查询设备详情，聚合设备基础信息、OSD快照、State快照和拓扑信息。
func (l *GetDeviceDetailLogic) GetDeviceDetail(in *djicloud.DeviceSnReq) (*djicloud.DeviceDetailRes, error) {
	var device gormmodel.DjiDevice
	if err := l.svcCtx.DB.WithContext(l.ctx).
		Where("device_sn = ?", in.DeviceSn).
		First(&device).Error; err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	res := &djicloud.DeviceDetailRes{Device: toDeviceInfo(&device)}
	var osd gormmodel.DjiDeviceOsdSnapshot
	if err := l.svcCtx.DB.WithContext(l.ctx).
		Where("device_sn = ?", in.DeviceSn).
		First(&osd).Error; err == nil {
		res.Osd = toOsdSnapshot(&osd)
	}
	var state gormmodel.DjiDeviceStateSnapshot
	if err := l.svcCtx.DB.WithContext(l.ctx).
		Where("device_sn = ?", in.DeviceSn).
		First(&state).Error; err == nil {
		res.State = toStateSnapshot(&state)
	}
	var topo []gormmodel.DjiDeviceTopo
	if err := l.svcCtx.DB.WithContext(l.ctx).
		Where("gateway_sn = ? OR sub_device_sn = ?", in.DeviceSn, in.DeviceSn).
		Order("update_time DESC, id DESC").
		Find(&topo).Error; err == nil {
		appendTopoInfo(res, topo)
	}
	return res, nil
}

func appendTopoInfo(res *djicloud.DeviceDetailRes, topo []gormmodel.DjiDeviceTopo) {
	for i := range topo {
		res.Topo = append(res.Topo, &djicloud.DeviceTopoInfo{
			GatewaySn:        topo[i].GatewaySn,
			SubDeviceSn:      topo[i].SubDeviceSn,
			Domain:           topo[i].Domain,
			SubDeviceType:    int32(topo[i].SubDeviceType),
			SubDeviceSubType: int32(topo[i].SubDeviceSubType),
			SubDeviceIndex:   topo[i].SubDeviceIndex,
			ThingVersion:     topo[i].ThingVersion,
		})
	}
}
