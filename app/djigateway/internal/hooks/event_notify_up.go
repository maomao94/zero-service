package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// HandleFlightTaskReadyEvent 由 events 上行、method=flighttask_ready 驱动。
func HandleFlightTaskReadyEvent(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) {
	logx.WithContext(ctx).Infof("[dji-gateway] flight task ready: sn=%s flight_ids=%v", gatewaySn, data.FlightIDs)
}

// HandleReturnHomeInfoEvent 由 events 上行、method=return_home_info 驱动。
func HandleReturnHomeInfoEvent(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) {
	logx.WithContext(ctx).Infof("[dji-gateway] return home info: sn=%s flight_id=%s last_point_type=%d points=%d",
		gatewaySn, data.FlightID, data.LastPointType, len(data.PlannedPathPoints))
}

// HandleCustomDataFromPsdkEvent 由 events 上行、method=custom_data_transmission_from_psdk 驱动。
func HandleCustomDataFromPsdkEvent(ctx context.Context, gatewaySn string, data *djisdk.CustomDataFromPsdkEvent) {
	logx.WithContext(ctx).Infof("[dji-gateway] psdk custom data: sn=%s value=%s", gatewaySn, data.Value)
}

// HandleHmsEventNotify 由 events 上行、method=hms 驱动。
func HandleHmsEventNotify(ctx context.Context, gatewaySn string, data *djisdk.HmsEventData) {
	logx.WithContext(ctx).Infof("[dji-gateway] hms event: sn=%s count=%d", gatewaySn, len(data.List))
	for _, item := range data.List {
		logx.WithContext(ctx).Infof("[dji-gateway] hms item: hms_id=%s level=%d module=%d in_the_sky=%d code=%s imminent=%v key=%s",
			item.HmsID, item.Level, item.Module, item.InTheSky, item.Code, item.Imminent, item.Key)
	}
}
