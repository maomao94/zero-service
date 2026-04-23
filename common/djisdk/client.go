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

// EventHandler 事件处理函数类型。
//   - ctx: 请求上下文
//   - event: 从设备接收到的事件消息
//   - 返回值: 处理过程中的错误，nil 表示成功
type EventHandler func(ctx context.Context, event *EventMessage) error

// Client DJI Cloud API 客户端，封装了与 DJI 设备通过 MQTT 协议进行交互的全部能力。
// 支持服务命令下发、属性设置、事件处理、航线管理、PSDK 通信、远程调试、指令飞行、相机控制、直播管理等功能。
type Client struct {
	mqttClient    *mqttx.Client
	pending       *antsx.PendingRegistry[*ServiceReply]
	ackTimeout    time.Duration
	eventHandlers map[string]EventHandler

	onFlightTaskProgress func(ctx context.Context, gatewaySn string, data *FlightTaskProgressEvent)
	onFlightTaskReady    func(ctx context.Context, gatewaySn string, data *FlightTaskReadyEvent)
	onReturnHomeInfo     func(ctx context.Context, gatewaySn string, data *ReturnHomeInfoEvent)
	onCustomDataFromPsdk func(ctx context.Context, gatewaySn string, data *CustomDataFromPsdkEvent)
	onHmsEventNotify     func(ctx context.Context, gatewaySn string, data *HmsEventData)
	onOsd                func(ctx context.Context, deviceSn string, data *OsdMessage)
	onState              func(ctx context.Context, deviceSn string, data *OsdMessage)
}

// NewClient 创建一个新的 DJI Cloud API 客户端实例。
//   - mqttClient: MQTT 客户端实例，用于与设备进行 MQTT 通信
//   - ackTimeout: 命令应答超时时间
//   - pendingTTL: 待处理请求的过期时间
//   - 返回值: 初始化完成的 Client 指针
func NewClient(mqttClient *mqttx.Client, ackTimeout time.Duration, pendingTTL time.Duration) *Client {
	return &Client{
		mqttClient:    mqttClient,
		pending:       antsx.NewPendingRegistry[*ServiceReply](antsx.WithDefaultTTL(pendingTTL)),
		ackTimeout:    ackTimeout,
		eventHandlers: make(map[string]EventHandler),
	}
}

// ==================== MQTT 回调处理 ====================

// HandleServicesReply 处理设备的 services_reply 主题消息回调。
// 解析设备返回的服务应答消息，并通过 tid 匹配将应答分发给对应的等待方。
//   - ctx: 请求上下文
//   - payload: MQTT 消息原始字节
//   - topic: 消息来源的 MQTT 主题
//   - 返回值: 解析失败时返回错误，成功时返回 nil
func (c *Client) HandleServicesReply(ctx context.Context, payload []byte, topic string, _ string) error {
	var reply ServiceReply
	if err := json.Unmarshal(payload, &reply); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal services_reply failed: %v, topic=%s", err, topic)
		return err
	}
	logx.WithContext(ctx).Infof("[dji-sdk] received reply: tid=%s method=%s result=%d", reply.Tid, reply.Method, reply.Data.Result)
	c.pending.Resolve(reply.Tid, &reply)
	return nil
}

// HandlePropertySetReply 处理设备的 property_set_reply 主题消息回调。
// 解析设备返回的属性设置应答消息，并通过 tid 匹配将应答分发给对应的等待方。
//   - ctx: 请求上下文
//   - payload: MQTT 消息原始字节
//   - topic: 消息来源的 MQTT 主题
//   - 返回值: 解析失败时返回错误，成功时返回 nil
func (c *Client) HandlePropertySetReply(ctx context.Context, payload []byte, topic string, _ string) error {
	var reply ServiceReply
	if err := json.Unmarshal(payload, &reply); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal property_set_reply failed: %v, topic=%s", err, topic)
		return err
	}
	logx.WithContext(ctx).Infof("[dji-sdk] property set reply: tid=%s result=%d", reply.Tid, reply.Data.Result)
	c.pending.Resolve(reply.Tid, &reply)
	return nil
}

