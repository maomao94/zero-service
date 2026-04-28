package hooks

import (
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
)

// RegisterDjiClientOptions 为 RegisterDjiClient 的依赖：在线缓存与航线进度缓存由 svc 层创建并注入。
type RegisterDjiClientOptions struct {
	OnlineCache         *collection.Cache
	FlightProgressCache *collection.Cache
}

func registerEventHandlers(c *djisdk.Client, progressCache *collection.Cache) {
	c.OnFlightTaskProgress(NewFlightTaskProgressHandler(progressCache))
	c.OnFlightTaskReady(HandleFlightTaskReadyEvent)
	c.OnReturnHomeInfo(HandleReturnHomeInfoEvent)
	c.OnCustomDataFromPsdk(HandleCustomDataFromPsdkEvent)
	c.OnHmsEventNotify(HandleHmsEventNotify)
}

func registerTelemetryHandlers(c *djisdk.Client, onlineCache *collection.Cache) {
	c.OnOsd(NewOsdHandler(onlineCache))
	c.OnState(HandleStateTelemetry)
	c.OnStatus(NewStatusHandler(onlineCache))
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
	registerEventHandlers(c, o.FlightProgressCache)
	registerTelemetryHandlers(c, o.OnlineCache)
	registerRequestHandlers(c)
	registerOnlineChecker(c, o.OnlineCache)
}
