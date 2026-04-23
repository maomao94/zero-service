package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// OnReturnHomeInfo 返航信息上报钩子。
// Topic: thing/product/{gateway_sn}/events
// Direction: up（设备→云平台）
// Method: return_home_info
// 设备返航时主动上报规划的返航路径信息。
func OnReturnHomeInfo(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) {
	logx.WithContext(ctx).Infof("[dji-gateway] return home info: gateway=%s flight_id=%s last_point_type=%d points=%d",
		gatewaySn, data.FlightID, data.LastPointType, len(data.PlannedPathPoints))
}