// HandleEvents 处理设备的 events 主题消息回调。
// 解析事件消息，优先匹配类型化钩子进行结构化分发，无匹配则走通用 eventHandlers 兜底。
// 若事件需要回复（need_reply=1），则自动发送事件回复。
//   - ctx: 请求上下文
//   - payload: MQTT 消息原始字节
//   - topic: 消息来源的 MQTT 主题
//   - 返回值: 解析失败或回复发送失败时返回错误，成功时返回 nil
func (c *Client) HandleEvents(ctx context.Context, payload []byte, topic string, _ string) error {
	var event EventMessage
	if err := json.Unmarshal(payload, &event); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal events failed: %v, topic=%s", err, topic)
		return err
	}
	logx.WithContext(ctx).Infof("[dji-sdk] received event: tid=%s method=%s need_reply=%d gateway=%s", event.Tid, event.Method, event.NeedReply, event.Gateway)

	handled := c.dispatchTypedEvent(ctx, event.Gateway, event.Method, payload)

	if !handled {
		if handler, ok := c.eventHandlers[event.Method]; ok {
			if err := handler(ctx, &event); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] event handler error: method=%s err=%v", event.Method, err)
			}
		}
	}

	if event.NeedReply == 1 {
		return c.replyEvent(ctx, event.Gateway, event.Tid, event.Bid, event.Method, 0)
	}
	return nil
}

// dispatchTypedEvent 按 method 匹配类型化钩子，解析 data 为对应结构体并调用。
// 返回 true 表示匹配到类型化钩子并已处理，false 表示无匹配。
func (c *Client) dispatchTypedEvent(ctx context.Context, gatewaySn, method string, raw []byte) bool {
	switch method {
	case MethodFlightTaskProgress:
		if c.onFlightTaskProgress != nil {
			var msg struct {
				Data FlightTaskProgressEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal FlightTaskProgressEvent failed: %v", err)
				return true
			}
			c.onFlightTaskProgress(ctx, gatewaySn, &msg.Data)
			return true
		}
	case MethodFlightTaskReady:
		if c.onFlightTaskReady != nil {
			var msg struct {
				Data FlightTaskReadyEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal FlightTaskReadyEvent failed: %v", err)
				return true
			}
			c.onFlightTaskReady(ctx, gatewaySn, &msg.Data)
			return true
		}
	case MethodReturnHomeInfo:
		if c.onReturnHomeInfo != nil {
			var msg struct {
				Data ReturnHomeInfoEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal ReturnHomeInfoEvent failed: %v", err)
				return true
			}
			c.onReturnHomeInfo(ctx, gatewaySn, &msg.Data)
			return true
		}
	case MethodCustomDataTransmissionFromPsdk:
		if c.onCustomDataFromPsdk != nil {
			var msg struct {
				Data CustomDataFromPsdkEvent `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal CustomDataFromPsdkEvent failed: %v", err)
				return true
			}
			c.onCustomDataFromPsdk(ctx, gatewaySn, &msg.Data)
			return true
		}
	case MethodHmsEventNotify:
		if c.onHmsEventNotify != nil {
			var msg struct {
				Data HmsEventData `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal HmsEventData failed: %v", err)
				return true
			}
			c.onHmsEventNotify(ctx, gatewaySn, &msg.Data)
			return true
		}
	}
	return false
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

// OnEvent 注册指定方法名的通用事件处理函数（兜底）。
// 当收到对应 method 的事件消息时，若没有类型化钩子匹配，将调用注册的 handler 进行处理。
//   - method: 事件方法名，对应 DJI Cloud API 中定义的事件类型
//   - handler: 事件处理函数
func (c *Client) OnEvent(method string, handler EventHandler) {
	c.eventHandlers[method] = handler
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
func (c *Client) OnState(handler func(ctx context.Context, deviceSn string, data *OsdMessage)) {
	c.onState = handler
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
	tid := uuid.New().String()
	bid := uuid.New().String()

	req := NewServiceRequest(tid, bid, method, data)
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("[dji-sdk] marshal request failed: %w", err)
	}

	topic := ServicesTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] send command: topic=%s method=%s tid=%s", topic, method, tid)

	reply, err := antsx.RequestReply(ctx, c.pending, tid, func() error {
		return c.mqttClient.Publish(ctx, topic, payload)
	})
	if err != nil {
		return tid, fmt.Errorf("[dji-sdk] command failed: method=%s tid=%s err=%w", method, tid, err)
	}

	if reply.Data.Result != 0 {
		return tid, fmt.Errorf("[dji-sdk] device returned error: method=%s tid=%s result=%d", method, tid, reply.Data.Result)
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
	logx.WithContext(ctx).Infof("[dji-sdk] send command (fire&forget): topic=%s method=%s tid=%s", topic, method, tid)

	if err := c.mqttClient.Publish(ctx, topic, payload); err != nil {
		return tid, fmt.Errorf("[dji-sdk] publish failed: method=%s tid=%s err=%w", method, tid, err)
	}
	return tid, nil
}

// ==================== 航线管理 ====================

// ExecuteFlightTask 执行航线飞行任务。
// 先下发 flighttask_prepare 命令进行航线准备，成功后再下发 flighttask_execute 命令开始执行。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - taskID: 飞行任务 ID
//   - wpmlURL: 航线文件（WPML 格式）的下载 URL
//   - 返回值 tid: 最后一次命令的事务 ID
//   - 返回值 err: 准备或执行阶段失败时的错误信息
func (c *Client) ExecuteFlightTask(ctx context.Context, gatewaySn, taskID, wpmlURL string) (string, error) {
	prepareData := &FlightTaskPrepareData{
		FlightID: taskID,
		TaskType: 0,
		File: FlightTaskFile{
			URL: wpmlURL,
		},
	}

	tid, err := c.SendCommand(ctx, gatewaySn, MethodFlightTaskPrepare, prepareData)
	if err != nil {
		return tid, fmt.Errorf("[dji-sdk] flighttask_prepare failed: %w", err)
	}

	executeData := &FlightTaskExecuteData{
		FlightID: taskID,
	}

	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskExecute, executeData)
}

