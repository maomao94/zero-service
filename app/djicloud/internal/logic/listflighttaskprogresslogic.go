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

type ListFlightTaskProgressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFlightTaskProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFlightTaskProgressLogic {
	return &ListFlightTaskProgressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListFlightTaskProgress 查询机巢航线任务最新快照。
func (l *ListFlightTaskProgressLogic) ListFlightTaskProgress(in *djicloud.ListFlightTaskProgressReq) (*djicloud.ListFlightTaskProgressRes, error) {
	page, pageSize := normalizePage(in.GetPage(), in.GetPageSize())
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiDockFlightTask{})
	if in.GatewaySn != "" {
		db = db.Where("gateway_sn = ?", in.GatewaySn)
	}
	if in.FlightId != "" {
		db = db.Where("flight_id = ?", in.FlightId)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询航线任务总数失败")
	}
	var records []gormmodel.DjiDockFlightTask
	if err := db.Order("reported_at DESC,id DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&records).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询航线任务列表失败")
	}
	list := make([]*djicloud.FlightTaskProgressInfo, 0, len(records))
	for i := range records {
		item := records[i]
		list = append(list, &djicloud.FlightTaskProgressInfo{
			Id:                   item.Id,
			FlightId:             item.FlightId,
			GatewaySn:            item.GatewaySn,
			WaylineMissionState:  int32(item.WaylineMissionState),
			CurrentWaypointIndex: int32(item.CurrentWaypointIndex),
			MediaCount:           int32(item.MediaCount),
			ProgressPercent:      item.ProgressPercent,
			ExtJson:              item.ExtJSON,
			ReportedAt:           timeMillis(item.ReportedAt),
			Status:               item.Status,
			CurrentStep:          int32(item.CurrentStep),
			TrackId:              item.TrackId.String,
			WaylineId:            int32(item.WaylineId),
			BreakPointJson:       item.BreakPointJSON,
			RawJson:              item.RawJSON,
		})
	}
	return &djicloud.ListFlightTaskProgressRes{Total: total, List: list}, nil
}
