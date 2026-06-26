package djisdk

import (
	"context"
	"time"
)

// StatusHandler 处理 sys/.../status 上行。
// 返回 error 统一打印日志；若 error 为 *PlatformError 则取其 Code 作为 status_reply 的 result，否则默认 PlatformResultHandlerError。
// 是否发布 status_reply 由 Client 的 ReplyConfig 控制。
type StatusHandler func(ctx context.Context, gatewaySn string, data *StatusMessage) error

// DrcUpHandler 处理 thing/product/{gateway_sn}/drc/up 设备上行报文；parsed 为 DrcUnmarshalUpData 解析结果，未知 method 时为 *DrcUnknownUpData。
type DrcUpHandler func(ctx context.Context, gatewaySn string, msg *DrcUpMessage, parsed any) error

// RequestHandler 处理 thing/.../requests 上行。output 供 requests_reply 的 data.output 使用。
// 返回 error 统一打印日志；若 error 为 *PlatformError 则取其 Code 作为 requests_reply 的 result，否则默认 PlatformResultHandlerError。
// 是否发布 requests_reply 由 Client 的 ReplyConfig 控制。
type RequestHandler func(ctx context.Context, gatewaySn string, msg *RequestMessage) (output any, err error)

type ReplyConfig struct {
	EnableEventReply   bool `json:",default=true"`
	EnableStatusReply  bool `json:",default=true"`
	EnableRequestReply bool `json:",default=true"`
}

func DefaultReplyConfig() ReplyConfig {
	return ReplyConfig{
		EnableEventReply:   true,
		EnableStatusReply:  true,
		EnableRequestReply: true,
	}
}

const defaultPendingTTL = 30 * time.Second

type handlers struct {
	onFlightTaskProgress             func(ctx context.Context, gatewaySn string, data *FlightTaskProgressEvent) error
	onFlightTaskReady                func(ctx context.Context, gatewaySn string, data *FlightTaskReadyEvent) error
	onReturnHomeInfo                 func(ctx context.Context, gatewaySn string, data *ReturnHomeInfoEvent) error
	onCustomDataTransmissionFromPsdk func(ctx context.Context, gatewaySn string, data *CustomDataFromPsdkEvent) error
	onCustomDataTransmissionFromEsdk func(ctx context.Context, gatewaySn string, data *CustomDataFromEsdkEvent) error
	onHmsEventNotify                 func(ctx context.Context, gatewaySn string, data *HmsEventData) error
	onRemoteLogFileUploadProgress    func(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadProgressEvent) error
	onOtaProgress                    func(ctx context.Context, gatewaySn string, data *OtaProgressEvent) error
	onUpdateTopo                     func(ctx context.Context, gatewaySn string, data *TopoUpdateData) error
	onOsd                            func(ctx context.Context, deviceSn string, data *OsdMessage) error
	onState                          func(ctx context.Context, deviceSn string, data *StateMessage) error
	onStatus                         StatusHandler
	onRequest                        RequestHandler
	onDrcUp                          DrcUpHandler
	onlineChecker                    func(gatewaySn string) bool
}

type ClientOption func(*clientOptions)

type clientOptions struct {
	handlers       handlers
	pendingTTL     time.Duration
	reply          ReplyConfig
	drcConfig      DrcConfig
	drcManagerOpts []drcManagerOption
}

func WithPendingTTL(ttl time.Duration) ClientOption {
	return func(options *clientOptions) {
		if ttl > 0 {
			options.pendingTTL = ttl
		}
	}
}

func WithReplyConfig(replyOptions ReplyConfig) ClientOption {
	return func(options *clientOptions) {
		options.reply = replyOptions
	}
}

func WithFlightTaskProgressHandler(handler func(ctx context.Context, gatewaySn string, data *FlightTaskProgressEvent) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onFlightTaskProgress = handler
	}
}

func WithFlightTaskReadyHandler(handler func(ctx context.Context, gatewaySn string, data *FlightTaskReadyEvent) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onFlightTaskReady = handler
	}
}

func WithReturnHomeInfoHandler(handler func(ctx context.Context, gatewaySn string, data *ReturnHomeInfoEvent) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onReturnHomeInfo = handler
	}
}

func WithCustomDataFromPsdkHandler(handler func(ctx context.Context, gatewaySn string, data *CustomDataFromPsdkEvent) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onCustomDataTransmissionFromPsdk = handler
	}
}

func WithCustomDataFromEsdkHandler(handler func(ctx context.Context, gatewaySn string, data *CustomDataFromEsdkEvent) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onCustomDataTransmissionFromEsdk = handler
	}
}

func WithHmsEventNotifyHandler(handler func(ctx context.Context, gatewaySn string, data *HmsEventData) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onHmsEventNotify = handler
	}
}

func WithRemoteLogFileUploadProgressHandler(handler func(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadProgressEvent) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onRemoteLogFileUploadProgress = handler
	}
}

func WithOtaProgressHandler(handler func(ctx context.Context, gatewaySn string, data *OtaProgressEvent) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onOtaProgress = handler
	}
}

func WithUpdateTopoHandler(handler func(ctx context.Context, gatewaySn string, data *TopoUpdateData) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onUpdateTopo = handler
	}
}

func WithOsdHandler(handler func(ctx context.Context, deviceSn string, data *OsdMessage) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onOsd = handler
	}
}

func WithStateHandler(handler func(ctx context.Context, deviceSn string, data *StateMessage) error) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onState = handler
	}
}

func WithStatusHandler(handler StatusHandler) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onStatus = handler
	}
}

func WithRequestHandler(handler RequestHandler) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onRequest = handler
	}
}

func WithDrcUpHandler(handler DrcUpHandler) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onDrcUp = handler
	}
}

func WithOnlineChecker(checker func(gatewaySn string) bool) ClientOption {
	return func(options *clientOptions) {
		options.handlers.onlineChecker = checker
	}
}

func WithDrcSessionEnabled(hook DrcSessionEnabledHook) ClientOption {
	return func(options *clientOptions) {
		options.drcManagerOpts = append(options.drcManagerOpts, withDrcOnSessionEnabled(hook))
	}
}

func WithDrcSessionDisabled(hook DrcSessionDisabledHook) ClientOption {
	return func(options *clientOptions) {
		options.drcManagerOpts = append(options.drcManagerOpts, withDrcOnSessionDisabled(hook))
	}
}

func WithDrcSessionExpired(hook DrcSessionExpiredHook) ClientOption {
	return func(options *clientOptions) {
		options.drcManagerOpts = append(options.drcManagerOpts, withDrcOnSessionExpired(hook))
	}
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		pendingTTL: defaultPendingTTL,
		reply:      DefaultReplyConfig(),
		drcConfig:  DefaultDrcConfig(),
	}
}

func applyOptions(opts ...ClientOption) clientOptions {
	opt := defaultClientOptions()
	for _, o := range opts {
		if o != nil {
			o(&opt)
		}
	}
	return opt
}