// ExecuteFlightTaskWithOptions 使用自定义配置执行航线飞行任务。
// 允许调用方完全控制 FlightTaskPrepareData 的内容（如断点续飞、任务类型、返航高度等），
// 先下发 flighttask_prepare 命令，成功后再下发 flighttask_execute 命令开始执行。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - prepare: 自定义的航线准备参数
//   - 返回值 tid: 最后一次命令的事务 ID
//   - 返回值 err: 准备或执行阶段失败时的错误信息
func (c *Client) ExecuteFlightTaskWithOptions(ctx context.Context, gatewaySn string, prepare *FlightTaskPrepareData) (string, error) {
	tid, err := c.SendCommand(ctx, gatewaySn, MethodFlightTaskPrepare, prepare)
	if err != nil {
		return tid, fmt.Errorf("[dji-sdk] flighttask_prepare failed: %w", err)
	}

	executeData := &FlightTaskExecuteData{
		FlightID: prepare.FlightID,
	}

	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskExecute, executeData)
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

// ==================== PSDK ====================

// SendPsdkCommand 向 PSDK 负载发送数据，使用默认负载索引 "0"。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - payload: 要发送的负载数据内容
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) SendPsdkCommand(ctx context.Context, gatewaySn, payload string) (string, error) {
	data := &PsdkWriteData{
		PayloadIndex: "0",
		Data:         payload,
	}
	return c.SendCommand(ctx, gatewaySn, MethodPsdkWrite, data)
}

// SendPsdkCommandWithIndex 向指定索引的 PSDK 负载发送数据。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - payloadIndex: 负载索引，标识目标 PSDK 负载设备
//   - payload: 要发送的负载数据内容
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) SendPsdkCommandWithIndex(ctx context.Context, gatewaySn, payloadIndex, payload string) (string, error) {
	data := &PsdkWriteData{
		PayloadIndex: payloadIndex,
		Data:         payload,
	}
	return c.SendCommand(ctx, gatewaySn, MethodPsdkWrite, data)
}

// SendCustomDataToPsdk 通过自定义数据透传通道向 PSDK 负载发送数据。
// 使用 custom_data_transmission_to_psdk 方法和 CustomDataTransmissionData 结构体。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - value: 要透传的自定义数据内容
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) SendCustomDataToPsdk(ctx context.Context, gatewaySn, value string) (string, error) {
	data := &CustomDataTransmissionData{
		Value: value,
	}
	return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToPsdk, data)
}

// ==================== 远程调试 - 机巢控制 ====================

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

