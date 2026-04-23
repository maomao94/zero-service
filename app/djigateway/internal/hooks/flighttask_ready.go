package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// OnFlightTaskReady 任务就绪通知钩子。
// Topic: thing/product/{gateway_sn}/events
// Direction: up（设备→云平台）
// Method: flighttask_ready
// 机巢中有任务满足就绪条件时主动上报可执行的任务 ID 列表。
func OnFlightTaskReady(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) {
	logx.WithContext(ctx).Infof("[dji-gateway] flight task ready: gateway=%s flight_ids=%v", gatewaySn, data.FlightIDs)
}
