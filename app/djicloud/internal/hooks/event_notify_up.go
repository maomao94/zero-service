package hooks

import (
	"context"
	"encoding/json"
	"time"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
)

func NewFlightTaskProgressHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
		if data == nil {
			return
		}
		ext := data.Ext
		progress := data.Progress
		logx.WithContext(ctx).Infof("[dji-cloud] flighttask_progress: sn=%s flight_id=%s state=%d waypoint=%d media_count=%d percent=%f",
			gatewaySn, ext.FlightID, ext.WaylineMissionState, ext.CurrentWaypointIndex, ext.MediaCount, progress.Percent)
		if err := gormx.CreateRecord(ctx, db, &gormmodel.DjiFlightTaskProgress{
			FlightId:             ext.FlightID,
			GatewaySn:            gatewaySn,
			WaylineMissionState:  ext.WaylineMissionState,
			CurrentWaypointIndex: ext.CurrentWaypointIndex,
			MediaCount:           ext.MediaCount,
			ProgressPercent:      progress.Percent,
			ExtJSON:              toJSONString(data),
			ReportedAt:           time.Now(),
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create flight task progress failed: %v", err)
		}
	}
}

func HandleFlightTaskReadyEvent(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) {
	if data == nil {
		return
	}
	logx.WithContext(ctx).Infof("[dji-cloud] flighttask_ready: sn=%s flight_ids=%v", gatewaySn, data.FlightIDs)
}

func NewReturnHomeInfoHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] return_home_info: sn=%s %+v", gatewaySn, *data)
		if err := gormx.CreateRecord(ctx, db, &gormmodel.DjiReturnHomeEvent{
			FlightId:               data.FlightID,
			GatewaySn:              gatewaySn,
			ReportedAt:             time.Now(),
			EventJSON:              toJSONString(data),
			HomeDockSn:             data.HomeDockSn,
			LastPointType:          data.LastPointType,
			PlannedPathPointCount:  len(data.PlannedPathPoints),
			MultiDockHomeInfoCount: len(data.MultiDockHomeInfo),
			NearestHomeDistance:    nearestHomeDistance(data.MultiDockHomeInfo),
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create return home event failed: %v", err)
		}
	}
}

func nearestHomeDistance(items []djisdk.DockHomeInfo) float64 {
	if len(items) == 0 {
		return 0
	}
	nearest := items[0].HomeDistance
	for i := 1; i < len(items); i++ {
		if items[i].HomeDistance < nearest {
			nearest = items[i].HomeDistance
		}
	}
	return nearest
}

func HandleCustomDataFromPsdkEvent(ctx context.Context, gatewaySn string, data *djisdk.CustomDataFromPsdkEvent) {
	if data == nil {
		return
	}
	logx.WithContext(ctx).Infof("[dji-cloud] custom_data_from_psdk: sn=%s value=%s", gatewaySn, data.Value)
}

func NewHmsEventNotifyHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.HmsEventData) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.HmsEventData) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] hms: sn=%s items=%d", gatewaySn, len(data.List))
		for _, item := range data.List {
			logx.WithContext(ctx).Infof("[dji-cloud] hms item: level=%d module=%d in_the_sky=%d code=%s device_type=%s imminent=%d component_index=%d sensor_index=%d",
				item.Level, item.Module, item.InTheSky, item.Code, item.DeviceType, item.Imminent, item.Args.ComponentIndex, item.Args.SensorIndex)
			if err := gormx.CreateRecord(ctx, db, &gormmodel.DjiHmsAlert{
				GatewaySn:      gatewaySn,
				Level:          item.Level,
				Module:         item.Module,
				Code:           item.Code,
				DeviceType:     item.DeviceType,
				Imminent:       item.Imminent,
				InTheSky:       item.InTheSky,
				ComponentIndex: item.Args.ComponentIndex,
				SensorIndex:    item.Args.SensorIndex,
				ReportedAt:     time.Now(),
			}); err != nil {
				logx.WithContext(ctx).Errorf("[dji-cloud] create hms alert failed: %v", err)
			}
		}
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
