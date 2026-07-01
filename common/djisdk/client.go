package djisdk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zero-service/common/mqttx"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// Client 封装**云平台侧**（上云接入端）的 MQTT 与协议能力：对设备 **Publish 下发**（如 services、property/set、drc/down），
// **通配订阅** 收设备上行（如 *reply、events、osd、requests、set_reply、drc/up 等）。见 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)。
// 含：services、events、property、osd/state、sys、requests、DRC 等，详情见包级 doc.go 与各 Topic 函数注释。
type Client struct {
	mqttClient mqttx.Client
	handlers   handlers
	pendingTTL time.Duration
	reply      ReplyConfig
	drcManager *drcManager
}

// Config bundles SDK-level configuration for MustNewClient (go-zero style).
// MqttConfig is the MQTT connection settings; Reply/Drc are folded into
// ClientOptions internally, leaving the opts argument for handler registration only.
type Config struct {
	MqttConfig mqttx.MqttConfig
	PendingTTL time.Duration `json:",default=30s"`
	Reply      ReplyConfig
	Drc        DrcConfig
}

func MustNewClient(cfg Config, opts ...ClientOption) *Client {
	opt := defaultClientOptions()
	if cfg.PendingTTL > 0 {
		opt.pendingTTL = cfg.PendingTTL
	}
	opt.reply = cfg.Reply
	opt.drcConfig = cfg.Drc
	for _, o := range opts {
		if o != nil {
			o(&opt)
		}
	}
	return buildClient(mqttx.MustNewClient(cfg.MqttConfig, replyRouters(opt.pendingTTL)...), &opt)
}

func NewClient(mqttClient mqttx.Client, opts ...ClientOption) *Client {
	opt := applyOptions(opts...)
	return buildClient(mqttClient, &opt)
}

func buildClient(mqttClient mqttx.Client, opt *clientOptions) *Client {
	c := &Client{
		mqttClient: mqttClient,
		handlers:   opt.handlers,
		pendingTTL: opt.pendingTTL,
		reply:      opt.reply,
	}
	if opt.drcConfig.HeartbeatInterval > 0 {
		c.drcManager = newDrcManager(c, opt.drcConfig, opt.drcManagerOpts...)
	}
	return c
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
	if c.handlers.onlineChecker != nil && !c.handlers.onlineChecker(gatewaySn) {
		err := fmt.Errorf("device offline, command rejected")
		logDjiSDKError(ctx, "command rejected", gatewaySn, method, "", err)
		return "", err
	}

	tid := uuid.New().String()
	bid := uuid.New().String()

	req := NewServiceRequest(tid, bid, method, data)
	payload, err := json.Marshal(req)
	if err != nil {
		err = fmt.Errorf("marshal request failed: %w", err)
		logDjiSDKError(ctx, "marshal request failed", gatewaySn, method, tid, err)
		return "", err
	}

	topic := ServicesTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] send_command %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", method, "tid", tid))

	reply, err := mqttx.RequestReply[*ServiceReply](ctx, c.mqttClient, servicesReplyTopicPattern(), tid, func() error {
		return c.mqttClient.Publish(ctx, topic, payload)
	}, c.pendingTTL)
	if err != nil {
		err = fmt.Errorf("command failed: %w", err)
		logDjiSDKError(ctx, "command failed", gatewaySn, method, tid, err)
		return tid, err
	}

	if reply.Data.Result != 0 {
		err = NewDJIError(reply.Data.Result)
		logDjiSDKError(ctx, "command rejected", gatewaySn, method, tid, err)
		return tid, err
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
		err = fmt.Errorf("marshal request failed: %w", err)
		logDjiSDKError(ctx, "marshal request failed", gatewaySn, method, tid, err)
		return "", err
	}

	topic := ServicesTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] send_command_fire_and_forget %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", method, "tid", tid))

	if err := c.mqttClient.Publish(ctx, topic, payload); err != nil {
		err = fmt.Errorf("publish failed: %w", err)
		logDjiSDKError(ctx, "publish failed", gatewaySn, method, tid, err)
		return tid, err
	}
	return tid, nil
}

