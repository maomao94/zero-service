package logic

import (
	"database/sql"
	"time"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
)

func commandRes(tid string, err error) (*djicloud.CommonRes, error) {
	message, reasonCode, err := commandError(err)
	if err == nil {
		return &djicloud.CommonRes{
			Code:       -1,
			Message:    message,
			Tid:        tid,
			ReasonCode: reasonCode,
		}, nil
	}
	return nil, err
}

func commandError(err error) (string, int32, error) {
	if djiErr, ok := djisdk.IsDJIError(err); ok {
		return djiErr.Message, int32(djiErr.Code), nil
	}
	return "", 0, err
}

func drcError(err error, operation string) error {
	return tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, operation)
}

func okRes(tid string) *djicloud.CommonRes {
	return &djicloud.CommonRes{Code: 0, Message: "success", Tid: tid}
}

const deviceOnlineTTL = 60 * time.Second

func deviceOnlineExpired(m *gormmodel.DjiDevice, now time.Time) bool {
	return m == nil || !m.IsOnline || !m.LastOnlineAt.Valid || now.Sub(m.LastOnlineAt.Time) > deviceOnlineTTL
}

func timeMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func timeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return carbon.CreateFromStdTime(t).ToDateTimeMicroString()
}

func nullTimeMillis(t sql.NullTime) int64 {
	if !t.Valid {
		return 0
	}
	return timeMillis(t.Time)
}

func normalizePage(page, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func toDeviceInfo(m *gormmodel.DjiDevice) *djicloud.DeviceInfo {
	if m == nil {
		return nil
	}
	return &djicloud.DeviceInfo{
		Id:              m.Id,
		DeviceSn:        m.DeviceSn,
		GatewaySn:       m.GatewaySn,
		DeviceType:      m.DeviceType,
		DeviceName:      m.DeviceName,
		Alias:           m.Alias,
		GroupName:       m.GroupName,
		FirmwareVersion: m.FirmwareVersion,
		HardwareVersion: m.HardwareVersion,
		IsOnline:        m.IsOnline,
		FirstOnlineAt:   nullTimeMillis(m.FirstOnlineAt),
		LastOnlineAt:    nullTimeMillis(m.LastOnlineAt),
	}
}

func toOsdSnapshot(m *gormmodel.DjiDeviceOsdSnapshot) *djicloud.DeviceOsdSnapshot {
	if m == nil {
		return nil
	}
	return &djicloud.DeviceOsdSnapshot{
		DeviceSn:   m.DeviceSn,
		GatewaySn:  m.GatewaySn,
		RawJson:    m.RawJSON,
		ReportedAt: carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString(),
	}
}

func toStateSnapshot(m *gormmodel.DjiDeviceStateSnapshot) *djicloud.DeviceStateSnapshot {
	if m == nil {
		return nil
	}
	return &djicloud.DeviceStateSnapshot{
		DeviceSn:   m.DeviceSn,
		GatewaySn:  m.GatewaySn,
		RawJson:    m.RawJSON,
		ReportedAt: carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString(),
	}
}

func toTelemetrySnapshotBrief(m *gormmodel.DjiDeviceOsdSnapshot) *djicloud.DeviceTelemetrySnapshotBrief {
	if m == nil {
		return nil
	}
	return &djicloud.DeviceTelemetrySnapshotBrief{
		DeviceSn:   m.DeviceSn,
		GatewaySn:  m.GatewaySn,
		ReportedAt: carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString(),
	}
}

func toTelemetrySnapshotBriefFromState(m *gormmodel.DjiDeviceStateSnapshot) *djicloud.DeviceTelemetrySnapshotBrief {
	if m == nil {
		return nil
	}
	return &djicloud.DeviceTelemetrySnapshotBrief{
		DeviceSn:   m.DeviceSn,
		GatewaySn:  m.GatewaySn,
		ReportedAt: carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString(),
	}
}

func toTopoInfoList(items []gormmodel.DjiDeviceTopo) []*djicloud.DeviceTopoInfo {
	if len(items) == 0 {
		return nil
	}
	list := make([]*djicloud.DeviceTopoInfo, 0, len(items))
	for i := range items {
		list = append(list, &djicloud.DeviceTopoInfo{
			GatewaySn:        items[i].GatewaySn,
			SubDeviceSn:      items[i].SubDeviceSn,
			Domain:           items[i].Domain,
			SubDeviceType:    int32(items[i].SubDeviceType),
			SubDeviceSubType: int32(items[i].SubDeviceSubType),
			SubDeviceIndex:   items[i].SubDeviceIndex,
			ThingVersion:     items[i].ThingVersion,
			DeviceType:       items[i].DeviceType,
			DeviceName:       items[i].DeviceName,
		})
	}
	return list
}

func toDockFlightTaskStateInfo(m *gormmodel.DjiDockDeviceFlightTaskState) *djicloud.DockFlightTaskStateInfo {
	if m == nil {
		return nil
	}
	trackId := ""
	if m.TrackId.Valid {
		trackId = m.TrackId.String
	}
	return &djicloud.DockFlightTaskStateInfo{
		GatewaySn:            m.GatewaySn,
		FlightId:             m.FlightId,
		Status:               m.Status,
		CurrentStep:          int32(m.CurrentStep),
		WaylineMissionState:  int32(m.WaylineMissionState),
		CurrentWaypointIndex: int32(m.CurrentWaypointIndex),
		MediaCount:           int32(m.MediaCount),
		ProgressPercent:      m.ProgressPercent,
		TrackId:              trackId,
		WaylineId:            int32(m.WaylineId),
		ReportedAt:           carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString(),
	}
}
