package djisdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zero-service/common/antsx"
	"zero-service/common/mqttx"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// EventMethodFallback 在 thing/.../events 上，对「本 Client 已注册的某个 method 字符串」的兜底处理。
// 当 **没有** 命中 SDK 预置的**通知型** method 分支（见 tryDispatchEventNotify）时才会调用，见 OnEvent 与 eventMethodFallbacks。
// need_reply=1 时，result 会写入 events_reply 的 data.result；err 非 nil 且 result 为 0 时视为 PlatformResultHandlerError。
type EventMethodFallback func(ctx context.Context, event *EventMessage) (result int, err error)

// StatusHandler 处理 sys/.../status 上行。返回值只表达业务处理结果；是否发布 status_reply 由 Client 的 ReplyOptions 控制。
type StatusHandler func(ctx context.Context, gatewaySn string, data *StatusMessage) int

// DrcUpHandler 处理 thing/product/{gateway_sn}/drc/up 设备上行报文；parsed 为 DrcUnmarshalUpData 解析结果，未知 method 时为 *DrcUnknownUpData。
type DrcUpHandler func(ctx context.Context, gatewaySn string, msg *DrcUpMessage, parsed any) error

// RequestHandler 处理 thing/.../requests 上行。返回值只表达业务处理结果与输出；是否发布 requests_reply 由 Client 的 ReplyOptions 控制。
// err 非 nil 时若 result 为 0 会视为 PlatformResultHandlerError 再组包，避免启用回复时无响应。
type RequestHandler func(ctx context.Context, gatewaySn string, req *RequestMessage) (result int, output any, err error)

type mqttClient interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	AddHandlerFunc(topic string, fn func(context.Context, []byte, string, string) error) error
}

type ReplyOptions struct {
	EnableEventReply   bool
	EnableStatusReply  bool
	EnableRequestReply bool
}

func DefaultReplyOptions() ReplyOptions {
	return ReplyOptions{
		EnableEventReply:   true,
		EnableStatusReply:  true,
		EnableRequestReply: true,
	}
}

const defaultPendingTTL = 30 * time.Second

type ClientOption func(*clientOptions)

type clientOptions struct {
	pendingTTL   time.Duration
	replyOptions ReplyOptions
}

func WithPendingTTL(ttl time.Duration) ClientOption {
	return func(options *clientOptions) {
		if ttl > 0 {
			options.pendingTTL = ttl
		}
	}
}

func WithReplyOptions(replyOptions ReplyOptions) ClientOption {
	return func(options *clientOptions) {
		options.replyOptions = replyOptions
	}
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		pendingTTL:   defaultPendingTTL,
		replyOptions: DefaultReplyOptions(),
	}
}

// Client 封装**云平台侧**（上云接入端）的 MQTT 与协议能力：对设备 **Publish 下发**（如 services、property/set、drc/down），
// **通配订阅** 收设备上行（如 *reply、events、osd、requests、set_reply、drc/up 等）。见 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)。
// 含：services、events、property、osd/state、sys、requests、DRC 等，详情见包级 doc.go 与各 Topic 函数注释。
type Client struct {
	mqttClient           mqttClient
	pending              *antsx.PendingRegistry[*ServiceReply]
	replyOptions         ReplyOptions
	eventMethodFallbacks map[string]EventMethodFallback
	onlineChecker        func(gatewaySn string) bool

	onFlightTaskProgress func(ctx context.Context, gatewaySn string, data *FlightTaskProgressEvent)
	onFlightTaskReady    func(ctx context.Context, gatewaySn string, data *FlightTaskReadyEvent)
	onReturnHomeInfo     func(ctx context.Context, gatewaySn string, data *ReturnHomeInfoEvent)
	onCustomDataFromPsdk func(ctx context.Context, gatewaySn string, data *CustomDataFromPsdkEvent)
	onHmsEventNotify     func(ctx context.Context, gatewaySn string, data *HmsEventData)
	onRemoteLogProgress  func(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadProgressEvent)
	onOtaProgress        func(ctx context.Context, gatewaySn string, data *OtaProgressEvent)
	onTopoUpdate         func(ctx context.Context, gatewaySn string, data *TopoUpdateData)
	onOsd                func(ctx context.Context, deviceSn string, data *OsdMessage)
	onState              func(ctx context.Context, deviceSn string, data *StateMessage)
	onStatus             StatusHandler
	onRequest            RequestHandler
	onDrcUp              DrcUpHandler
}

func NewClient(mqttClient *mqttx.Client, opts ...ClientOption) *Client {
	return newClient(mqttClient, opts...)
}

func newClient(mqttClient mqttClient, opts ...ClientOption) *Client {
	options := defaultClientOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return &Client{
		mqttClient:           mqttClient,
		pending:              antsx.NewPendingRegistry[*ServiceReply](antsx.WithDefaultTTL(options.pendingTTL)),
		replyOptions:         options.replyOptions,
		eventMethodFallbacks: make(map[string]EventMethodFallback),
	}
}

func logFields(fields ...any) string {
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok || key == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", key, fields[i+1]))
	}
	return strings.Join(parts, " ")
}

// ==================== MQTT 回调处理 ====================

// HandleServicesReply 处理 thing/.../services_reply（与 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html) 中 services 请求-应答对）。
// 将应答按 tid 交还 SendCommand 等挂起的调用方。
func (c *Client) HandleServicesReply(ctx context.Context, payload []byte, topic string, _ string) error {
	var reply ServiceReply
	if err := json.Unmarshal(payload, &reply); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal services_reply failed: %v, topic=%s", err, topic)
		return err
	}
	logx.WithContext(ctx).Infof("[dji-sdk] services_reply %s", logFields("topic", topic, "gateway_sn", extractDeviceSnFromTopic(topic), "method", reply.Method, "tid", reply.Tid, "result", reply.Data.Result))
	c.pending.Resolve(reply.Tid, &reply)
	return nil
}

// HandlePropertySetReply 收 **设备 → 云** 的 property/set_reply（[Topic 总览与 Properties](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html) 中与云**下发** property/set 成对）。按 tid 交还 SetProperty 的挂起调用，与 HandleServicesReply、SendCommand 一致，无应用层 On* 回调（与 services_reply 相同）。
func (c *Client) HandlePropertySetReply(ctx context.Context, payload []byte, topic string, _ string) error {
	var reply ServiceReply
	if err := json.Unmarshal(payload, &reply); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal property_set_reply failed: %v, topic=%s", err, topic)
		return err
	}
	gatewaySn := extractDeviceSnFromTopic(topic)
	logx.WithContext(ctx).Infof("[dji-sdk] property_set_reply %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", reply.Method, "tid", reply.Tid, "result", reply.Data.Result))
	c.pending.Resolve(reply.Tid, &reply)
	return nil
}