// ==================== 设备属性（Properties） ====================

// PropertySet 设置设备属性（对应 DJI method: property_set）。
// 通过 property/set 主题向设备下发可写物模型属性，并等待 property/set_reply。
func (c *Client) PropertySet(ctx context.Context, gatewaySn string, properties PropertySetData) (string, error) {
	tid := uuid.New().String()
	bid := uuid.New().String()

	req := NewServiceRequest(tid, bid, MethodPropertySet, properties)
	payload, err := json.Marshal(req)
	if err != nil {
		err = fmt.Errorf("marshal property_set failed: %w", err)
		logDjiSDKError(ctx, "marshal property_set failed", gatewaySn, MethodPropertySet, tid, err)
		return "", err
	}

	topic := PropertySetTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] property_set %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", MethodPropertySet, "tid", tid))

	reply, err := mqttx.RequestReply[*ServiceReply](ctx, c.mqttClient, propertySetReplyTopicPattern(), tid, func() error {
		return c.mqttClient.Publish(ctx, topic, payload)
	}, c.pendingTTL)
	if err != nil {
		err = fmt.Errorf("property_set failed: %w", err)
		logDjiSDKError(ctx, "property_set failed", gatewaySn, MethodPropertySet, tid, err)
		return tid, err
	}

	if reply.Data.Result != 0 {
		err = NewDJIError(reply.Data.Result)
		logDjiSDKError(ctx, "property_set rejected", gatewaySn, MethodPropertySet, tid, err)
		return tid, err
	}

	return tid, nil
}

// ==================== 设备管理（Device） ====================

// 设备拓扑 update_topo 为 status 上行，由 OnUpdateTopo/OnStatus 处理。

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

// FlightTaskUndo 取消指定的飞行任务（对应 DJI method: flighttask_undo）。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - flightIDs: 要取消的飞行任务 ID 列表
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlightTaskUndo(ctx context.Context, gatewaySn string, flightIDs []string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskUndo, &FlightTaskCancelData{FlightIDs: flightIDs})
}

// PauseFlightTask 暂停当前正在执行的飞行任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 暂停参数（含 flight_id 和可选的 wayline_id）
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) PauseFlightTask(ctx context.Context, gatewaySn string, data *FlightTaskPauseData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskPause, data)
}

// FlightTaskRecovery 恢复已暂停的飞行任务（对应 DJI method: flighttask_recovery）。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 恢复参数（含 flight_id 和可选的 wayline_id）
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlightTaskRecovery(ctx context.Context, gatewaySn string, data *FlightTaskRecoveryData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskRecovery, data)
}

// StopFlightTask 强制停止当前航线任务。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 停止参数（含 flight_id 和可选的 wayline_id）
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) StopFlightTask(ctx context.Context, gatewaySn string, data *FlightTaskStopData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightTaskStop, data)
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

// DrcModeEnter 进入指令飞行（DRC）模式；经 **thing/.../services** 发 **drc_mode_enter** method，**services_reply** 为应答，非 drc/* topic。进入后在 **drc/down** 可发杆量（见 StickControl）。见 [DRC 文档](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html)。
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
//   - data: 停止参数，包含 fly_to_id
//   - 返回值 tid: 本次请求的事务 ID
//   - 返回值 err: 命令发送或设备返回错误时的错误信息
func (c *Client) FlyToPointStop(ctx context.Context, gatewaySn string, data *FlyToPointStopData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlyToPointStop, data)
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

