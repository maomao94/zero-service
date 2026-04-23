package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// OnFlightTaskProgress 航线任务进度上报钩子。
// Topic: thing/product/{gateway_sn}/events
// Direction: up（设备→云平台）
// Method: flighttask_progress
// 机巢执行航线任务时主动定频上报进度，包含当前航点、任务状态、媒体文件数、断点信息等。
// 业务端可在此钩子中实现进度持久化、WebSocket 推送、告警触发等逻辑。
func OnFlightTaskProgress(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
	logx.WithContext(ctx).Infof("[dji-gateway] flight task progress: gateway=%s flight_id=%s state=%d waypoint=%d media_count=%d track_id=%s",
		gatewaySn, data.Ext.FlightID, data.Ext.WaylineMissionState, data.Ext.CurrentWaypointIndex, data.Ext.MediaCount, data.Ext.TrackID)

	if data.Ext.BreakPoint != nil {
		logx.WithContext(ctx).Infof("[dji-gateway] flight task break_point: index=%d state=%d progress=%.2f wayline_id=%d break_reason=%d",
			data.Ext.BreakPoint.Index, data.Ext.BreakPoint.State, data.Ext.BreakPoint.Progress, data.Ext.BreakPoint.WaylineID, data.Ext.BreakPoint.BreakReason)
	}
}