// HandleEvents 处理 thing/.../events。
//
//	① 先走 tryDispatchEventNotify：SDK 预置的**通知类** method（进度、HMS 等），设备侧多 **need_reply=0**；
//	② 未命中再跑 OnEvent 的 EventMethodFallback（扩展 method 或需自定义 events_reply 的 result 时）；
//	最后若 need_reply=1，用合并后的 result 发 events_reply（① 中成功一般为 0，非 0 多为本 SDK 解包 data 失败）。
func (c *Client) HandleEvents(ctx context.Context, payload []byte, topic string, _ string) error {
	var event EventMessage
	if err := json.Unmarshal(payload, &event); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal events failed: %v, topic=%s", err, topic)
		return err
	}
	logx.WithContext(ctx).Infof("[dji-sdk] events %s", logFields("topic", topic, "gateway_sn", event.Gateway, "method", event.Method, "tid", event.Tid, "need_reply", event.NeedReply))

	handled, replyResult := c.tryDispatchEventNotify(ctx, event.Gateway, event.Method, payload)
	if !handled {
		if h, ok := c.eventMethodFallbacks[event.Method]; ok {
			var err error
			replyResult, err = h(ctx, &event)
			if err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] event fallback error: method=%s err=%v", event.Method, err)
				if replyResult == PlatformResultOK {
					replyResult = PlatformResultHandlerError
				}
			}
		}
	}

	if event.NeedReply == 1 && c.replyOptions.EnableEventReply {
		return c.replyEvent(ctx, event.Gateway, event.Tid, event.Bid, event.Method, replyResult)
	}
	if event.NeedReply == 1 {
		logx.WithContext(ctx).Infof("[dji-sdk] skip event reply: gateway=%s method=%s tid=%s", event.Gateway, event.Method, event.Tid)
	}
	return nil
}