// StickControl 经 drc/down 即发即忘地下发 stick_control 杆量（对应 DJI method: stick_control）。
// seq 位于顶层，data 包含 roll、pitch、throttle、yaw、gimbal_pitch，不等待 services_reply。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
//   - data: 杆量数据（DrcStickControlData）
//   - 返回值: 序列化或发布失败时的错误信息
func (c *Client) StickControl(ctx context.Context, gatewaySn string, seq int, data *DrcStickControlData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodStickControl, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

func (c *Client) publishDrcDown(ctx context.Context, gatewaySn string, msg *DrcDownMessage) (string, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		err = fmt.Errorf("marshal drc/down failed: %w", err)
		logDjiSDKError(ctx, "marshal drc/down failed", gatewaySn, msg.Method, msg.Tid, err)
		return msg.Tid, err
	}
	topic := DrcDownTopic(gatewaySn)
	logx.WithContext(ctx).Infof("[dji-sdk] drc_down %s", logFields("topic", topic, "gateway_sn", gatewaySn, "method", msg.Method, "tid", msg.Tid, "seq", msg.Seq))
	if err := c.mqttClient.Publish(ctx, topic, payload); err != nil {
		err = fmt.Errorf("publish drc/down failed: %w", err)
		logDjiSDKError(ctx, "publish drc/down failed", gatewaySn, msg.Method, msg.Tid, err)
		return msg.Tid, err
	}
	return msg.Tid, nil
}

func logDjiSDKError(ctx context.Context, msg, gatewaySn, method, tid string, err error) {
	fields := []logx.LogField{
		logx.Field("gateway_sn", gatewaySn),
	}
	if method != "" {
		fields = append(fields, logx.Field("method", method))
	}
	if tid != "" {
		fields = append(fields, logx.Field("tid", tid))
	}
	logx.WithContext(ctx).Errorw("[dji-sdk] "+msg+": "+err.Error(), fields...)
}

// SendDrcHeartBeat 经 drc/down 即发即忘地下发 heart_beat 心跳。
func (c *Client) SendDrcHeartBeat(ctx context.Context, gatewaySn string, dataTimestampMillis int64) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcHeartBeat, DrcHeartBeatDownData{Timestamp: dataTimestampMillis}, nil)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DroneEmergencyStop 经 drc/down 即发即忘地下发 drone_emergency_stop。
func (c *Client) DroneEmergencyStop(ctx context.Context, gatewaySn string, seq int) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDroneEmergencyStop, DroneEmergencyStopData{}, &seq)
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

// FlightAreasUpdate 触发自定义飞行区文件更新（仅通知信号）。
// flight_areas_update 为触发更新通知，不含文件数据；设备收到后通过 flight_areas_get 拉取文件。
//   - ctx: 请求上下文
//   - gatewaySn: 网关设备序列号
func (c *Client) FlightAreasUpdate(ctx context.Context, gatewaySn string) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodFlightAreasUpdate, &FlightAreasUpdateData{})
}

// ==================== PSDK 功能（PSDK） ====================

// PsdkUIResourceUpload 上传 PSDK UI 资源。
func (c *Client) PsdkUIResourceUpload(ctx context.Context, gatewaySn string, data *PsdkUIResourceUploadData) (string, error) {
	return c.SendCommand(ctx, gatewaySn, MethodPsdkUIResourceUpload, data)
}

// ==================== PSDK 互联互通（PSDK Transmit） ====================

// CustomDataTransmissionToPsdk 自定义数据透传至 PSDK 负载设备。
// 下行对应 custom_data_transmission_to_psdk；上行 custom_data_transmission_from_psdk 由 OnCustomDataTransmissionFromPsdk 处理。
func (c *Client) CustomDataTransmissionToPsdk(ctx context.Context, gatewaySn, value string) (string, error) {
	data := &CustomDataTransmissionData{
		Value: value,
	}
	return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToPsdk, data)
}

// ==================== ESDK 互联互通（ESDK Transmit） ====================

