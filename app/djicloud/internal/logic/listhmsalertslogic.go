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

type ListHmsAlertsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListHmsAlertsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListHmsAlertsLogic {
	return &ListHmsAlertsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListHmsAlerts 查询 HMS 告警记录。
func (l *ListHmsAlertsLogic) ListHmsAlerts(in *djicloud.ListHmsAlertsReq) (*djicloud.ListHmsAlertsRes, error) {
	page, pageSize := normalizePage(in.GetPage(), in.GetPageSize())
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiHmsAlert{})
	if in.GatewaySn != "" {
		db = db.Where("gateway_sn = ?", in.GatewaySn)
	}
	if in.DeviceSn != "" {
		return &djicloud.ListHmsAlertsRes{}, nil
	}
	if in.Level > 0 {
		db = db.Where("level = ?", in.Level)
	}
	if in.AckedStatus == 1 || in.AckedStatus == 2 {
		db = db.Where("acked = ?", in.AckedStatus == 2)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var alerts []gormmodel.DjiHmsAlert
	if err := db.Order("reported_at DESC,id DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&alerts).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	list := make([]*djicloud.HmsAlertInfo, 0, len(alerts))
	for i := range alerts {
		item := alerts[i]
		list = append(list, &djicloud.HmsAlertInfo{
			Id:             item.Id,
			GatewaySn:      item.GatewaySn,
			Level:          int32(item.Level),
			Module:         int32(item.Module),
			Code:           item.Code,
			DeviceType:     item.DeviceType,
			Imminent:       int32(item.Imminent),
			InTheSky:       int32(item.InTheSky),
			ComponentIndex: int32(item.ComponentIndex),
			SensorIndex:    int32(item.SensorIndex),
			Acked:          int32(item.Acked),
			AckedAt:        nullTimeMillis(item.AckedAt),
			AckedBy:        item.AckedBy,
			ReportedAt:     timeMillis(item.ReportedAt),
			ItemJson:       item.ItemJSON,
		})
	}
	return &djicloud.ListHmsAlertsRes{Total: total, List: list}, nil
}
