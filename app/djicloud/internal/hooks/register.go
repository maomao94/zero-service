package hooks

import (
	"zero-service/app/djicloud/internal/drc"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/collection"
)

// RegisterDjiClientOptions 为 RegisterDjiClient 的依赖，在线缓存由 svc 层创建并注入。
type RegisterDjiClientOptions struct {
	DB          *gormx.DB
	OnlineCache *collection.Cache
	DrcManager  *drc.Manager
	PushCli     socketpush.SocketPushClient
}

func registerEventHandlers(c *djisdk.Client, db *gormx.DB) {
	c.OnFlightTaskProgress(NewFlightTaskProgressHandler(db))
	c.OnFlightTaskReady(NewFlightTaskReadyHandler(db))
	c.OnReturnHomeInfo(NewReturnHomeInfoHandler(db))
	c.OnCustomDataFromPsdk(HandleCustomDataFromPsdkEvent)
	c.OnCustomDataFromEsdk(HandleCustomDataFromEsdkEvent)
	c.OnOtaProgress(HandleOtaProgressEvent)
	c.OnHmsEventNotify(NewHmsEventNotifyHandler(db))
	c.OnRemoteLogFileUploadProgress(NewRemoteLogFileUploadProgressHandler(db))
}

func registerTelemetryHandlers(c *djisdk.Client, db *gormx.DB, onlineCache *collection.Cache, drcMgr *drc.Manager, pushCli socketpush.SocketPushClient) {
	c.OnOsd(NewOsdHandler(db, onlineCache))
	c.OnState(NewStateTelemetryHandler(db, onlineCache))
	c.OnStatus(NewStatusHandler(db, onlineCache))
	c.OnDrcUp(NewDrcUpHandler(db, drcMgr, pushCli))
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
	registerTelemetryHandlers(c, o.DB, o.OnlineCache, o.DrcManager, o.PushCli)
	registerRequestHandlers(c)
	registerOnlineChecker(c, o.OnlineCache)
}
