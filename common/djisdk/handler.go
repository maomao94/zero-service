package djisdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zero-service/common/mqttx"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// tsFields 返回设备时间戳字段（原始 ms 和格式化字符串）。
func tsFields(ms int64) []logx.LogField {
	if ms <= 0 {
		return nil
	}
	return []logx.LogField{
		logx.Field("ts", ms),
		logx.Field("ts_fmt", carbon.CreateFromTimestampMilli(ms).ToDateTimeMilliString()),
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

func replyRouters(ttl time.Duration) []mqttx.ClientOption {
	if ttl <= 0 {
		return nil
	}
	return []mqttx.ClientOption{
		mqttx.WithReplyRouter(ServicesReplyTopicPattern(), newServicesReplyRouter(ttl)),
		mqttx.WithReplyRouter(PropertySetReplyTopicPattern(), newPropertySetReplyRouter(ttl)),
	}
}

func decodeServiceReply(kind string) mqttx.ReplyDecoderFunc[*ServiceReply] {
	return func(ctx context.Context, payload []byte, topic string, _ string) (mqttx.ReplyMessage[*ServiceReply], error) {
		var reply ServiceReply
		if err := json.Unmarshal(payload, &reply); err != nil {
			logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal %s failed: %v, topic=%s", kind, err, topic)
			return mqttx.ReplyMessage[*ServiceReply]{}, err
		}
		logx.WithContext(ctx).Infof("[dji-sdk] %s %s", kind, logFields("topic", topic, "gateway_sn", extractDeviceSnFromTopic(topic), "method", reply.Method, "tid", reply.Tid, "result", reply.Data.Result))
		return mqttx.ReplyMessage[*ServiceReply]{Tid: reply.Tid, Value: &reply}, nil
	}
}

func newServiceReplyRouter(name string, ttl time.Duration, decoder mqttx.ReplyDecoder[*ServiceReply]) *mqttx.ReplyRouter[*ServiceReply] {
	if ttl <= 0 {
		ttl = defaultPendingTTL
	}
	return mqttx.NewReplyRouter[*ServiceReply](decoder, mqttx.WithReplyRouterName(name), mqttx.WithReplyRouterTTL(ttl))
}

func newServicesReplyRouter(ttl time.Duration) *mqttx.ReplyRouter[*ServiceReply] {
	return newServiceReplyRouter("dji-services-reply", ttl, decodeServiceReply("services_reply"))
}

func newPropertySetReplyRouter(ttl time.Duration) *mqttx.ReplyRouter[*ServiceReply] {
	return newServiceReplyRouter("dji-property-set-reply", ttl, decodeServiceReply("property_set_reply"))
}

// ==================== MQTT 回调处理 ====================

// HandleEvents 处理 thing/.../events。
//
//	先走 tryDispatchEventNotify：SDK 预置的通知类 method（进度、HMS 等），设备侧多 need_reply=0；
//	未命中则打印 method 和 payload 字节数，不输出原始 payload；
//	若 need_reply=1，用 result 发 events_reply（成功为 0，失败为 1）。
func (c *Client) HandleEvents(ctx context.Context, payload []byte, topic string, _ string) error {
	var event EventMessage
	if err := json.Unmarshal(payload, &event); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal events failed: %v, topic=%s", err, topic)
		return err
	}

	eventCtx := logx.ContextWithFields(ctx,
		logx.Field("gateway_sn", event.Gateway),
		logx.Field("method", event.Method),
		logx.Field("tid", event.Tid),
		logx.Field("bid", event.Bid),
		logx.Field("need_reply", event.NeedReply),
	)
	eventCtx = logx.ContextWithFields(eventCtx, tsFields(event.Timestamp)...)
	logx.WithContext(eventCtx).Info("[dji-sdk] events")

	handled, replyResult := c.tryDispatchEventNotify(eventCtx, event.Gateway, event.Method, payload)
	if !handled {
		logx.WithContext(eventCtx).Infof("[dji-sdk] no handler for event method=%s", event.Method)
	}

	if event.NeedReply == 1 && c.reply.EnableEventReply {
		return c.eventReply(eventCtx, event.Gateway, event.Tid, event.Bid, event.Method, replyResult)
	}
	if event.NeedReply == 1 {
		logx.WithContext(eventCtx).Infof("[dji-sdk] skip event reply: gateway=%s method=%s tid=%s", event.Gateway, event.Method, event.Tid)
	}
	return nil
}

// tryDispatchEventNotify 仅处理本 SDK 已**按 method 建模**的若干**设备上行通知**（OnFlightTaskProgress/Ready、HMS、OTA、日志等）。
// 与协议一致时，这些多为**只上报、不要求 events_reply**（need_reply=0），回调侧只做落库/推送等。
// handler 返回 error 时统一打印日志；若 error 为 *PlatformError 则取其 Code 作为 events_reply 的 result，否则默认 PlatformResultHandlerError。
// 返回 handled=true 表示本 method 已由本分支处理，**不再**调用 eventMethodFallbacks。未注册的 method 走 OnEvent 兜底。
func (c *Client) tryDispatchEventNotify(ctx context.Context, gatewaySn, method string, raw []byte) (handled bool, result PlatformResult) {
	switch method {
	case MethodFlightTaskProgress:
		if c.handlers.onFlightTaskProgress != nil {
			var msg struct {
				Data struct {
					Output FlightTaskProgressEvent `json:"output"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal FlightTaskProgressEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onFlightTaskProgress(ctx, gatewaySn, &msg.Data.Output); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onFlightTaskProgress error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	case MethodFlightTaskReady:
		if c.handlers.onFlightTaskReady != nil {
			var msg struct {
				Data FlightTaskReadyEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal FlightTaskReadyEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onFlightTaskReady(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onFlightTaskReady error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	case MethodReturnHomeInfo:
		if c.handlers.onReturnHomeInfo != nil {
			var msg struct {
				Data ReturnHomeInfoEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal ReturnHomeInfoEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onReturnHomeInfo(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onReturnHomeInfo error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	case MethodCustomDataTransmissionFromPsdk:
		if c.handlers.onCustomDataTransmissionFromPsdk != nil {
			var msg struct {
				Data CustomDataFromPsdkEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal CustomDataFromPsdkEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onCustomDataTransmissionFromPsdk(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onCustomDataTransmissionFromPsdk error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	case MethodHmsEventNotify:
		if c.handlers.onHmsEventNotify != nil {
			var msg struct {
				Data HmsEventData `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal HmsEventData failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onHmsEventNotify(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onHmsEventNotify error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	case MethodRemoteLogFileUploadProgress:
		if c.handlers.onRemoteLogFileUploadProgress != nil {
			var msg struct {
				Data RemoteLogFileUploadProgressEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal RemoteLogFileUploadProgressEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onRemoteLogFileUploadProgress(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onRemoteLogFileUploadProgress error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	case MethodOtaProgress:
		if c.handlers.onOtaProgress != nil {
			var msg struct {
				Data OtaProgressEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal OtaProgressEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onOtaProgress(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onOtaProgress error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	case MethodCustomDataTransmissionFromEsdk:
		if c.handlers.onCustomDataTransmissionFromEsdk != nil {
			var msg struct {
				Data CustomDataFromEsdkEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal CustomDataFromEsdkEvent failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onCustomDataTransmissionFromEsdk(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onCustomDataTransmissionFromEsdk error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	}
	return false, PlatformResultHandlerError
}

// eventReply 向设备发送事件回复消息。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - tid: 事务 ID
//   - bid: 业务 ID
//   - method: 事件方法名
//   - result: 回复结果码，0 表示成功
//   - 返回值: 序列化或发送失败时返回错误，成功时返回 nil
func (c *Client) eventReply(ctx context.Context, gatewaySn, tid, bid, method string, result PlatformResult) error {
	reply := NewEventReply(tid, bid, method, result)
	data, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("[dji-sdk] marshal event_reply failed: %w", err)
	}
	topic := EventsReplyTopic(gatewaySn)
	return c.mqttClient.Publish(ctx, topic, data)
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
	if c.handlers.onOsd == nil {
		return nil
	}
	var msg OsdMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal osd failed: %v, topic=%s", err, topic)
		return err
	}
	deviceSn := extractDeviceSnFromTopic(topic)

	osdCtx := logx.ContextWithFields(ctx,
		logx.Field("device_sn", deviceSn),
		logx.Field("tid", msg.Tid),
		logx.Field("bid", msg.Bid),
	)
	osdCtx = logx.ContextWithFields(osdCtx, tsFields(msg.Timestamp)...)
	if err := c.handlers.onOsd(osdCtx, deviceSn, &msg); err != nil {
		logx.WithContext(osdCtx).Errorf("[dji-sdk] onOsd error: sn=%s err=%v", deviceSn, err)
	}
	return nil
}

// HandleState 处理设备的 state 主题消息回调。
// 解析设备状态数据，并通过 onState 钩子分发给上层业务。
//   - ctx: 请求上下文
//   - payload: MQTT 消息原始字节
//   - topic: 消息来源的 MQTT 主题，格式 thing/product/{device_sn}/state
//   - 返回值: 解析失败时返回错误，成功时返回 nil
func (c *Client) HandleState(ctx context.Context, payload []byte, topic string, _ string) error {
	if c.handlers.onState == nil {
		return nil
	}
	var msg StateMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal state failed: %v, topic=%s", err, topic)
		return err
	}
	deviceSn := extractDeviceSnFromTopic(topic)

	stateCtx := logx.ContextWithFields(ctx,
		logx.Field("device_sn", deviceSn),
		logx.Field("tid", msg.Tid),
		logx.Field("bid", msg.Bid),
	)
	stateCtx = logx.ContextWithFields(stateCtx, tsFields(msg.Timestamp)...)
	if err := c.handlers.onState(stateCtx, deviceSn, &msg); err != nil {
		logx.WithContext(stateCtx).Errorf("[dji-sdk] onState error: sn=%s err=%v", deviceSn, err)
	}
	return nil
}

// HandleStatus 处理 sys/product/+/status；先按 method 预分发已知强类型，再交给 OnStatus 全局回调，最后按 ReplyConfig 决定是否发布 status_reply。
func (c *Client) HandleStatus(ctx context.Context, payload []byte, topic string, _ string) error {
	var msg StatusMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal status failed: %v, topic=%s", err, topic)
		return err
	}
	gatewaySn := extractDeviceSnFromTopic(topic)

	statusCtx := logx.ContextWithFields(ctx,
		logx.Field("gateway_sn", gatewaySn),
		logx.Field("method", msg.Method),
		logx.Field("tid", msg.Tid),
		logx.Field("bid", msg.Bid),
	)
	statusCtx = logx.ContextWithFields(statusCtx, tsFields(msg.Timestamp)...)
	logx.WithContext(statusCtx).Info("[dji-sdk] status")

	handled, result := c.tryDispatchStatusNotify(statusCtx, gatewaySn, msg.Method, payload)
	if !handled && c.handlers.onStatus != nil {
		if err := c.handlers.onStatus(statusCtx, gatewaySn, &msg); err != nil {
			logx.WithContext(statusCtx).Errorf("[dji-sdk] onStatus error: sn=%s method=%s err=%v", gatewaySn, msg.Method, err)
			result = ResultFromError(err)
		}
	}
	if !c.reply.EnableStatusReply {
		logx.WithContext(statusCtx).Infof("[dji-sdk] skip status reply: sn=%s method=%s tid=%s", gatewaySn, msg.Method, msg.Tid)
		return nil
	}
	return c.statusReply(statusCtx, gatewaySn, msg.Tid, msg.Bid, result)
}

// tryDispatchStatusNotify 处理本 SDK 已按 method 建模的 status 上行（如 update_topo）。
// 返回 handled=true 表示已由强类型分支处理，不再调用 OnStatus；result 供写 status_reply。
func (c *Client) tryDispatchStatusNotify(ctx context.Context, gatewaySn, method string, raw []byte) (handled bool, result PlatformResult) {
	switch method {
	case MethodUpdateTopo:
		if c.handlers.onUpdateTopo != nil {
			var msg struct {
				Data TopoUpdateData `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal TopoUpdateData failed: %v", err)
				return true, PlatformResultHandlerError
			}
			if err := c.handlers.onUpdateTopo(ctx, gatewaySn, &msg.Data); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] onUpdateTopo error: sn=%s err=%v", gatewaySn, err)
				return true, ResultFromError(err)
			}
			return true, PlatformResultOK
		}
	}
	return false, PlatformResultHandlerError
}

// statusReply 向 [status_reply](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html) 发报文，data.result 同 EventReplyData 简形时与 events_reply 的 result 语义一致，特殊 method 以协议为准。
func (c *Client) statusReply(ctx context.Context, gatewaySn, tid, bid string, result PlatformResult) error {
	reply := StatusReply{
		Tid:       tid,
		Bid:       bid,
		Timestamp: time.Now().UnixMilli(),
		Data:      EventReplyData{Result: int(result)},
	}
	data, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("[dji-sdk] marshal status_reply failed: %w", err)
	}
	topic := StatusReplyTopic(gatewaySn)
	return c.mqttClient.Publish(ctx, topic, data)
}

// HandleRequests 处理 thing/product/+/requests；先执行业务分发，再按 ReplyConfig.EnableRequestReply 决定是否发布 requests_reply。
// 未注册 OnRequest 且启用回复时回 PlatformResultHandlerError（1），2 保留为 PlatformResultTimeout。
func (c *Client) HandleRequests(ctx context.Context, payload []byte, topic string, _ string) error {
	var msg RequestMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal requests failed: %v, topic=%s", err, topic)
		return err
	}
	gatewaySn := extractDeviceSnFromTopic(topic)

	reqCtx := logx.ContextWithFields(ctx,
		logx.Field("gateway_sn", gatewaySn),
		logx.Field("method", msg.Method),
		logx.Field("tid", msg.Tid),
		logx.Field("bid", msg.Bid),
	)
	reqCtx = logx.ContextWithFields(reqCtx, tsFields(msg.Timestamp)...)
	logx.WithContext(reqCtx).Info("[dji-sdk] requests")

	if c.handlers.onRequest == nil {
		if !c.reply.EnableRequestReply {
			logx.WithContext(reqCtx).Infof("[dji-sdk] skip request reply: sn=%s method=%s tid=%s", gatewaySn, msg.Method, msg.Tid)
			return nil
		}
		return c.requestsReply(reqCtx, gatewaySn, &msg, PlatformResultHandlerError, nil)
	}
	output, err := c.handlers.onRequest(reqCtx, gatewaySn, &msg)
	var result PlatformResult
	if err != nil {
		logx.WithContext(reqCtx).Errorf("[dji-sdk] request handler error: method=%s err=%v", msg.Method, err)
		result = ResultFromError(err)
	} else {
		result = PlatformResultOK
	}
	if !c.reply.EnableRequestReply {
		logx.WithContext(reqCtx).Infof("[dji-sdk] skip request reply: sn=%s method=%s tid=%s", gatewaySn, msg.Method, msg.Tid)
		return nil
	}
	return c.requestsReply(reqCtx, gatewaySn, &msg, result, output)
}

// requestsReply 向 thing/.../requests_reply 发报文，Envelope 复用 RequestReply（data 内 result、output 与 services_reply 常见同形，以协议为准）。
func (c *Client) requestsReply(ctx context.Context, gatewaySn string, req *RequestMessage, result PlatformResult, output any) error {
	reply := RequestReply{
		Tid:       req.Tid,
		Bid:       req.Bid,
		Timestamp: time.Now().UnixMilli(),
		Method:    req.Method,
		Data:      ServiceReplyData{Result: int(result), Output: output},
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

	drcCtx := logx.ContextWithFields(ctx,
		logx.Field("gateway_sn", gatewaySn),
		logx.Field("method", msg.Method),
		logx.Field("tid", msg.Tid),
		logx.Field("bid", msg.Bid),
	)
	drcCtx = logx.ContextWithFields(drcCtx, tsFields(msg.Timestamp)...)
	parsed, perr := DrcUnmarshalUpData(msg.Method, msg.Data)
	if perr != nil {
		logx.WithContext(drcCtx).Errorf("[dji-sdk] drc/up data parse: method=%s err=%v", msg.Method, perr)
		return perr
	}
	sum := DrcUpPayloadSummary(parsed)
	if sum == "" {
		logx.WithContext(drcCtx).Info("[dji-sdk] drc_up")
	} else {
		logx.WithContext(drcCtx).Infof("[dji-sdk] drc_up summary=%s", sum)
	}
	if c.drcManager != nil && msg.Method == MethodDrcHeartBeat {
		c.drcManager.OnDeviceHeartbeat(drcCtx, gatewaySn)
	}
	if c.handlers.onDrcUp == nil {
		return nil
	}
	return c.handlers.onDrcUp(drcCtx, gatewaySn, msg, parsed)
}

// ==================== 订阅管理 ====================

// SubscribeAll 以**云侧**身份通配订阅设备上行（*reply、events、osd、state、status、requests、**drc/up** 等），并注册处理函数，表见 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)。
// 含 **property/set_reply**（设备对云**下发** property/set 的回执，**非**云发设备）。
func (c *Client) SubscribeAll() error {
	topics := map[string]func(context.Context, []byte, string, string) error{
		EventsTopicPattern():   c.HandleEvents,
		OsdTopicPattern():      c.HandleOsd,
		StateTopicPattern():    c.HandleState,
		RequestsTopicPattern(): c.HandleRequests,
		StatusTopicPattern():   c.HandleStatus,
		DrcUpTopicPattern():    c.HandleDrcUp,
	}
	for topic, handler := range topics {
		if err := c.mqttClient.AddHandlerFunc(topic, handler); err != nil {
			return fmt.Errorf("[dji-sdk] subscribe %s failed: %w", topic, err)
		}
	}
	logx.Info("[dji-sdk] subscribed all wildcard topics")
	return nil
}