// tryDispatchEventNotify 仅处理本 SDK 已**按 method 建模**的若干**设备上行通知**（OnFlightTaskProgress/Ready、HMS 等）：
// 与协议一致时，这些多为**只上报、不要求 events_reply**（need_reply=0），回调侧只做落库/推送等。
// 返回的 result 供极少数「仍带 need_reply=1」或解包 data 失败时写回 events_reply；成功时恒为 PlatformResultOK。
// 返回 handled=true 表示本 method 已由本分支处理，**不再**调用 eventMethodFallbacks。未注册的 method 走 OnEvent 兜底。
func (c *Client) tryDispatchEventNotify(ctx context.Context, gatewaySn, method string, raw []byte) (handled bool, result int) {
	switch method {
	case MethodFlightTaskProgress:
		if c.onFlightTaskProgress != nil {
			var msg struct {
				Data struct {
					Output FlightTaskProgressEvent `json:"output"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal FlightTaskProgressEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onFlightTaskProgress(ctx, gatewaySn, &msg.Data.Output)
			return true, PlatformResultOK
		}
	case MethodFlightTaskReady:
		if c.onFlightTaskReady != nil {
			var msg struct {
				Data FlightTaskReadyEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal FlightTaskReadyEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onFlightTaskReady(ctx, gatewaySn, &msg.Data)
			return true, PlatformResultOK
		}
	case MethodReturnHomeInfo:
		if c.onReturnHomeInfo != nil {
			var msg struct {
				Data ReturnHomeInfoEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal ReturnHomeInfoEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onReturnHomeInfo(ctx, gatewaySn, &msg.Data)
			return true, PlatformResultOK
		}
	case MethodCustomDataTransmissionFromPsdk:
		if c.onCustomDataFromPsdk != nil {
			var msg struct {
				Data CustomDataFromPsdkEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal CustomDataFromPsdkEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onCustomDataFromPsdk(ctx, gatewaySn, &msg.Data)
			return true, PlatformResultOK
		}
	case MethodHmsEventNotify:
		if c.onHmsEventNotify != nil {
			var msg struct {
				Data HmsEventData `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal HmsEventData failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onHmsEventNotify(ctx, gatewaySn, &msg.Data)
			return true, PlatformResultOK
		}
	case MethodRemoteLogFileUploadProgress:
		if c.onRemoteLogProgress != nil {
			var msg struct {
				Data RemoteLogFileUploadProgressEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal RemoteLogFileUploadProgressEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onRemoteLogProgress(ctx, gatewaySn, &msg.Data)
			return true, PlatformResultOK
		}
	case MethodOtaProgress:
		if c.onOtaProgress != nil {
			var msg struct {
				Data OtaProgressEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal OtaProgressEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onOtaProgress(ctx, gatewaySn, &msg.Data)
			return true, PlatformResultOK
		}
	case MethodUpdateTopo:
		if c.onTopoUpdate != nil {
			var msg struct {
				Data TopoUpdateData `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal TopoUpdateData failed: %v", err)
				return true, PlatformResultHandlerError
			}
			c.onTopoUpdate(ctx, gatewaySn, &msg.Data)
			return true, PlatformResultOK
		}
	}
	return false, PlatformResultOK
}

// replyEvent 向设备发送事件回复消息。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - tid: 事务 ID
//   - bid: 业务 ID
//   - method: 事件方法名
//   - result: 回复结果码，0 表示成功
//   - 返回值: 序列化或发送失败时返回错误，成功时返回 nil
func (c *Client) replyEvent(ctx context.Context, gatewaySn, tid, bid, method string, result int) error {
	reply := NewEventReply(tid, bid, method, result)
	data, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("[dji-sdk] marshal event_reply failed: %w", err)
	}
	topic := EventsReplyTopic(gatewaySn)
	return c.mqttClient.Publish(ctx, topic, data)
}

// OnEvent 按 method 注册 **扩展/兜底**（未在 tryDispatchEventNotify 中预置的 method、或同 method 不挂 On* 时）。
// 与 tryDispatchEventNotify 中各 On* 二选一：后者存在时优先生效，见 EventMethodFallback。
func (c *Client) OnEvent(method string, handler EventMethodFallback) {
	c.eventMethodFallbacks[method] = handler
}

// OnFlightTaskProgress 注册航线任务进度上报钩子。
// 方向 up：设备→云平台。对应 method: flighttask_progress。
// 机巢执行航线任务时主动定频上报进度，钩子只负责通知，业务端自行决定如何处理。
//   - handler: 回调函数，携带已解析的 FlightTaskProgressEvent 结构体
func (c *Client) OnFlightTaskProgress(handler func(ctx context.Context, gatewaySn string, data *FlightTaskProgressEvent)) {
	c.onFlightTaskProgress = handler
}

// OnFlightTaskReady 注册任务就绪通知钩子。
// 方向 up：设备→云平台。对应 method: flighttask_ready。
// 机巢中有任务满足就绪条件时主动上报，钩子只负责通知。
//   - handler: 回调函数，携带已解析的 FlightTaskReadyEvent 结构体
func (c *Client) OnFlightTaskReady(handler func(ctx context.Context, gatewaySn string, data *FlightTaskReadyEvent)) {
	c.onFlightTaskReady = handler
}

// OnReturnHomeInfo 注册返航信息上报钩子。
// 方向 up：设备→云平台。对应 method: return_home_info。
// 设备返航时主动上报规划路径信息，钩子只负责通知。
//   - handler: 回调函数，携带已解析的 ReturnHomeInfoEvent 结构体
func (c *Client) OnReturnHomeInfo(handler func(ctx context.Context, gatewaySn string, data *ReturnHomeInfoEvent)) {
	c.onReturnHomeInfo = handler
}

// OnCustomDataFromPsdk 注册 PSDK 自定义数据上报钩子。
// 方向 up：设备→云平台。对应 method: custom_data_transmission_from_psdk。
// PSDK 负载设备有自定义数据上报时通过 events topic 推送，钩子只负责通知。
//   - handler: 回调函数，携带已解析的 CustomDataFromPsdkEvent 结构体
func (c *Client) OnCustomDataFromPsdk(handler func(ctx context.Context, gatewaySn string, data *CustomDataFromPsdkEvent)) {
	c.onCustomDataFromPsdk = handler
}

// OnHmsEventNotify 注册 HMS 健康告警上报钩子。
// 方向 up：设备→云平台。对应 method: hms。
// 设备上报健康管理系统告警和状态事件时触发，钩子只负责通知。
//   - handler: 回调函数，携带已解析的 HmsEventData 结构体
func (c *Client) OnHmsEventNotify(handler func(ctx context.Context, gatewaySn string, data *HmsEventData)) {
	c.onHmsEventNotify = handler
}

func (c *Client) OnRemoteLogFileUploadProgress(handler func(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadProgressEvent)) {
	c.onRemoteLogProgress = handler
}

func (c *Client) OnOtaProgress(handler func(ctx context.Context, gatewaySn string, data *OtaProgressEvent)) {
	c.onOtaProgress = handler
}

func (c *Client) OnTopoUpdate(handler func(ctx context.Context, gatewaySn string, data *TopoUpdateData)) {
	c.onTopoUpdate = handler
}

func (c *Client) OnUpdateTopo(handler func(ctx context.Context, gatewaySn string, data *TopoUpdateData)) {
	c.OnTopoUpdate(handler)
}

// OnOsd 注册设备 OSD 遥测数据上报钩子。
// Topic: thing/product/{device_sn}/osd
// 方向 up：设备→云平台。
// 设备定期推送实时遥测数据（飞行姿态、GPS 坐标、电池电量等），钩子只负责通知。
//   - handler: 回调函数，携带设备 SN 和已解析的 OsdMessage 结构体
func (c *Client) OnOsd(handler func(ctx context.Context, deviceSn string, data *OsdMessage)) {
	c.onOsd = handler
}

// OnState 注册设备状态上报钩子。
// Topic: thing/product/{device_sn}/state
// 方向 up：设备→云平台。
// 设备上报自身状态信息（固件版本、在线状态、设备能力集等），钩子只负责通知。
//   - handler: 回调函数，携带设备 SN 和已解析的 OsdMessage 结构体
func (c *Client) OnState(handler func(ctx context.Context, deviceSn string, data *StateMessage)) {
	c.onState = handler
}

// OnStatus 注册 sys/.../status 上行业务处理器；回调会始终参与分发，status_reply 是否发送由 ReplyOptions.EnableStatusReply 控制。
func (c *Client) OnStatus(handler StatusHandler) {
	c.onStatus = handler
}

// OnRequest 注册 thing/.../requests 上行业务处理器；回调会始终参与分发，requests_reply 是否发送由 ReplyOptions.EnableRequestReply 控制。
func (c *Client) OnRequest(handler RequestHandler) {
	c.onRequest = handler
}

// OnDrcUp 注册 **thing/.../drc/up** 设备→云 处理；由 [HandleDrcUp]、[SubscribeAll] 调用。未注册时仅打 Info 日志。
func (c *Client) OnDrcUp(handler DrcUpHandler) {
	c.onDrcUp = handler
}

// SetOnlineChecker 设置设备在线状态检查函数。
// 设置后，SendCommand 在发送命令前会先调用此函数检查设备是否在线，离线则快速拒绝。
//   - checker: 在线检查函数，接收 gatewaySn，返回 true 表示在线
func (c *Client) SetOnlineChecker(checker func(gatewaySn string) bool) {
	c.onlineChecker = checker
}

// ==================== 基础命令发送 ====================

// SendCommand 向设备发送服务命令并等待应答。
// 发送命令后会阻塞等待设备的 services_reply 应答，若设备返回非零 result 则视为失败。
//   - ctx: 请求上下文，可通过 context 控制超时和取消
//   - gatewaySn: 网关设备序列号
//   - method: 命令方法名
//   - data: 命令参数数据，将被序列化为 JSON
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 发送失败、等待超时或设备返回错误时的错误信息
func (c *Client) SendCommand(ctx context.Context, gatewaySn, method string, data any) (string, error) {
	if c.onlineChecker != nil && !c.onlineChecker(gatewaySn) {
		return "", fmt.Errorf("[dji-sdk] device offline: sn=%s, command rejected", gatewaySn)
	}

	tid := uuid.New().String()
	bid := uuid.New().String()

	req := NewServiceRequest(tid, bid, method, data)
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("[dji-sdk] marshal request failed: %w", err)
	}

	topic := ServicesTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] send_command %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", method, "tid", tid))

	reply, err := antsx.RequestReply(ctx, c.pending, tid, func() error {
		return c.mqttClient.Publish(ctx, topic, payload)
	})
	if err != nil {
		return tid, fmt.Errorf("[dji-sdk] command failed: method=%s tid=%s err=%w", method, tid, err)
	}

	if reply.Data.Result != 0 {
		return tid, NewDJIError(reply.Data.Result)
	}

	return tid, nil
}

// SendCommandFireAndForget 向设备发送服务命令，不等待应答（即发即忘）。
// 仅将命令发布到 MQTT 主题，不会阻塞等待设备回复。适用于不需要确认结果的场景。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - method: 命令方法名
//   - data: 命令参数数据，将被序列化为 JSON
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 序列化或发布失败时的错误信息
func (c *Client) SendCommandFireAndForget(ctx context.Context, gatewaySn, method string, data any) (string, error) {
	tid := uuid.New().String()
	bid := uuid.New().String()

	req := NewServiceRequest(tid, bid, method, data)
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("[dji-sdk] marshal request failed: %w", err)
	}

	topic := ServicesTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] send_command_fire_and_forget %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", method, "tid", tid))

	if err := c.mqttClient.Publish(ctx, topic, payload); err != nil {
		return tid, fmt.Errorf("[dji-sdk] publish failed: method=%s tid=%s err=%w", method, tid, err)
	}
	return tid, nil
}

// ==================== 设备属性（Properties） ====================

// SetProperty 设置设备属性。
// 通过 property/set 主题向设备下发可写物模型属性，并等待 property/set_reply。
func (c *Client) SetProperty(ctx context.Context, gatewaySn string, properties PropertySetData) (string, error) {
	tid := uuid.New().String()
	bid := uuid.New().String()

	req := NewServiceRequest(tid, bid, MethodPropertySet, properties)
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("[dji-sdk] marshal property_set failed: %w", err)
	}

	topic := PropertySetTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] property_set %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", MethodPropertySet, "tid", tid))

	reply, err := antsx.RequestReply(ctx, c.pending, tid, func() error {
		return c.mqttClient.Publish(ctx, topic, payload)
	})
	if err != nil {
		return tid, fmt.Errorf("[dji-sdk] property_set failed: tid=%s err=%w", tid, err)
	}

	if reply.Data.Result != 0 {
		return tid, fmt.Errorf("[dji-sdk] property_set device error: tid=%s result=%d", tid, reply.Data.Result)
	}

	return tid, nil
}

// ==================== 设备管理（Device） ====================

// 设备拓扑 update_topo 为 status 上行，由 OnTopoUpdate/OnStatus 处理。

// ==================== 组织管理（Organization） ====================

// 组织管理 airport_* 为 requests 上行，由 OnRequest 处理。

// ==================== 直播功能（Live） ====================

// LiveStartPush 开始直播推流。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 推流参数，包含推流地址类型、URL、视频 ID 和画质等
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) LiveStartPush(ctx context.Context, gatewaySn string, data *LiveStartPushData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodLiveStartPush, data)
}

// LiveStopPush 停止直播推流。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 停止推流参数，包含视频 ID
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) LiveStopPush(ctx context.Context, gatewaySn string, data *LiveStopPushData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodLiveStopPush, data)
}

// LiveSetQuality 设置直播画质。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 画质设置参数，包含视频 ID 和目标画质
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) LiveSetQuality(ctx context.Context, gatewaySn string, data *LiveSetQualityData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodLiveSetQuality, data)
}

// LiveLensChange 切换直播推流使用的相机镜头。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 镜头切换参数，包含视频 ID 和目标镜头类型
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) LiveLensChange(ctx context.Context, gatewaySn string, data *LiveLensChangeData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodLiveLensChange, data)
}

// LiveCameraChange 切换直播相机。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 相机切换参数，包含 video_id、camera_index
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) LiveCameraChange(ctx context.Context, gatewaySn string, data *LiveCameraChangeData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodLiveCameraChange, data)
}

// ==================== 媒体功能（Media） ====================

// MediaUploadFlighttaskMediaPrioritize 优先上传指定航线任务媒体。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 航线任务媒体上传参数，包含 flight_id
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) MediaUploadFlighttaskMediaPrioritize(ctx context.Context, gatewaySn string, data *MediaUploadFlighttaskMediaPrioritizeData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodMediaUploadFlighttaskMediaPrioritize, data)
}

// MediaFastUpload 快速上传指定媒体文件。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 媒体快速上传参数，包含 file_id
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) MediaFastUpload(ctx context.Context, gatewaySn string, data *MediaFastUploadData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodMediaFastUpload, data)
}

// MediaHighestPriorityUploadFlighttask 最高优先级上传指定航线任务媒体。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 航线任务媒体上传参数，包含 flight_id
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) MediaHighestPriorityUploadFlighttask(ctx context.Context, gatewaySn string, data *MediaHighestPriorityUploadFlighttaskData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodMediaHighestPriorityUploadFlighttask, data)
}

// ==================== 航线功能（Wayline） ====================

// FlightTaskPrepare 下发航线任务准备指令。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 航线任务准备参数，序列化为 services 请求的 data 字段
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlightTaskPrepare(ctx context.Context, gatewaySn string, data *FlightTaskPrepareData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskPrepare, data)
}

// FlightTaskExecute 下发航线任务执行指令。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - flightID: 航线任务 ID，须与 FlightTaskPrepare 中的 FlightID 一致
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlightTaskExecute(ctx context.Context, gatewaySn, flightID string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskExecute, &FlightTaskExecuteData{FlightID: flightID})
}

// CancelFlightTask 取消指定的飞行任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - flightIDs: 要取消的飞行任务 ID 列表
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CancelFlightTask(ctx context.Context, gatewaySn string, flightIDs []string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskCancel, &FlightTaskCancelData{FlightIDs: flightIDs})
}

// PauseFlightTask 暂停当前正在执行的飞行任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) PauseFlightTask(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskPause, struct{}{})
}

// ResumeFlightTask 恢复已暂停的飞行任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) ResumeFlightTask(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskResume, struct{}{})
}

// StopFlightTask 强制停止当前航线任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) StopFlightTask(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskStop, struct{}{})
}

// ReturnHome 控制无人机一键返航。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) ReturnHome(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodReturnHome, &ReturnHomeData{})
}

// ReturnHomeCancelAutoReturn 取消自动返航。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) ReturnHomeCancelAutoReturn(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodReturnHomeCancelAutoReturn, &ReturnHomeCancelData{})
}

// ReturnSpecificHome 控制飞行器返航至指定备降点。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 指定返航点参数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) ReturnSpecificHome(ctx context.Context, gatewaySn string, data *ReturnSpecificHomeData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodReturnSpecificHome, data)
}

// ==================== HMS 管理（HMS） ====================

// HMS 告警上行事件由 HandleEvents 与 OnEvent 处理。

// ==================== 远程调试（Cmd） ====================

// DebugModeOpen 开启机巢调试模式。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DebugModeOpen(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDebugModeOpen, &DebugModeData{})
}

// DebugModeClose 关闭机巢调试模式。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DebugModeClose(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDebugModeClose, &DebugModeData{})
}

// CoverOpen 打开机巢舱盖。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CoverOpen(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCoverOpen, &CoverData{})
}

// CoverClose 关闭机巢舱盖。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CoverClose(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCoverClose, &CoverData{})
}

// CoverForceClose 强制关闭机巢舱盖。
// 在常规关闭无法完成时使用，会忽略部分安全检查。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CoverForceClose(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCoverForceClose, &CoverData{})
}

// DroneOpen 开启机巢中的无人机电源。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DroneOpen(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDroneOpen, &DroneData{})
}

// DroneClose 关闭机巢中的无人机电源。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DroneClose(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDroneClose, &DroneData{})
}

// DeviceReboot 重启机巢设备。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DeviceReboot(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDeviceReboot, &DeviceRebootData{})
}

// ChargeOpen 开启机巢充电功能。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) ChargeOpen(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodChargeOpen, &ChargeData{})
}

// ChargeClose 关闭机巢充电功能。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) ChargeClose(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodChargeClose, &ChargeData{})
}

// DroneFormat 格式化无人机存储。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DroneFormat(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDroneFormat, &FormatData{})
}

// DeviceFormat 格式化机巢设备存储。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DeviceFormat(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDeviceFormat, &FormatData{})
}

// SupplementLightOpen 开启机巢补光灯。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) SupplementLightOpen(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodSupplementLightOpen, &SupplementLightData{})
}

// SupplementLightClose 关闭机巢补光灯。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) SupplementLightClose(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodSupplementLightClose, &SupplementLightData{})
}

// BatteryStoreModeSwitch 切换电池保养模式。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - enable: 开关状态，1 为开启，0 为关闭
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) BatteryStoreModeSwitch(ctx context.Context, gatewaySn string, enable int) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodBatteryStoreModeSwitch, &BatteryStoreModeSwitchData{Enable: enable})
}

// AlarmStateSwitch 切换机巢声光报警状态。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - action: 动作标识，具体含义参考 DJI Cloud API 文档
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) AlarmStateSwitch(ctx context.Context, gatewaySn string, action int) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodAlarmStateSwitch, &AlarmStateSwitchData{Action: action})
}

// AirConditionerModeSwitch 切换机巢空调工作模式。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - action: 空调模式标识，具体含义参考 DJI Cloud API 文档
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) AirConditionerModeSwitch(ctx context.Context, gatewaySn string, action int) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodAirConditionerModeSwitch, &AirConditionerModeSwitchData{Action: action})
}

// BatteryMaintenanceSwitch 切换电池保养功能开关。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - enable: 开关状态，1 为开启，0 为关闭
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) BatteryMaintenanceSwitch(ctx context.Context, gatewaySn string, enable int) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodBatteryMaintenanceSwitch, &BatteryMaintenanceSwitchData{Enable: enable})
}

// ==================== 固件升级（Firmware） ====================

// OtaCreate 创建 OTA 固件升级任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 升级参数，包含待升级设备列表
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) OtaCreate(ctx context.Context, gatewaySn string, data *OtaCreateData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodOtaCreate, data)
}

// ==================== 远程日志（Log） ====================

// RemoteLogFileList 查询可上传的远程日志文件列表。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 远程日志列表查询参数，包含目标设备和模块
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) RemoteLogFileList(ctx context.Context, gatewaySn string, data *RemoteLogFileListData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodRemoteLogFileList, data)
}

// RemoteLogFileUploadStart 开始上传远程日志文件。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 远程日志上传参数，包含待上传文件列表
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) RemoteLogFileUploadStart(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadStartData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodRemoteLogFileUploadStart, data)
}

// RemoteLogFileUploadUpdate 更新远程日志文件上传任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 远程日志上传更新参数，包含文件列表
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) RemoteLogFileUploadUpdate(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadUpdateData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodRemoteLogFileUploadUpdate, data)
}

// RemoteLogFileUploadCancel 取消远程日志文件上传任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 远程日志上传取消参数，包含文件列表
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) RemoteLogFileUploadCancel(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadCancelData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodRemoteLogFileUploadCancel, data)
}

// ==================== 配置更新（Config） ====================

// ConfigUpdate 下发设备配置更新。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 配置更新参数，包含设备配置键值
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) ConfigUpdate(ctx context.Context, gatewaySn string, data *ConfigUpdateData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodConfigUpdate, data)
}

// ==================== 指令飞行（DRC） ====================

// FlightAuthorityGrab 获取飞行控制权。
// 在指令飞行前需要先获取飞行控制权。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlightAuthorityGrab(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightAuthorityGrab, &FlightAuthorityGrabData{})
}

// PayloadAuthorityGrab 获取负载控制权。
// 在控制相机、云台等负载设备前需要先获取负载控制权。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) PayloadAuthorityGrab(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodPayloadAuthorityGrab, &PayloadAuthorityGrabData{})
}

// DrcModeEnter 进入指令飞行（DRC）模式；经 **thing/.../services** 发 **drc_mode_enter** method，**services_reply** 为应答，非 drc/* topic。进入后在 **drc/down** 可发杆量（见 SendDrcStickControl）。见 [DRC 文档](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html)。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: DRC 模式进入参数，包含 MQTT Broker 连接信息等
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DrcModeEnter(ctx context.Context, gatewaySn string, data *DrcModeEnterData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDrcModeEnter, data)
}

// DrcModeExit 退出指令飞行（DRC）模式；**services** 上的 method **drc_mode_exit**，**services_reply** 应答。见 [DRC 文档](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html)。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DrcModeExit(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDrcModeExit, &DrcModeExitData{})
}

// FlyToPoint 飞往指定航点。
// 控制无人机从当前位置飞往一组指定的航点。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 飞行参数，包含最大速度、航点列表等
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlyToPoint(ctx context.Context, gatewaySn string, data *FlyToPointData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlyToPoint, data)
}

// FlyToPointStop 停止当前的飞往航点任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlyToPointStop(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlyToPointStop, struct{}{})
}

// TakeoffToPoint 起飞到指定坐标点。
// 无人机从当前位置起飞并飞往指定的目标坐标点。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 起飞参数，包含目标经纬度、高度、安全起飞高度等
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) TakeoffToPoint(ctx context.Context, gatewaySn string, data *TakeoffToPointData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodTakeoffToPoint, data)
}

// SendDrcStickControl 经 drc/down 即发即忘地下发 stick_control 杆量。
// seq 位于顶层，data 包含 roll、pitch、throttle、yaw、gimbal_pitch，不等待 services_reply。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 杆量数据（DrcStickControlData）
//   - 返回值: 序列化或发布失败时的错误信息
func (c *Client) SendDrcStickControl(ctx context.Context, gatewaySn string, seq int, data *DrcStickControlData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodStickControl, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

func (c *Client) publishDrcDown(ctx context.Context, gatewaySn string, msg *DrcDownMessage) (string, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return msg.Tid, fmt.Errorf("[dji-sdk] marshal drc/down failed: method=%s tid=%s err=%w", msg.Method, msg.Tid, err)
	}
	topic := DrcDownTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] drc_down %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", msg.Method, "tid", msg.Tid, "seq", msg.Seq))
	if err := c.mqttClient.Publish(ctx, topic, payload); err != nil {
		return msg.Tid, fmt.Errorf("[dji-sdk] publish drc/down failed: method=%s tid=%s err=%w", msg.Method, msg.Tid, err)
	}
	return msg.Tid, nil
}

// SendDrcHeartBeat 经 drc/down 即发即忘地下发 heart_beat 心跳，seq 与 data 同级，data.timestamp 表示心跳时间戳。
func (c *Client) SendDrcHeartBeat(ctx context.Context, gatewaySn string, seq int, dataTimestampMillis int64) (string, error) {
	seqp := seq
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcHeartBeat, DrcHeartBeatDownData{Timestamp: dataTimestampMillis}, &seqp)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DroneEmergencyStop 经 drc/down 即发即忘地下发 drone_emergency_stop。
func (c *Client) DroneEmergencyStop(ctx context.Context, gatewaySn string) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDroneEmergencyStop, DroneEmergencyStopData{}, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// ==================== 指令飞行（DRC）相机/云台控制 ====================

// CameraModeSwitch 切换相机拍摄模式（拍照/录像等）。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 相机模式切换参数，包含负载索引和目标模式
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraModeSwitch(ctx context.Context, gatewaySn string, data *CameraModeSwitchData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraModeSwitch, data)
}

// CameraPhotoTake 控制相机拍照。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 拍照参数，包含负载索引
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraPhotoTake(ctx context.Context, gatewaySn string, data *CameraPhotoTakeData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraPhotoTake, data)
}

// CameraPhotoStop 停止相机连续拍照。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - payloadIndex: 负载设备索引
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraPhotoStop(ctx context.Context, gatewaySn, payloadIndex string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraPhotoStop, &CameraPhotoTakeData{PayloadIndex: payloadIndex})
}

// CameraRecordingStart 开始录像。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 录像启动参数，包含负载索引
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraRecordingStart(ctx context.Context, gatewaySn string, data *CameraRecordingStartData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraRecordingStart, data)
}

// CameraRecordingStop 停止录像。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 录像停止参数，包含负载索引
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraRecordingStop(ctx context.Context, gatewaySn string, data *CameraRecordingStopData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraRecordingStop, data)
}

// CameraFocalLengthSet 设置相机焦距（变焦倍数）。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 焦距设置参数，包含负载索引、相机类型和变焦倍数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraFocalLengthSet(ctx context.Context, gatewaySn string, data *CameraFocalLengthSetData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraFocalLengthSet, data)
}

// GimbalReset 重置云台角度。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 云台重置参数，包含负载索引和重置模式
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) GimbalReset(ctx context.Context, gatewaySn string, data *GimbalResetData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodGimbalReset, data)
}

// CameraAim 控制相机对准指定屏幕坐标点。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 瞄准参数，包含负载索引、相机类型和目标坐标
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraAim(ctx context.Context, gatewaySn string, data *CameraAimData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraAim, data)
}

// CameraPointFocusAction 控制相机在指定屏幕坐标执行对焦。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 对焦参数，包含负载索引、相机类型和对焦坐标
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraPointFocusAction(ctx context.Context, gatewaySn string, data *CameraPointFocusActionData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraPointFocusAction, data)
}

// CameraScreenSplit 控制相机画面分屏显示。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 分屏参数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraScreenSplit(ctx context.Context, gatewaySn string, data *CameraScreenSplitData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraScreenSplit, data)
}

// CameraPhotoStorageSet 设置拍照存储位置。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 存储设置参数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraPhotoStorageSet(ctx context.Context, gatewaySn string, data *CameraStorageSetData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraPhotoStorageSet, data)
}

// CameraVideoStorageSet 设置录像存储位置。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 存储设置参数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraVideoStorageSet(ctx context.Context, gatewaySn string, data *CameraStorageSetData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraVideoStorageSet, data)
}

// CameraLookAt 控制相机持续朝向指定地理坐标。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 朝向参数，包含负载索引和目标坐标
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraLookAt(ctx context.Context, gatewaySn string, data *CameraLookAtData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraLookAt, data)
}

// CameraScreenDrag 通过屏幕拖拽方式控制云台转动。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 拖动参数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraScreenDrag(ctx context.Context, gatewaySn string, data *CameraScreenDragData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraScreenDrag, data)
}

// CameraIrMeteringPoint 设置红外相机指定点测温。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 测温点参数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraIrMeteringPoint(ctx context.Context, gatewaySn string, data *CameraIrMeteringPointData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraIrMeteringPoint, data)
}

// CameraIrMeteringArea 设置红外相机指定区域测温。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 测温区域参数
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraIrMeteringArea(ctx context.Context, gatewaySn string, data *CameraIrMeteringAreaData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraIrMeteringArea, data)
}

// ==================== 自定义飞行区（Custom Fly Region） ====================

// FlightAreasUpdate 触发自定义飞行区文件更新。
func (c *Client) FlightAreasUpdate(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightAreasUpdate, &FlightAreasUpdateData{})
}

// ==================== PSDK 功能（PSDK） ====================

// PsdkUIResourceUpload 上传 PSDK UI 资源。
func (c *Client) PsdkUIResourceUpload(ctx context.Context, gatewaySn string, data *PsdkUIResourceUploadData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodPsdkFloatUp, data)
}

// ==================== PSDK 互联互通（PSDK Transmit） ====================

// SendCustomDataToPsdk 自定义数据透传至 PSDK 负载设备。
// 下行对应 custom_data_transmission_to_psdk；上行 custom_data_transmission_from_psdk 由 OnCustomDataFromPsdk 处理。
func (c *Client) SendCustomDataToPsdk(ctx context.Context, gatewaySn, value string) (string, error) {
	data := &CustomDataTransmissionData{
		Value: value,
	}
	return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToPsdk, data)
}

// ==================== ESDK 互联互通（ESDK Transmit） ====================

// SendCustomDataToEsdk 自定义数据透传至 ESDK 设备。
func (c *Client) SendCustomDataToEsdk(ctx context.Context, gatewaySn, value string) (string, error) {
	data := &CustomDataToEsdkData{Value: value}
	return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToEsdk, data)
}

// ==================== 远程解禁（Flysafe） ====================

// UnlockLicenseSwitch 启用或禁用设备的单个解禁证书。
func (c *Client) UnlockLicenseSwitch(ctx context.Context, gatewaySn string, data *UnlockLicenseSwitchData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodUnlockLicenseSwitch, data)
}

// UnlockLicenseUpdate 更新设备的解禁证书。
func (c *Client) UnlockLicenseUpdate(ctx context.Context, gatewaySn string, data *UnlockLicenseUpdateData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodUnlockLicenseUpdate, data)
}

// UnlockLicenseList 获取设备的解禁证书列表。
func (c *Client) UnlockLicenseList(ctx context.Context, gatewaySn string, data *UnlockLicenseListData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodUnlockLicenseList, data)
}

// ==================== AirSense ====================

// AirSense 当前仅包含设备上行告警事件，可通过 OnEvent 处理。

// ==================== 远程控制（Remote Control） ====================

// DrcForceLanding 经 drc/down 下发强制降落。
func (c *Client) DrcForceLanding(ctx context.Context, gatewaySn string) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcForceLanding, struct{}{}, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcEmergencyLanding 经 drc/down 下发紧急降落。
func (c *Client) DrcEmergencyLanding(ctx context.Context, gatewaySn string) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcEmergencyLanding, struct{}{}, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcLinkageZoomSet 经 drc/down 设置红外联动变焦状态。
func (c *Client) DrcLinkageZoomSet(ctx context.Context, gatewaySn string, data *DrcLinkageZoomSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcLinkageZoomSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcVideoResolutionSet 经 drc/down 设置视频分辨率。
func (c *Client) DrcVideoResolutionSet(ctx context.Context, gatewaySn string, data *DrcVideoResolutionSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcVideoResolutionSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcIntervalPhotoSet 经 drc/down 设置定时拍参数。
func (c *Client) DrcIntervalPhotoSet(ctx context.Context, gatewaySn string, data *DrcIntervalPhotoSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcIntervalPhotoSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcInitialStateSubscribe 经 drc/down 订阅 DRC 初始状态。
func (c *Client) DrcInitialStateSubscribe(ctx context.Context, gatewaySn string) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcInitialStateSubscribe, struct{}{}, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcNightLightsStateSet 经 drc/down 设置夜航灯状态。
func (c *Client) DrcNightLightsStateSet(ctx context.Context, gatewaySn string, data *DrcNightLightsStateSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcNightLightsStateSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcStealthStateSet 经 drc/down 设置隐蔽模式状态。
func (c *Client) DrcStealthStateSet(ctx context.Context, gatewaySn string, data *DrcStealthStateSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcStealthStateSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraApertureValueSet 经 drc/down 设置相机光圈。
func (c *Client) DrcCameraApertureValueSet(ctx context.Context, gatewaySn string, data *DrcCameraApertureValueSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraApertureValueSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraShutterSet 经 drc/down 设置相机快门。
func (c *Client) DrcCameraShutterSet(ctx context.Context, gatewaySn string, data *DrcCameraShutterSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraShutterSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraIsoSet 经 drc/down 设置相机 ISO。
func (c *Client) DrcCameraIsoSet(ctx context.Context, gatewaySn string, data *DrcCameraIsoSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraIsoSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraMechanicalShutterSet 经 drc/down 设置机械快门。
func (c *Client) DrcCameraMechanicalShutterSet(ctx context.Context, gatewaySn string, data *DrcCameraMechanicalShutterSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraMechanicalShutterSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraDewarpingSet 经 drc/down 设置镜头去畸变。
func (c *Client) DrcCameraDewarpingSet(ctx context.Context, gatewaySn string, data *DrcCameraDewarpingSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraDewarpingSet, data, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// ==================== OSD / State 回调处理 ====================

// extractDeviceSnFromTopic 从 MQTT topic 中提取第三段设备/网关 SN（thing 与 sys 同形：*/product/{sn}/...）。
// 适用如 thing/product/{sn}/osd、thing/product/{sn}/requests、sys/product/{sn}/status。
func extractDeviceSnFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// HandleOsd 处理设备的 osd 主题消息回调。
// 解析设备 OSD 遥测数据，并通过 onOsd 钩子分发给上层业务。
//   - ctx: 请求上下文
//   - payload: MQTT 消息原始字节
//   - topic: 消息来源的 MQTT 主题，格式 thing/product/{device_sn}/osd
//   - 返回值: 解析失败时返回错误，成功时返回 nil
func (c *Client) HandleOsd(ctx context.Context, payload []byte, topic string, _ string) error {
	if c.onOsd == nil {
		return nil
	}
	var msg OsdMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal osd failed: %v, topic=%s", err, topic)
		return err
	}
	deviceSn := extractDeviceSnFromTopic(topic)
	c.onOsd(ctx, deviceSn, &msg)
	return nil
}

// HandleState 处理设备的 state 主题消息回调。
// 解析设备状态数据，并通过 onState 钩子分发给上层业务。
//   - ctx: 请求上下文
//   - payload: MQTT 消息原始字节
//   - topic: 消息来源的 MQTT 主题，格式 thing/product/{device_sn}/state
//   - 返回值: 解析失败时返回错误，成功时返回 nil
func (c *Client) HandleState(ctx context.Context, payload []byte, topic string, _ string) error {
	if c.onState == nil {
		return nil
	}
	var msg StateMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal state failed: %v, topic=%s", err, topic)
		return err
	}
	deviceSn := extractDeviceSnFromTopic(topic)
	c.onState(ctx, deviceSn, &msg)
	return nil
}

// HandleStatus 处理 sys/product/+/status；先执行业务分发，再按 ReplyOptions.EnableStatusReply 决定是否发布 status_reply。
func (c *Client) HandleStatus(ctx context.Context, payload []byte, topic string, _ string) error {
	var msg StatusMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal status failed: %v, topic=%s", err, topic)
		return err
	}
	gatewaySn := extractDeviceSnFromTopic(topic)
	logx.WithContext(ctx).Infof("[dji-sdk] status %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", msg.Method, "tid", msg.Tid))

	result := PlatformResultOK
	if c.onStatus != nil {
		result = c.onStatus(ctx, gatewaySn, &msg)
	}
	if !c.replyOptions.EnableStatusReply {
		logx.WithContext(ctx).Infof("[dji-sdk] skip status reply: sn=%s method=%s tid=%s", gatewaySn, msg.Method, msg.Tid)
		return nil
	}
	return c.replyStatus(ctx, gatewaySn, msg.Tid, msg.Bid, result)
}

// replyStatus 向 [status_reply](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html) 发报文，data.result 同 EventReplyData 简形时与 events_reply 的 result 语义一致，特殊 method 以协议为准。
func (c *Client) replyStatus(ctx context.Context, gatewaySn, tid, bid string, result int) error {
	reply := StatusReply{
		Tid:       tid,
		Bid:       bid,
		Timestamp: time.Now().UnixMilli(),
		Data:      EventReplyData{Result: result},
	}
	data, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("[dji-sdk] marshal status_reply failed: %w", err)
	}
	topic := StatusReplyTopic(gatewaySn)
	return c.mqttClient.Publish(ctx, topic, data)
}

// HandleRequests 处理 thing/product/+/requests；先执行业务分发，再按 ReplyOptions.EnableRequestReply 决定是否发布 requests_reply。
// 未注册 OnRequest 且启用回复时回 PlatformResultHandlerError（1），2 保留为 PlatformResultTimeout。
func (c *Client) HandleRequests(ctx context.Context, payload []byte, topic string, _ string) error {
	var msg RequestMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal requests failed: %v, topic=%s", err, topic)
		return err
	}
	gatewaySn := extractDeviceSnFromTopic(topic)
	logx.WithContext(ctx).Infof("[dji-sdk] requests %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", msg.Method, "tid", msg.Tid))

	if c.onRequest == nil {
		if !c.replyOptions.EnableRequestReply {
			logx.WithContext(ctx).Infof("[dji-sdk] skip request reply: sn=%s method=%s tid=%s", gatewaySn, msg.Method, msg.Tid)
			return nil
		}
		return c.replyToRequest(ctx, gatewaySn, &msg, PlatformResultHandlerError, nil)
	}
	result, output, err := c.onRequest(ctx, gatewaySn, &msg)
	if err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] request handler error: method=%s err=%v", msg.Method, err)
		if result == PlatformResultOK {
			result = PlatformResultHandlerError
		}
	}
	if !c.replyOptions.EnableRequestReply {
		logx.WithContext(ctx).Infof("[dji-sdk] skip request reply: sn=%s method=%s tid=%s", gatewaySn, msg.Method, msg.Tid)
		return nil
	}
	return c.replyToRequest(ctx, gatewaySn, &msg, result, output)
}

// replyToRequest 向 thing/.../requests_reply 发报文，Envelope 复用 RequestReply（data 内 result、output 与 services_reply 常见同形，以协议为准）。
func (c *Client) replyToRequest(ctx context.Context, gatewaySn string, req *RequestMessage, result int, output any) error {
	reply := RequestReply{
		Tid:       req.Tid,
		Bid:       req.Bid,
		Timestamp: time.Now().UnixMilli(),
		Method:    req.Method,
		Data:      ServiceReplyData{Result: result, Output: output},
	}
	data, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("[dji-sdk] marshal requests_reply failed: %w", err)
	}
	return c.mqttClient.Publish(ctx, RequestsReplyTopic(gatewaySn), data)
}

// HandleDrcUp 处理 thing/product/{gateway_sn}/drc/up 设备上行。
// 已知 method 解析为强类型，未知 method 保留 raw data 后继续调用 OnDrcUp。
func (c *Client) HandleDrcUp(ctx context.Context, payload []byte, topic string, _ string) error {
	msg, err := DrcUpMessageFromJSON(payload)
	if err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal drc/up failed: %v, topic=%s", err, topic)
		return err
	}
	gatewaySn := extractDeviceSnFromTopic(topic)
	parsed, perr := DrcUnmarshalUpData(msg.Method, msg.Data)
	if perr != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] drc/up data parse: method=%s err=%v", msg.Method, perr)
		return perr
	}
	sum := DrcUpPayloadSummary(msg.Method, parsed)
	if sum == "" {
		logx.WithContext(ctx).Infof("[dji-sdk] drc_up %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", msg.Method, "tid", msg.Tid, "ts", msg.Timestamp))
	} else {
		logx.WithContext(ctx).Infof("[dji-sdk] drc_up %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", msg.Method, "tid", msg.Tid, "ts", msg.Timestamp, "summary", sum))
	}
	if c.onDrcUp == nil {
		return nil
	}
	return c.onDrcUp(ctx, gatewaySn, msg, parsed)
}

// ==================== 订阅管理 ====================

// SubscribeAll 以**云侧**身份通配订阅设备上行（*reply、events、osd、state、status、requests、**drc/up** 等），并注册处理函数，表见 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)。
// 含 **property/set_reply**（设备对云**下发** property/set 的回执，**非**云发设备）。
func (c *Client) SubscribeAll() error {
	topics := map[string]func(context.Context, []byte, string, string) error{
		ServicesReplyTopicPattern():    c.HandleServicesReply,
		EventsTopicPattern():           c.HandleEvents,
		PropertySetReplyTopicPattern(): c.HandlePropertySetReply,
		OsdTopicPattern():              c.HandleOsd,
		StateTopicPattern():            c.HandleState,
		RequestsTopicPattern():         c.HandleRequests,
		StatusTopicPattern():           c.HandleStatus,
		DrcUpTopicPattern():            c.HandleDrcUp,
	}
	for topic, handler := range topics {
		if err := c.mqttClient.AddHandlerFunc(topic, handler); err != nil {
			return fmt.Errorf("[dji-sdk] subscribe %s failed: %w", topic, err)
		}
	}
	logx.Info("[dji-sdk] subscribed all wildcard topics")
	return nil
}

// SubscribeServicesReply 订阅 services_reply 通配主题。
// 注册 HandleServicesReply 作为消息回调处理函数。
//   - 返回值: 订阅失败时返回错误，成功时返回 nil
func (c *Client) SubscribeServicesReply() error {
	return c.mqttClient.AddHandlerFunc(ServicesReplyTopicPattern(), c.HandleServicesReply)
}

// SubscribeEvents 订阅 events 通配主题。
// 注册 HandleEvents 作为消息回调处理函数。
//   - 返回值: 订阅失败时返回错误，成功时返回 nil
func (c *Client) SubscribeEvents() error {
	return c.mqttClient.AddHandlerFunc(EventsTopicPattern(), c.HandleEvents)
}

// SubscribePropertySetReply 订阅 [property/set_reply](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/properties.html) 通配主题，注册 HandlePropertySetReply。
func (c *Client) SubscribePropertySetReply() error {
	return c.mqttClient.AddHandlerFunc(PropertySetReplyTopicPattern(), c.HandlePropertySetReply)
}

// SubscribeRequests 订阅 thing/.../requests 通配主题。见 [Requests | organization](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/organization.html)。
func (c *Client) SubscribeRequests() error {
	return c.mqttClient.AddHandlerFunc(RequestsTopicPattern(), c.HandleRequests)
}

// SubscribeDrcUp 订阅 [thing/.../drc/up](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html) 通配主题，注册 [HandleDrcUp]。
func (c *Client) SubscribeDrcUp() error {
	return c.mqttClient.AddHandlerFunc(DrcUpTopicPattern(), c.HandleDrcUp)
}

// ==================== 生命周期 ====================

// Close 关闭客户端，释放资源。
// 关闭待处理请求注册表，清理所有未完成的等待操作。
func (c *Client) Close() {
	c.pending.Close()
}
