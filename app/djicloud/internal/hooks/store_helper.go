package hooks

import (
	"database/sql"
	"encoding/json"
	"time"
)

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
