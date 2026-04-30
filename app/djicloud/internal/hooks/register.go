package hooks

import (
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/collection"
)

// RegisterDjiClientOptions 为 RegisterDjiClient 的依赖，在线缓存由 svc 层创建并注入。
type RegisterDjiClientOptions struct {
	DB          *gormx.DB
	OnlineCache *collection.Cache
}

func registerEventHandlers(c *djisdk.Client, db *gormx.DB) {
	c.OnFlightTaskProgress(NewFlightTaskProgressHandler(db))
	c.OnFlightTaskReady(NewFlightTaskReadyHandler(db))
	c.OnReturnHomeInfo(NewReturnHomeInfoHandler(db))
	c.OnCustomDataFromPsdk(HandleCustomDataFromPsdkEvent)
	c.OnHmsEventNotify(NewHmsEventNotifyHandler(db))
	c.OnRemoteLogFileUploadProgress(NewRemoteLogFileUploadProgressHandler(db))
}

func registerTelemetryHandlers(c *djisdk.Client, db *gormx.DB, onlineCache *collection.Cache) {
	c.OnOsd(NewOsdHandler(db, onlineCache))
	c.OnState(NewStateTelemetryHandler(db, onlineCache))
	c.OnStatus(NewStatusHandler(db, onlineCache))
	c.OnDrcUp(NewDrcUpHandler(onlineCache))
}

func registerRequestHandlers(c *djisdk.Client) {
	c.OnRequest(NewDeviceRequestHandler())
}

func registerOnlineChecker(c *djisdk.Client, onlineCache *collection.Cache) {
	if onlineCache == nil {
		return
	}
	c.SetOnlineChecker(func(gatewaySn string) bool { return IsOnline(onlineCache, gatewaySn) })
}

// RegisterDjiClient 向 djisdk 注册本包内全部 MQTT 上行处理与在线检查。
func RegisterDjiClient(c *djisdk.Client, o RegisterDjiClientOptions) {
	if c == nil {
		return
	}
	registerEventHandlers(c, o.DB)
	registerTelemetryHandlers(c, o.DB, o.OnlineCache)
	registerRequestHandlers(c)
	registerOnlineChecker(c, o.OnlineCache)
}
