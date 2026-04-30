package logic

import (
	"database/sql"
	"time"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
)

func errRes(tid string, err error) *djicloud.CommonRes {
	if djiErr, ok := djisdk.IsDJIError(err); ok {
		return &djicloud.CommonRes{
			Code:       -1,
			Message:    djiErr.Message,
			Tid:        tid,
			ReasonCode: int32(djiErr.Code),
		}
	}
	return &djicloud.CommonRes{Code: -1, Message: err.Error(), Tid: tid}
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
		DeviceDomain:    m.DeviceDomain,
		DeviceType:      int32(m.DeviceType),
		DeviceSubType:   int32(m.DeviceSubType),
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
		DeviceSn:     m.DeviceSn,
		GatewaySn:    m.GatewaySn,
		DeviceDomain: m.DeviceDomain,
		DataJson:     m.DataJSON,
		ReportedAt:   timeMillis(m.ReportedAt),
	}
}

func toStateSnapshot(m *gormmodel.DjiDeviceStateSnapshot) *djicloud.DeviceStateSnapshot {
	if m == nil {
		return nil
	}
	return &djicloud.DeviceStateSnapshot{
		DeviceSn:     m.DeviceSn,
		GatewaySn:    m.GatewaySn,
		DeviceDomain: m.DeviceDomain,
		DataJson:     m.DataJSON,
		ReportedAt:   timeMillis(m.ReportedAt),
	}
}