// CustomDataTransmissionToEsdk 自定义数据透传至 ESDK 设备。
func (c *Client) CustomDataTransmissionToEsdk(ctx context.Context, gatewaySn, value string) (string, error) {
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
func (c *Client) DrcForceLanding(ctx context.Context, gatewaySn string, seq int) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcForceLanding, struct{}{}, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcEmergencyLanding 经 drc/down 下发紧急降落。
func (c *Client) DrcEmergencyLanding(ctx context.Context, gatewaySn string, seq int) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcEmergencyLanding, struct{}{}, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcLinkageZoomSet 经 drc/down 设置红外联动变焦状态。
func (c *Client) DrcLinkageZoomSet(ctx context.Context, gatewaySn string, seq int, data *DrcLinkageZoomSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcLinkageZoomSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcVideoResolutionSet 经 drc/down 设置视频分辨率。
func (c *Client) DrcVideoResolutionSet(ctx context.Context, gatewaySn string, seq int, data *DrcVideoResolutionSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcVideoResolutionSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcIntervalPhotoSet 经 drc/down 设置定时拍参数。
func (c *Client) DrcIntervalPhotoSet(ctx context.Context, gatewaySn string, seq int, data *DrcIntervalPhotoSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcIntervalPhotoSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcInitialStateSubscribe 经 drc/down 订阅 DRC 初始状态。
func (c *Client) DrcInitialStateSubscribe(ctx context.Context, gatewaySn string, seq int) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcInitialStateSubscribe, struct{}{}, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcNightLightsStateSet 经 drc/down 设置夜航灯状态。
func (c *Client) DrcNightLightsStateSet(ctx context.Context, gatewaySn string, seq int, data *DrcNightLightsStateSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcNightLightsStateSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcStealthStateSet 经 drc/down 设置隐蔽模式状态。
func (c *Client) DrcStealthStateSet(ctx context.Context, gatewaySn string, seq int, data *DrcStealthStateSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcStealthStateSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraApertureValueSet 经 drc/down 设置相机光圈。
func (c *Client) DrcCameraApertureValueSet(ctx context.Context, gatewaySn string, seq int, data *DrcCameraApertureValueSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraApertureValueSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraShutterSet 经 drc/down 设置相机快门。
func (c *Client) DrcCameraShutterSet(ctx context.Context, gatewaySn string, seq int, data *DrcCameraShutterSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraShutterSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraIsoSet 经 drc/down 设置相机 ISO。
func (c *Client) DrcCameraIsoSet(ctx context.Context, gatewaySn string, seq int, data *DrcCameraIsoSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraIsoSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraMechanicalShutterSet 经 drc/down 设置机械快门。
func (c *Client) DrcCameraMechanicalShutterSet(ctx context.Context, gatewaySn string, seq int, data *DrcCameraMechanicalShutterSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraMechanicalShutterSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

// DrcCameraDewarpingSet 经 drc/down 设置镜头去畸变。
func (c *Client) DrcCameraDewarpingSet(ctx context.Context, gatewaySn string, seq int, data *DrcCameraDewarpingSetData) (string, error) {
	msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodDrcCameraDewarpingSet, data, &seq)
	return c.publishDrcDown(ctx, gatewaySn, msg)
}

func (c *Client) Close() {
	if c.drcManager != nil {
		c.drcManager.Close()
	}
	c.mqttClient.Close()
}

// ==================== DRC Manager API ====================

func (c *Client) EnableDrc(ctx context.Context, gatewaySn string, opts ...DrcEnableOption) error {
	if c.drcManager == nil {
		err := fmt.Errorf("drcManager not configured, use WithDrcConfig option")
		logDjiSDKError(ctx, "drc_manager not configured", gatewaySn, "", "", err)
		return err
	}
	var o drcEnableOptions
	for _, opt := range opts {
		opt(&o)
	}
	return c.drcManager.Enable(ctx, gatewaySn, o.maxTimeout)
}

func (c *Client) DisableDrc(ctx context.Context, gatewaySn string) error {
	if c.drcManager == nil {
		err := fmt.Errorf("drcManager not configured, use WithDrcConfig option")
		logDjiSDKError(ctx, "drc_manager not configured", gatewaySn, "", "", err)
		return err
	}
	return c.drcManager.Disable(ctx, gatewaySn)
}

func (c *Client) DrcNextSeq(gatewaySn string) (int, error) {
	if c.drcManager == nil {
		err := fmt.Errorf("drcManager not configured, use WithDrcConfig option")
		logx.Errorw("[dji-sdk] drc_manager not configured", logx.Field("gateway_sn", gatewaySn))
		return 0, err
	}
	return c.drcManager.GetNextSeq(gatewaySn)
}

func (c *Client) DrcStatus(gatewaySn string) (enabled bool, startedAt, lastHb time.Time, nextSeq int, alive bool) {
	if c.drcManager == nil {
		return
	}
	return c.drcManager.GetStatus(gatewaySn)
}
