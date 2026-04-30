package hooks

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type deviceVersions struct {
	FirmwareVersion string
	HardwareVersion string
}

func reportTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Now()
	}
	return time.UnixMilli(ms)
}

func toJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func sqlNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

func extractDeviceVersions(v any) deviceVersions {
	raw, err := json.Marshal(v)
	if err != nil {
		return deviceVersions{}
	}
	var data struct {
		FirmwareVersion string `json:"firmware_version"`
		HardwareVersion string `json:"hardware_version"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return deviceVersions{}
	}
	return deviceVersions{
		FirmwareVersion: strings.TrimSpace(data.FirmwareVersion),
		HardwareVersion: strings.TrimSpace(data.HardwareVersion),
	}
}

func appendVersionUpdateColumns(columns []string, versions deviceVersions) []string {
	if versions.FirmwareVersion != "" {
		columns = append(columns, "firmware_version")
	}
	if versions.HardwareVersion != "" {
		columns = append(columns, "hardware_version")
	}
	return columns
}

func waylineMissionStateText(state int) string {
	switch state {
	case 0:
		return "disconnected"
	case 1:
		return "waypoint_unsupported"
	case 2:
		return "wayline_ready"
	case 3:
		return "wayline_uploading"
	case 4:
		return "wayline_prepared"
	case 5:
		return "entering_wayline"
	case 6:
		return "wayline_executing"
	case 7:
		return "wayline_interrupted"
	case 8:
		return "wayline_resuming"
	case 9:
		return "wayline_stopped"
	default:
		return "unknown"
	}
}
