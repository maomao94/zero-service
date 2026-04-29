package hooks

import (
	"encoding/json"
	"time"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
)

type flightTaskProgressLastEntry struct {
	CachedAtMs   int64
	ProgressJSON string
}

// StoreFlightTaskProgressLast 将最近一次 flighttask_progress 写入内存缓存（按 gateway_sn）。
func StoreFlightTaskProgressLast(c *collection.Cache, gatewaySn string, data *djisdk.FlightTaskProgressEvent) {
	if c == nil || gatewaySn == "" || data == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	c.Set(gatewaySn, flightTaskProgressLastEntry{
		CachedAtMs:   time.Now().UnixMilli(),
		ProgressJSON: string(b),
	})
}

// GetFlightTaskProgressLast 读取最近一次缓存；无记录时 has=false。
func GetFlightTaskProgressLast(c *collection.Cache, gatewaySn string) (has bool, cachedAtMs int64, progressJSON string) {
	if c == nil || gatewaySn == "" {
		return false, 0, ""
	}
	v, ok := c.Get(gatewaySn)
	if !ok {
		return false, 0, ""
	}
	e, ok := v.(flightTaskProgressLastEntry)
	if !ok {
		return false, 0, ""
	}
	return true, e.CachedAtMs, e.ProgressJSON
}
