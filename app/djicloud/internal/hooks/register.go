package hooks

import (
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/collection"
)

type RegisterDjiClientOptions struct {
	DB                 *gormx.DB
	OnlineCache        *collection.Cache
	PushCli            socketpush.SocketPushClient
	DisableOsdSQLTrace bool
}

func eventHandlerOptions(db *gormx.DB) []djisdk.ClientOption {
	return []djisdk.ClientOption{
		djisdk.WithFlightTaskProgressHandler(NewFlightTaskProgressHandler(db)),
		djisdk.WithFlightTaskReadyHandler(NewFlightTaskReadyHandler(db)),
		djisdk.WithReturnHomeInfoHandler(NewReturnHomeInfoHandler(db)),
		djisdk.WithCustomDataFromPsdkHandler(HandleCustomDataFromPsdkEvent),
		djisdk.WithCustomDataFromEsdkHandler(HandleCustomDataFromEsdkEvent),
		djisdk.WithOtaProgressHandler(HandleOtaProgressEvent),
		djisdk.WithHmsEventNotifyHandler(NewHmsEventNotifyHandler(db)),
		djisdk.WithRemoteLogFileUploadProgressHandler(NewRemoteLogFileUploadProgressHandler(db)),
	}
}

func telemetryHandlerOptions(db *gormx.DB, onlineCache *collection.Cache, pushCli socketpush.SocketPushClient, disableOsdSQLTrace bool) []djisdk.ClientOption {
	return []djisdk.ClientOption{
		djisdk.WithOsdHandler(NewOsdHandler(db, onlineCache, pushCli, disableOsdSQLTrace)),
		djisdk.WithStateHandler(NewStateTelemetryHandler(db, onlineCache, pushCli)),
		djisdk.WithStatusHandler(NewStatusHandler(db, onlineCache)),
	}
}

func requestHandlerOptions() []djisdk.ClientOption {
	return []djisdk.ClientOption{
		djisdk.WithRequestHandler(NewDeviceRequestHandler()),
	}
}

func onlineCheckerOption(onlineCache *collection.Cache) djisdk.ClientOption {
	return djisdk.WithOnlineChecker(func(gatewaySn string) bool { return IsOnline(onlineCache, gatewaySn) })
}

// WithDjiClientOptions 返回向 djisdk.Client 注入本包内全部 MQTT 上行处理与在线检查的 ClientOption 列表。
func WithDjiClientOptions(o RegisterDjiClientOptions) []djisdk.ClientOption {
	var opts []djisdk.ClientOption
	if o.DB != nil {
		opts = append(opts, eventHandlerOptions(o.DB)...)
		opts = append(opts, telemetryHandlerOptions(o.DB, o.OnlineCache, o.PushCli, o.DisableOsdSQLTrace)...)
	}
	opts = append(opts, requestHandlerOptions()...)
	if o.OnlineCache != nil {
		opts = append(opts, onlineCheckerOption(o.OnlineCache))
	}
	return opts
}
