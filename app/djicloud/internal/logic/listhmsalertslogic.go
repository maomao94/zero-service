package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
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
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiHmsAlert{}).
		Table("dji_hms_alert dha").
		Joins("LEFT JOIN dji_device dd ON dha.gateway_sn = dd.gateway_sn")
	if in.GatewaySn != "" {
		db = db.Where("dd.gateway_sn = ?", in.GatewaySn)
	}
	if in.Level > 0 {
		db = db.Where("dha.level = ?", in.Level)
	}
	if in.AckedStatus == 0 || in.AckedStatus == 1 {
		db = db.Where("dha.acked = ?", in.AckedStatus)
	}
	var alerts []gormmodel.DjiHmsAlert
	queryDB := db.WithContext(l.ctx).Order("dha.reported_at DESC,dha.id DESC")
	queryDB = queryDB.Select("dha.*, dd.is_online")
	pageResult, err := gormx.QueryPage(queryDB, int(in.GetPage()), int(in.GetPageSize()), &alerts)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询HMS告警列表失败")
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
			ReportedAt:     carbon.CreateFromStdTime(item.ReportedAt).ToDateTimeMicroString(),
			ItemJson:       item.ItemJSON,
		})
	}
	return &djicloud.ListHmsAlertsRes{Total: pageResult.Total, List: list}, nil
}
