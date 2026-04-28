package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewFlightTaskProgressHandler 构造 method=flighttask_progress 的 events 上行处理器。
func NewFlightTaskProgressHandler(progressCache *collection.Cache) func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
		if data == nil {
			return
		}
		StoreFlightTaskProgressLast(progressCache, gatewaySn, data)

		logx.WithContext(ctx).Infof("[dji-gateway] flight task progress: sn=%s flight_id=%s state=%d waypoint=%d media_count=%d track_id=%s",
			gatewaySn, data.Ext.FlightID, data.Ext.WaylineMissionState, data.Ext.CurrentWaypointIndex, data.Ext.MediaCount, data.Ext.TrackID)

		if data.Ext.BreakPoint != nil {
			logx.WithContext(ctx).Infof("[dji-gateway] flight task break_point: index=%d state=%d progress=%.2f wayline_id=%d break_reason=%d",
				data.Ext.BreakPoint.Index, data.Ext.BreakPoint.State, data.Ext.BreakPoint.Progress, data.Ext.BreakPoint.WaylineID, data.Ext.BreakPoint.BreakReason)
		}
	}
}
