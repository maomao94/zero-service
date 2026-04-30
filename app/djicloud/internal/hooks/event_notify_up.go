package hooks

import (
	"context"
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
		if gatewaySn == "" || ext.FlightID == "" {
			logx.WithContext(ctx).Errorf("[dji-cloud] skip flighttask_progress with empty identity: sn=%s flight_id=%s", gatewaySn, ext.FlightID)
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] flighttask_progress: sn=%s flight_id=%s state=%d(%s) waypoint=%d media_count=%d percent=%f",
			gatewaySn, ext.FlightID, ext.WaylineMissionState, waylineMissionStateText(ext.WaylineMissionState), ext.CurrentWaypointIndex, ext.MediaCount, progress.Percent)
		record := gormmodel.DjiDockFlightTask{
			FlightId:             ext.FlightID,
			GatewaySn:            gatewaySn,
			Status:               data.Status,
			CurrentStep:          progress.CurrentStep,
			WaylineMissionState:  ext.WaylineMissionState,
			CurrentWaypointIndex: ext.CurrentWaypointIndex,
			MediaCount:           ext.MediaCount,
			ProgressPercent:      progress.Percent,
			TrackId:              ext.TrackID,
			WaylineId:            ext.WaylineID,
			BreakPointJSON:       toJSONString(ext.BreakPoint),
			EventJSON:            toJSONString(data),
			ExtJSON:              toJSONString(ext),
			ReportedAt:           time.Now(),
		}
		updateColumns := []string{
			"status", "current_step", "wayline_mission_state", "current_waypoint_index", "media_count", "progress_percent", "track_id", "wayline_id", "break_point_json", "event_json", "ext_json", "reported_at", "update_time",
		}
		if err := db.Transact(func(tx *gormx.DB) error {
			if err := gormx.Upsert(ctx, tx, &record, gormx.Columns("gateway_sn", "flight_id"), updateColumns); err != nil {
				return err
			}
			deviceState := gormmodel.DjiDockDeviceFlightTaskState{
				FlightId:             record.FlightId,
				GatewaySn:            record.GatewaySn,
				Status:               record.Status,
				CurrentStep:          record.CurrentStep,
				WaylineMissionState:  record.WaylineMissionState,
				CurrentWaypointIndex: record.CurrentWaypointIndex,
				MediaCount:           record.MediaCount,
				ProgressPercent:      record.ProgressPercent,
				TrackId:              record.TrackId,
				WaylineId:            record.WaylineId,
				BreakPointJSON:       record.BreakPointJSON,
				EventJSON:            record.EventJSON,
				ExtJSON:              record.ExtJSON,
				ReportedAt:           record.ReportedAt,
			}
			return gormx.Upsert(ctx, tx, &deviceState, gormx.Columns("gateway_sn"), append([]string{"flight_id"}, updateColumns...))
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert flight task progress state failed: %v", err)
		}
	}
}

func NewFlightTaskReadyHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] flighttask_ready: sn=%s flight_ids=%v count=%d", gatewaySn, data.FlightIDs, len(data.FlightIDs))
		flightIdJSON := toJSONString(data.FlightIDs)
		eventJSON := toJSONString(data)
		if err := gormx.CreateRecord(ctx, db, &gormmodel.DjiFlightTaskReady{
			GatewaySn:    gatewaySn,
			FlightIdJSON: flightIdJSON,
			EventJSON:    eventJSON,
			FlightCount:  len(data.FlightIDs),
			ReportedAt:   time.Now(),
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create flight task ready event failed: %v", err)
		}
	}
}

func NewReturnHomeInfoHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] return_home_info: sn=%s %+v", gatewaySn, *data)
		if err := gormx.CreateRecord(ctx, db, &gormmodel.DjiReturnHomeEvent{
			FlightId:              data.FlightID,
			GatewaySn:             gatewaySn,
			ReportedAt:            time.Now(),
			EventJSON:             toJSONString(data),
			HomeDockSn:            data.HomeDockSn,
			LastPointType:         data.LastPointType,
			PlannedPathPointCount: len(data.PlannedPathPoints),
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create return home event failed: %v", err)
		}
	}
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
				ItemJSON:       toJSONString(item),
				ReportedAt:     time.Now(),
			}); err != nil {
				logx.WithContext(ctx).Errorf("[dji-cloud] create hms alert failed: %v", err)
			}
		}
	}
}

func NewRemoteLogFileUploadProgressHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.RemoteLogFileUploadProgressEvent) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.RemoteLogFileUploadProgressEvent) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] remote_log_fileupload_progress: sn=%s file_count=%d", gatewaySn, len(data.Files))
		if err := gormx.CreateRecord(ctx, db, &gormmodel.DjiRemoteLogEvent{
			GatewaySn:  gatewaySn,
			Method:     "fileupload_progress",
			EventJSON:  toJSONString(data),
			FileCount:  len(data.Files),
			ReportedAt: time.Now(),
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create remote log progress event failed: %v", err)
		}
	}
}
