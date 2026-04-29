package hooks

import (
	"context"
	"encoding/json"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

func NewFlightTaskProgressHandler(progressCache *collection.Cache) func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] flighttask_progress: sn=%s flight_id=%s state=%d waypoint=%d media_count=%d",
			gatewaySn, data.Ext.FlightID, data.Ext.WaylineMissionState, data.Ext.CurrentWaypointIndex, data.Ext.MediaCount)
		StoreFlightTaskProgressLast(progressCache, gatewaySn, data)
	}
}

func HandleFlightTaskReadyEvent(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) {
	if data == nil {
		return
	}
	logx.WithContext(ctx).Infof("[dji-cloud] flighttask_ready: sn=%s flight_ids=%v", gatewaySn, data.FlightIDs)
}

func HandleReturnHomeInfoEvent(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) {
	if data == nil {
		return
	}
	logx.WithContext(ctx).Infof("[dji-cloud] return_home_info: sn=%s %+v", gatewaySn, *data)
}

func HandleCustomDataFromPsdkEvent(ctx context.Context, gatewaySn string, data *djisdk.CustomDataFromPsdkEvent) {
	if data == nil {
		return
	}
	logx.WithContext(ctx).Infof("[dji-cloud] custom_data_from_psdk: sn=%s value=%s", gatewaySn, data.Value)
}

func HandleHmsEventNotify(ctx context.Context, gatewaySn string, data *djisdk.HmsEventData) {
	if data == nil {
		return
	}
	logx.WithContext(ctx).Infof("[dji-cloud] hms: sn=%s items=%d", gatewaySn, len(data.List))
	for _, item := range data.List {
		logx.WithContext(ctx).Infof("[dji-cloud] hms item: level=%d module=%d in_the_sky=%d code=%s device_type=%s imminent=%d component_index=%d sensor_index=%d",
			item.Level, item.Module, item.InTheSky, item.Code, item.DeviceType, item.Imminent, item.Args.ComponentIndex, item.Args.SensorIndex)
	}
}

func HandleRemoteLogFileUploadResultEvent(ctx context.Context, gatewaySn string, data *djisdk.RemoteLogFileUploadResultEvent) {
	if data == nil {
		return
	}
	summary, _ := json.Marshal(data.Files)
	logx.WithContext(ctx).Infof("[dji-cloud] remote_log_fileupload_result: sn=%s files=%s", gatewaySn, string(summary))
}

func HandleRemoteLogFileUploadProgressEvent(ctx context.Context, gatewaySn string, data *djisdk.RemoteLogFileUploadProgressEvent) {
	if data == nil {
		return
	}
	summary, _ := json.Marshal(data.Files)
	logx.WithContext(ctx).Infof("[dji-cloud] remote_log_fileupload_progress: sn=%s files=%s", gatewaySn, string(summary))
}