// BatteryStoreModeSwitchSwitch 切换电池保养存储模式。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - enable: 开关状态，1 为开启，0 为关闭
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) BatteryStoreModeSwitchSwitch(ctx context.Context, gatewaySn string, enable int) (string, error) {
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

// ==================== 属性设置 ====================

// SetProperty 设置设备属性。
// 通过 property/set 主题向设备下发属性设置命令并等待应答。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - properties: 要设置的属性键值对
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 发送失败、等待超时或设备返回错误时的错误信息
func (c *Client) SetProperty(ctx context.Context, gatewaySn string, properties PropertySetData) (string, error) {
	tid := uuid.New().String()
	bid := uuid.New().String()

	req := NewServiceRequest(tid, bid, MethodPropertySet, properties)
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("[dji-sdk] marshal property_set failed: %w", err)
	}

	topic := PropertySetTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] set property: topic=%s tid=%s", topic, tid)

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

// ==================== 指令飞行 ====================

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

// DrcModeEnter 进入指令飞行（DRC）模式。
// 进入 DRC 模式后，可通过 DRC 通道向无人机发送实时飞行控制指令。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: DRC 模式进入参数，包含 MQTT Broker 连接信息等
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DrcModeEnter(ctx context.Context, gatewaySn string, data *DrcModeEnterData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDrcModeEnter, data)
}

// DrcModeExit 退出指令飞行（DRC）模式。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DrcModeExit(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDrcModeExit, &DrcModeExitData{})
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

// DroneEmergencyStop 无人机紧急停桨。
// 危险操作：会立即停止所有电机，无人机将失去动力坠落。仅在紧急情况下使用。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) DroneEmergencyStop(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodDroneEmergencyStop, &DroneEmergencyStopData{})
}

// SendDrcCommand 通过 DRC 通道发送实时飞行控制指令。
// 该方法仅发布消息到 DRC 下行主题，不等待应答。适用于高频实时控制场景（如摇杆操控）。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 飞行控制数据，包含 X/Y/H/W 轴控制量和序列号
//   - 返回值: 序列化或发布失败时的错误信息
func (c *Client) SendDrcCommand(ctx context.Context, gatewaySn string, data *DroneControlData) error {
	payload, err := json.Marshal(NewServiceRequest(uuid.New().String(), uuid.New().String(), MethodDroneControl, data))
	if err != nil {
		return fmt.Errorf("[dji-sdk] marshal drc command failed: %w", err)
	}
	topic := DrcDownTopic(gatewaySn)
	return c.mqttClient.Publish(ctx, topic, payload)
}

// ==================== 相机控制 ====================

// CameraModeSwitch 切换相机拍摄模式（拍照/录像等）。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 相机模式切换参数，包含负载索引和目标模式
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) CameraModeSwitch(ctx context.Context, gatewaySn string, data *CameraModeSwitchData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodCameraModeSwitchCamera, data)
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

// ==================== 直播 ====================

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

// ==================== OSD / State 回调处理 ====================

// extractDeviceSnFromTopic 从 MQTT topic 中提取设备 SN。
// topic 格式: thing/product/{device_sn}/osd 或 thing/product/{device_sn}/state
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
	var msg OsdMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal state failed: %v, topic=%s", err, topic)
		return err
	}
	deviceSn := extractDeviceSnFromTopic(topic)
	c.onState(ctx, deviceSn, &msg)
	return nil
}

// ==================== 订阅管理 ====================

// SubscribeAll 批量订阅所有通配主题。
// 订阅 services_reply、events、property_set_reply 三类通配主题，并注册对应的消息回调处理函数。
//   - 返回值: 订阅失败时返回第一个遇到的错误，全部成功时返回 nil
func (c *Client) SubscribeAll() error {
	topics := map[string]func(context.Context, []byte, string, string) error{
		ServicesReplyTopicPattern():    c.HandleServicesReply,
		EventsTopicPattern():           c.HandleEvents,
		PropertySetReplyTopicPattern(): c.HandlePropertySetReply,
		OsdTopicPattern():              c.HandleOsd,
		StateTopicPattern():            c.HandleState,
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

// SubscribePropertySetReply 订阅 property_set_reply 通配主题。
// 注册 HandlePropertySetReply 作为消息回调处理函数。
//   - 返回值: 订阅失败时返回错误，成功时返回 nil
func (c *Client) SubscribePropertySetReply() error {
	return c.mqttClient.AddHandlerFunc(PropertySetReplyTopicPattern(), c.HandlePropertySetReply)
}

// ==================== 生命周期 ====================

// Close 关闭客户端，释放资源。
// 关闭待处理请求注册表，清理所有未完成的等待操作。
func (c *Client) Close() {
	c.pending.Close()
}
