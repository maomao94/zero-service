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
		return nil, status.Error(codes.Internal, err.Error())
	}
	var records []gormmodel.DjiDockFlightTask
	if err := db.Order("reported_at DESC,id DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&records).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
			TrackId:              item.TrackId,
			WaylineId:            int32(item.WaylineId),
			BreakPointJson:       item.BreakPointJSON,
			EventJson:            item.EventJSON,
		})
	}
	return &djicloud.ListFlightTaskProgressRes{Total: total, List: list}, nil
}
