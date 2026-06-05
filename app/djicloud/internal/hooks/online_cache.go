package hooks

import (
	"github.com/zeromicro/go-zero/core/collection"
)

// OnlineValue 为 go-zero collection.Cache 中表示「在 MQTT/业务意义上在线」的占位值。
// OSD、status 等链路上会 Set(gatewaySn, OnlineValue) 以刷新/建立在线态。
const OnlineValue = "1"

// IsOnline 根据与 hooks 同逻辑写入的 onlineCache 判断机巢/网关是否视为在线。
func IsOnline(onlineCache *collection.Cache, gatewaySn string) bool {
	if onlineCache == nil {
		return false
	}
	_, ok := onlineCache.Get(gatewaySn)
	return ok
}
