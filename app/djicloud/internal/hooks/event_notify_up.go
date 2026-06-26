package hooks

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
)

func NewFlightTaskProgressHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) error {
	return func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskProgressEvent) error {
		if data == nil {
			return nil
		}
		ext := data.Ext
		progress := data.Progress
		if gatewaySn == "" || ext.FlightID == "" {
			logx.WithContext(ctx).Errorf("[dji-cloud] skip flighttask_progress with empty identity: sn=%s flight_id=%s", gatewaySn, ext.FlightID)
			return nil
		}
		logx.WithContext(ctx).Infof("[dji-cloud] flighttask_progress: sn=%s flight_id=%s state=%d(%s) waypoint=%d media_count=%d percent=%f",
			gatewaySn, ext.FlightID, ext.WaylineMissionState, waylineMissionStateText(ext.WaylineMissionState), ext.CurrentWaypointIndex, ext.MediaCount, progress.Percent)
		trackID := sql.NullString{String: ext.TrackID, Valid: ext.TrackID != ""}
		if trackID.Valid {
			logx.WithContext(ctx).Errorf("[dji-cloud] flighttask_progress with invalid track_id: sn=%s flight_id=%s track_id is null", gatewaySn, ext.FlightID)
		}
		record := gormmodel.DjiDockFlightTask{
			FlightId:             ext.FlightID,
			GatewaySn:            gatewaySn,
			Status:               data.Status,
			CurrentStep:          progress.CurrentStep,
			WaylineMissionState:  ext.WaylineMissionState,
			CurrentWaypointIndex: ext.CurrentWaypointIndex,
			MediaCount:           ext.MediaCount,
			ProgressPercent:      progress.Percent,
			TrackId:              trackID,
			WaylineId:            ext.WaylineID,
			BreakPointJSON:       toJSONString(ext.BreakPoint),
			RawJSON:              toJSONString(data),
			ExtJSON:              toJSONString(ext),
			ReportedAt:           time.Now(),
		}
		updateData := map[string]any{
			"status":                 record.Status,
			"current_step":           record.CurrentStep,
			"wayline_mission_state":  record.WaylineMissionState,
			"current_waypoint_index": record.CurrentWaypointIndex,
			"media_count":            record.MediaCount,
			"progress_percent":       record.ProgressPercent,
			"track_id":               record.TrackId,
			"wayline_id":             record.WaylineId,
			"break_point_json":       record.BreakPointJSON,
			"raw_json":               record.RawJSON,
			"ext_json":               record.ExtJSON,
			"reported_at":            record.ReportedAt,
		}
		if err := db.Transact(func(tx *gormx.DB) error {
			c := tx.WithContext(ctx)
			if err := c.Where(map[string]any{"gateway_sn": gatewaySn, "flight_id": ext.FlightID}).Assign(updateData).FirstOrCreate(&record).Error; err != nil {
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
				RawJSON:              record.RawJSON,
				ExtJSON:              record.ExtJSON,
				ReportedAt:           record.ReportedAt,
			}
			deviceStateUpdateData := map[string]any{
				"flight_id":              deviceState.FlightId,
				"status":                 deviceState.Status,
				"current_step":           deviceState.CurrentStep,
				"wayline_mission_state":  deviceState.WaylineMissionState,
				"current_waypoint_index": deviceState.CurrentWaypointIndex,
				"media_count":            deviceState.MediaCount,
				"progress_percent":       deviceState.ProgressPercent,
				"track_id":               deviceState.TrackId,
				"wayline_id":             deviceState.WaylineId,
				"break_point_json":       deviceState.BreakPointJSON,
				"raw_json":               deviceState.RawJSON,
				"ext_json":               deviceState.ExtJSON,
				"reported_at":            deviceState.ReportedAt,
			}
			return c.Where(map[string]any{"gateway_sn": gatewaySn}).Assign(deviceStateUpdateData).FirstOrCreate(&deviceState).Error
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert flight task progress state failed: %v", err)
		}
		return nil
	}
}

func NewFlightTaskReadyHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) error {
	return func(ctx context.Context, gatewaySn string, data *djisdk.FlightTaskReadyEvent) error {
		if data == nil {
			return nil
		}
		logx.WithContext(ctx).Infof("[dji-cloud] flighttask_ready: sn=%s flight_ids=%v count=%d", gatewaySn, data.FlightIDs, len(data.FlightIDs))
		flightIdJSON := toJSONString(data.FlightIDs)
		rawJSON := toJSONString(data)
		if err := db.WithContext(ctx).Create(&gormmodel.DjiFlightTaskReady{
			GatewaySn:    gatewaySn,
			FlightIdJSON: flightIdJSON,
			RawJSON:      rawJSON,
			FlightCount:  len(data.FlightIDs),
			ReportedAt:   time.Now(),
		}).Error; err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create flight task ready event failed: %v", err)
		}
		return nil
	}
}

func NewReturnHomeInfoHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) error {
	return func(ctx context.Context, gatewaySn string, data *djisdk.ReturnHomeInfoEvent) error {
		if data == nil {
			return nil
		}
		logx.WithContext(ctx).Infof("[dji-cloud] return_home_info: sn=%s %+v", gatewaySn, *data)
		if err := db.WithContext(ctx).Create(&gormmodel.DjiReturnHomeEvent{
			FlightId:              data.FlightID,
			GatewaySn:             gatewaySn,
			ReportedAt:            time.Now(),
			RawJSON:               toJSONString(data),
			HomeDockSn:            data.HomeDockSn,
			LastPointType:         data.LastPointType,
			PlannedPathPointCount: len(data.PlannedPathPoints),
		}).Error; err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create return home event failed: %v", err)
		}
		return nil
	}
}

func HandleCustomDataFromPsdkEvent(ctx context.Context, gatewaySn string, data *djisdk.CustomDataFromPsdkEvent) error {
	if data == nil {
		return nil
	}
	logx.WithContext(ctx).Infof("[dji-cloud] custom_data_from_psdk: sn=%s value=%s", gatewaySn, data.Value)
	return nil
}

func NewHmsEventNotifyHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.HmsEventData) error {
	return func(ctx context.Context, gatewaySn string, data *djisdk.HmsEventData) error {
		if data == nil {
			return nil
		}
		logx.WithContext(ctx).Infof("[dji-cloud] hms: sn=%s items=%d", gatewaySn, len(data.List))
		for _, item := range data.List {
			logx.WithContext(ctx).Infof("[dji-cloud] hms item: level=%d module=%d in_the_sky=%d code=%s device_type=%s imminent=%d component_index=%d sensor_index=%d",
				item.Level, item.Module, item.InTheSky, item.Code, item.DeviceType, item.Imminent, item.Args.ComponentIndex, item.Args.SensorIndex)
			if err := db.WithContext(ctx).Create(&gormmodel.DjiHmsAlert{
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
			}).Error; err != nil {
				logx.WithContext(ctx).Errorf("[dji-cloud] create hms alert failed: %v", err)
			}
		}
		return nil
	}
}

func NewRemoteLogFileUploadProgressHandler(db *gormx.DB) func(ctx context.Context, gatewaySn string, data *djisdk.RemoteLogFileUploadProgressEvent) error {
	return func(ctx context.Context, gatewaySn string, data *djisdk.RemoteLogFileUploadProgressEvent) error {
		if data == nil {
			return nil
		}
		logx.WithContext(ctx).Infof("[dji-cloud] remote_log_fileupload_progress: sn=%s file_count=%d", gatewaySn, len(data.Files))
		if err := db.WithContext(ctx).Create(&gormmodel.DjiRemoteLogEvent{
			GatewaySn:  gatewaySn,
			Method:     "fileupload_progress",
			RawJSON:    toJSONString(data),
			FileCount:  len(data.Files),
			ReportedAt: time.Now(),
		}).Error; err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create remote log progress event failed: %v", err)
		}
		return nil
	}
}

// HandleOtaProgressEvent 处理固件升级进度事件（OTA Progress）。
// 对应 DJI Cloud API event method: ota_progress（Events, up）。
func HandleOtaProgressEvent(ctx context.Context, gatewaySn string, data *djisdk.OtaProgressEvent) error {
	if data == nil {
		return nil
	}
	logx.WithContext(ctx).Infof("[dji-cloud] ota_progress: sn=%s device_count=%d", gatewaySn, len(data.Devices))
	for _, dev := range data.Devices {
		logx.WithContext(ctx).Infof("[dji-cloud] ota_progress device: sn=%s model=%s status=%d progress=%d result=%d",
			dev.SN, dev.DeviceModel, dev.Status, dev.Progress, dev.Result)
	}
	return nil
}

// HandleCustomDataFromEsdkEvent 处理 ESDK 自定义数据上报事件。
// 对应 DJI Cloud API event method: custom_data_transmission_from_esdk（Events, up）。
func HandleCustomDataFromEsdkEvent(ctx context.Context, gatewaySn string, data *djisdk.CustomDataFromEsdkEvent) error {
	if data == nil {
		return nil
	}
	logx.WithContext(ctx).Infof("[dji-cloud] custom_data_from_esdk: sn=%s value=%s", gatewaySn, data.Value)
	return nil
}
