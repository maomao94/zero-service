package djisdk

import "time"

// 平台在 status_reply、events_reply、requests_reply 等 data 内 result 的辅助码（与 6 位业务 error code 不同；简单枚举时与上云约定对齐）。
// 大疆侧常见：0=成功；**2 常表示超时**（见 [错误码](https://developer.dji.com/doc/cloud-api-tutorial/cn/error-code.html) 及各 MQTT 协议 data 说明），故 **禁止将 2 用作与超时无关的占位**（如未注册 handler 应使用 1 或其它与文档一致的值）。
const (
	PlatformResultOK           = 0
	PlatformResultHandlerError = 1 // 云侧未实现/未注册 handler、解包失败、非超时类内部错误
	PlatformResultTimeout      = 2 // 与文档中 result=2 表示**超时** 对齐；仅在实际超时或协议明确要求填 2 时使用
)

// ==================== 公共消息结构 ====================

// ServiceRequest 服务请求消息，用于云端向设备下发服务调用指令。
type ServiceRequest struct {
	// Tid 事务 ID，全局唯一标识一次请求，格式为 UUID 字符串。必填。
	Tid string `json:"tid"`
	// Bid 业务 ID，标识一个业务流程，格式为 UUID 字符串。必填。
	Bid string `json:"bid"`
	// Timestamp 消息时间戳，单位毫秒。必填。
	Timestamp int64 `json:"timestamp"`
	// Method 服务方法名，标识具体的服务调用。必填。
	Method string `json:"method"`
	// Gateway 网关设备 SN，标识消息来源网关。可选。
	Gateway string `json:"gateway,omitempty"`
	// Data 请求数据体，具体内容由 Method 决定。必填。
	Data any `json:"data"`
}

// ServiceReply 服务应答消息，设备对服务请求的响应。
type ServiceReply struct {
	// Tid 事务 ID，与请求中的 Tid 对应。必填。
	Tid string `json:"tid"`
	// Bid 业务 ID，与请求中的 Bid 对应。必填。
	Bid string `json:"bid"`
	// Timestamp 应答时间戳，单位毫秒。必填。
	Timestamp int64 `json:"timestamp"`
	// Method 服务方法名，与请求中的 Method 对应。必填。
	Method string `json:"method"`
	// Data 应答数据体，包含结果码和输出数据。必填。
	Data ServiceReplyData `json:"data"`
}

// ServiceReplyData 服务应答数据体。
type ServiceReplyData struct {
	// Result 结果码，0 表示成功，非 0 表示失败。必填。
	Result int `json:"result"`
	// Output 输出数据，具体内容由 Method 决定。可选。
	Output any `json:"output,omitempty"`
}

// EventMessage 事件消息，设备主动上报的事件通知。
type EventMessage struct {
	// Tid 事务 ID，全局唯一标识一次事件。必填。
	Tid string `json:"tid"`
	// Bid 业务 ID，标识一个业务流程。必填。
	Bid string `json:"bid"`
	// Timestamp 事件时间戳，单位毫秒。必填。
	Timestamp int64 `json:"timestamp"`
	// Method 事件方法名，标识具体的事件类型。必填。
	Method string `json:"method"`
	// Gateway 网关设备 SN，标识消息来源网关。可选。
	Gateway string `json:"gateway,omitempty"`
	// NeedReply 是否需要应答，1 表示需要，0 表示不需要。可选。
	NeedReply int `json:"need_reply,omitempty"`
	// Data 事件数据体，具体内容由 Method 决定。必填。
	Data any `json:"data"`
}

// EventReply 事件应答消息，云端对设备上报事件的响应。
type EventReply struct {
	// Tid 事务 ID，与事件中的 Tid 对应。必填。
	Tid string `json:"tid"`
	// Bid 业务 ID，与事件中的 Bid 对应。必填。
	Bid string `json:"bid"`
	// Timestamp 应答时间戳，单位毫秒。必填。
	Timestamp int64 `json:"timestamp"`
	// Method 事件方法名，与事件中的 Method 对应。必填。
	Method string `json:"method"`
	// Data 应答数据体，包含结果码。必填。
	Data EventReplyData `json:"data"`
}

// EventReplyData 事件应答数据体。
type EventReplyData struct {
	// Result 结果码，0 表示成功，非 0 表示失败。必填。
	Result int `json:"result"`
}

// OsdMessage OSD 遥测消息，设备定期上报的状态信息。
type OsdMessage struct {
	// Tid 事务 ID。必填。
	Tid string `json:"tid"`
	// Bid 业务 ID。必填。
	Bid string `json:"bid"`
	// Timestamp 消息时间戳，单位毫秒。必填。
	Timestamp int64 `json:"timestamp"`
	// Gateway 网关设备 SN。可选。
	Gateway string `json:"gateway,omitempty"`
	// Data OSD 遥测数据体，具体内容因设备类型而异。必填。
	Data any `json:"data"`
}

// RequestMessage 设备经 thing/.../requests 上行的报文。见
// [Requests 组织](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/organization.html)。
type RequestMessage struct {
	// Tid 事务 ID，全局唯一标识。必填。
	Tid string `json:"tid"`
	// Bid 业务 ID。必填。
	Bid string `json:"bid"`
	// Timestamp 消息时间戳，单位毫秒。必填。
	Timestamp int64 `json:"timestamp"`
	// Method 请求方法名。必填。
	Method string `json:"method"`
	// Gateway 网关设备 SN。可选。
	Gateway string `json:"gateway,omitempty"`
	// Data 请求数据体。必填。
	Data any `json:"data"`
}

// StatusMessage 由 sys/.../status 上行的状态。见
// [Status 设备](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html) 与上云状态/拓扑等说明。
type StatusMessage struct {
	// Tid 事务 ID。必填。
	Tid string `json:"tid"`
	// Bid 业务 ID。必填。
	Bid string `json:"bid"`
	// Timestamp 消息时间戳，单位毫秒。必填。
	Timestamp int64 `json:"timestamp"`
	// Method 状态方法名。必填。
	Method string `json:"method"`
	// Gateway 网关设备 SN。可选。
	Gateway string `json:"gateway,omitempty"`
	// Data 状态数据体。必填。
	Data any `json:"data"`
}

type RequestReply struct {
	Tid       string           `json:"tid"`
	Bid       string           `json:"bid"`
	Timestamp int64            `json:"timestamp"`
	Method    string           `json:"method"`
	Data      ServiceReplyData `json:"data"`
}

type StatusReply struct {
	Tid       string         `json:"tid"`
	Bid       string         `json:"bid"`
	Timestamp int64          `json:"timestamp"`
	Data      EventReplyData `json:"data"`
}

// NewServiceRequest 创建服务请求消息，自动填充当前毫秒时间戳。
func NewServiceRequest(tid, bid, method string, data any) *ServiceRequest {
	return &ServiceRequest{
		Tid:       tid,
		Bid:       bid,
		Timestamp: time.Now().UnixMilli(),
		Method:    method,
		Data:      data,
	}
}

// NewEventReply 创建事件应答消息，自动填充当前毫秒时间戳。
func NewEventReply(tid, bid, method string, result int) *EventReply {
	return &EventReply{
		Tid:       tid,
		Bid:       bid,
		Timestamp: time.Now().UnixMilli(),
		Method:    method,
		Data:      EventReplyData{Result: result},
	}
}

// ==================== 一、航线管理（Wayline Management） ====================

// FlightTaskPrepareData 航线任务准备数据。
// 对应 DJI Cloud API method: flighttask_prepare 的 data 载荷。
type FlightTaskPrepareData struct {
	// FlightID 航线任务 ID，全局唯一标识。必填。
	FlightID string `json:"flight_id"`
	// ExecuteTime 计划执行时间，Unix 毫秒时间戳；为 0 或不填表示立即执行。可选。
	ExecuteTime int64 `json:"execute_time,omitempty"`
	// TaskType 任务类型，0: 立即执行, 1: 定时执行, 2: 条件触发。必填。
	TaskType int `json:"task_type"`
	// WaylineType 航线类型，0: 普通航线, 1: 协调航线。可选。
	WaylineType int `json:"wayline_type,omitempty"`
	// File 航线文件信息，包含下载地址和指纹校验。必填。
	File FlightTaskFile `json:"file"`
	// BreakPoint 断点续飞信息，用于从指定断点恢复飞行。可选。
	BreakPoint *BreakPoint `json:"break_point,omitempty"`
	// RthAltitude 返航高度，单位米，取值范围 [20, 500]。可选。
	RthAltitude int `json:"rth_altitude,omitempty"`
	// OutOfControlAction 失控动作，0: 返航, 1: 悬停, 2: 降落。可选。
	OutOfControlAction int `json:"out_of_control_action,omitempty"`
	// ExitWaylineWhenRCLost 遥控器信号丢失时是否退出航线，0: 不退出, 1: 退出。可选。
	ExitWaylineWhenRCLost int `json:"exit_wayline_when_rc_lost,omitempty"`
	// SimulateMission 模拟任务配置，仅作为 flighttask_prepare.data.simulate_mission 子结构随 FlightTaskPrepare 下发。可选。
	SimulateMission *SimulateMission `json:"simulate_mission,omitempty"`
}

// FlightTaskFile 航线任务文件信息。
type FlightTaskFile struct {
	// URL 航线文件下载地址。必填。
	URL string `json:"url"`
	// Fingerprint 文件指纹，用于完整性校验，通常为 MD5 值。可选。
	Fingerprint string `json:"fingerprint,omitempty"`
}

// BreakPoint 断点续飞信息，描述航线中断恢复的位置。
type BreakPoint struct {
	// Index 断点序号，表示中断时的航点索引，int 类型，从 0 开始。必填。
	Index int `json:"index"`
	// State 断点状态，0: 在航段上, 1: 在航点上。必填。
	State int `json:"state"`
	// Progress 断点进度，取值范围 [0, 1]，表示当前航段已完成的比例。必填。
	Progress float64 `json:"progress"`
	// WaylineID 航线 ID，标识断点所在的航线。必填。
	WaylineID int `json:"wayline_id"`
}

// SimulateMission 模拟任务配置，用于在 flighttask_prepare.data.simulate_mission 中声明仿真飞行参数。
type SimulateMission struct {
	// IsEnable 是否启用模拟任务。必填。
	IsEnable bool `json:"is_enable"`
	// Latitude 模拟起飞点纬度，取值范围 [-90, 90]。可选，启用时必填。
	Latitude float64 `json:"latitude,omitempty"`
	// Longitude 模拟起飞点经度，取值范围 [-180, 180]。可选，启用时必填。
	Longitude float64 `json:"longitude,omitempty"`
}

// FlightTaskExecuteData 航线任务执行数据。
// 对应 DJI Cloud API method: flighttask_execute 的 data 载荷。
type FlightTaskExecuteData struct {
	// FlightID 航线任务 ID，与准备阶段的 FlightID 一致。必填。
	FlightID string `json:"flight_id"`
}

// FlightTaskCancelData 航线任务取消数据。
// 对应 DJI Cloud API method: flighttask_undo 的 data 载荷。
type FlightTaskCancelData struct {
	// FlightIDs 待取消的航线任务 ID 列表。必填。
	FlightIDs []string `json:"flight_ids"`
}

// ReturnSpecificHomeData 返航至指定点数据。
type ReturnSpecificHomeData struct {
	// Latitude 指定返航点纬度，取值范围 [-90, 90]。必填。
	Latitude float64 `json:"latitude"`
	// Longitude 指定返航点经度，取值范围 [-180, 180]。必填。
	Longitude float64 `json:"longitude"`
	// Height 指定返航点高度，单位米。必填。
	Height float64 `json:"height"`
}

// ==================== 一、航线管理 - 航线进度事件 ====================

// FlightTaskProgressData 航线任务进度事件数据。
type FlightTaskProgressData struct {
	// Ext 进度扩展信息，包含航线执行的详细状态。必填。
	Ext EventProgressExt `json:"ext"`
}

// EventProgressExt 航线进度扩展信息。
type EventProgressExt struct {
	// CurrentWaypointIndex 当前执行的航点索引，从 0 开始。必填。
	CurrentWaypointIndex int `json:"current_waypoint_index"`
	// WaylineMissionState 航线任务状态，参考 DJI 航线任务状态枚举。必填。
	WaylineMissionState int `json:"wayline_mission_state"`
	// MediaCount 已拍摄的媒体文件数量。必填。
	MediaCount int `json:"media_count"`
	// TrackID 航迹 ID。必填。
	TrackID string `json:"track_id"`
	// FlightID 航线任务 ID。必填。
	FlightID string `json:"flight_id"`
	// BreakPoint 断点续飞信息，任务中断时上报。可选。
	BreakPoint *BreakPoint `json:"break_point,omitempty"`
}

// ==================== 二、PSDK 自定义数据透传（PSDK Custom Data Transmission） ====================

// PsdkWriteData PSDK 数据写入请求，用于向 PSDK 负载发送数据。
type PsdkWriteData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"，如 "53-0"。必填。
	PayloadIndex string `json:"payload_index"`
	// Data 待发送的数据内容，Base64 编码字符串。必填。
	Data string `json:"data"`
}

// CustomDataTransmissionData 自定义数据透传消息，用于 PSDK 设备与云端之间的自定义数据传输。
type CustomDataTransmissionData struct {
	// Value 自定义消息内容，长度小于 256 字符。必填。
	Value string `json:"value"`
}

// ==================== 四、远程调试 - 机巢控制（Remote Debug） ====================

// DebugModeData 远程调试模式数据，开启或关闭调试模式，无额外参数。
type DebugModeData struct{}

// CoverData 机场开关舱盖指令数据，无额外参数。
type CoverData struct{}

// DroneData 无人机开关机指令数据，无额外参数。
type DroneData struct{}

// DeviceRebootData 设备重启指令数据，无额外参数。
type DeviceRebootData struct{}

// ChargeData 机场充电指令数据，无额外参数。
type ChargeData struct{}

// FormatData 存储格式化指令数据，无额外参数。
type FormatData struct{}

// SupplementLightData 补光灯控制指令数据，无额外参数。
type SupplementLightData struct{}

// BatteryStoreModeSwitchData 电池保养存储模式切换数据。
type BatteryStoreModeSwitchData struct {
	// Enable 是否启用电池存储模式，1: 启用, 0: 关闭。必填。
	Enable int `json:"enable"`
}

// AlarmStateSwitchData 机场声光报警开关数据。
type AlarmStateSwitchData struct {
	// Action 报警动作，0: 关闭, 1: 开启。必填。
	Action int `json:"action"`
}

// BatteryMaintenanceSwitchData 电池保养功能开关数据。
type BatteryMaintenanceSwitchData struct {
	// Enable 是否启用电池保养，1: 启用, 0: 关闭。必填。
	Enable int `json:"enable"`
}

// AirConditionerModeSwitchData 机场空调模式切换数据。
type AirConditionerModeSwitchData struct {
	// Action 空调动作，0: 关闭, 1: 制冷, 2: 制热, 3: 除湿。必填。
	Action int `json:"action"`
}

// ==================== 四、远程调试 - 事件进度 ====================

// DebugProgressData 远程调试进度事件数据。
type DebugProgressData struct {
	// Result 结果码，0 表示成功，非 0 表示失败。必填。
	Result int `json:"result"`
	// Output 进度输出详情。必填。
	Output DebugProgressOutput `json:"output"`
}

// DebugProgressOutput 远程调试进度输出。
type DebugProgressOutput struct {
	// Status 当前执行状态，如 "in_progress"、"ok"、"failed"。必填。
	Status string `json:"status"`
	// Progress 进度详情信息。可选，执行中时上报。
	Progress *DebugProgressDetail `json:"progress,omitempty"`
}

// DebugProgressDetail 远程调试进度详情。
type DebugProgressDetail struct {
	// Percent 进度百分比，取值范围 [0, 100]。必填。
	Percent int `json:"percent"`
	// StepKey 当前步骤标识，用于展示当前执行步骤的描述。可选。
	StepKey string `json:"step_key,omitempty"`
}

// ==================== 七、物模型属性（Property） ====================
// PropertySetData **仅用于云 → 设备** 的 `property/set` 载荷；由云平台/本服务 SetProperty 随 MethodPropertySet 一同下发。键为物模型可写属性名。
// 设备只读/遥测等场景见 **thing/.../state、osd** 等，勿与本「写属性」混用方向。
type PropertySetData map[string]any

// ==================== 三、指令飞行控制（Live Flight Controls / DRC） ====================
// 协议与 [DRC 上云](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html) 及 [DRC 杆量](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html#drc-%E6%9D%86%E9%87%8F%E6%8E%A7%E5%88%B6) 一致；飞控/模式类走 services 载荷，杆量用 DrcStickControlData 发 drc/down。

// DrcModeEnterData 进入指令飞行（DRC）模式请求数据（Dock3 协议）。
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html
type DrcModeEnterData struct {
	// MqttBroker DRC 专用 MQTT Broker 连接信息。必填。
	MqttBroker DrcMqttBroker `json:"mqtt_broker"`
	// OsdFrequency OSD 遥测数据上报频率，取值范围 [1, 30]，单位 Hz。必填。
	OsdFrequency int `json:"osd_frequency"`
	// HsiFrequency HSI（水平态势感知）数据上报频率，取值范围 [1, 30]，单位 Hz。必填。
	HsiFrequency int `json:"hsi_frequency"`
}

// DrcMqttBroker DRC 专用 MQTT Broker 连接信息。
type DrcMqttBroker struct {
	// Address MQTT Broker 地址，格式 "host:port"。必填。
	Address string `json:"address"`
	// ClientID MQTT 客户端 ID。必填。
	ClientID string `json:"client_id"`
	// Username MQTT 连接用户名。必填。
	Username string `json:"username"`
	// Password MQTT 连接密码。必填。
	Password string `json:"password"`
	// ExpireTime 连接过期时间，Unix 秒级时间戳。必填。
	ExpireTime int64 `json:"expire_time"`
	// EnableTLS 是否启用 TLS 加密连接，默认 false。
	EnableTLS bool `json:"enable_tls,omitempty"`
}

// DrcModeExitData 指令飞行模式退出数据，无额外参数。
type DrcModeExitData struct{}

// FlightAuthorityGrabData 飞行控制权抢夺数据，无额外参数。
type FlightAuthorityGrabData struct{}

// PayloadAuthorityGrabData 负载控制权抢夺数据，无额外参数。
type PayloadAuthorityGrabData struct{}

// TakeoffToPointData 一键起飞到指定坐标数据。
type TakeoffToPointData struct {
	// FlightID 飞行任务 ID，全局唯一标识。必填。
	FlightID string `json:"flight_id"`
	// TargetLatitude 目标纬度，取值范围 [-90, 90]。必填。
	TargetLatitude float64 `json:"target_latitude"`
	// TargetLongitude 目标经度，取值范围 [-180, 180]。必填。
	TargetLongitude float64 `json:"target_longitude"`
	// TargetHeight 目标高度，单位米，相对于起飞点的海拔高度。必填。
	TargetHeight float64 `json:"target_height"`
	// SecurityTakeoffHeight 安全起飞高度，单位米，无人机起飞后先爬升至此高度。必填。
	SecurityTakeoffHeight float64 `json:"security_takeoff_height"`
	// RthAltitude 返航高度，单位米。可选。
	RthAltitude float64 `json:"rth_altitude,omitempty"`
	// RCLostAction 遥控器信号丢失动作，0: 返航, 1: 悬停, 2: 降落。可选。
	RCLostAction int `json:"rc_lost_action,omitempty"`
	// MaxSpeed 最大飞行速度，单位 m/s。可选。
	MaxSpeed float64 `json:"max_speed,omitempty"`
	// CommanderFlightHeight 指令飞行高度，单位米。可选。
	CommanderFlightHeight float64 `json:"commander_flight_height,omitempty"`
}

// FlyToPointData 飞向指定坐标数据，支持多航点路径。
type FlyToPointData struct {
	// MaxSpeed 最大飞行速度，单位 m/s。必填。
	MaxSpeed float64 `json:"max_speed"`
	// FlyToID 飞行任务 ID，全局唯一标识。必填。
	FlyToID string `json:"fly_to_id"`
	// Points 航点列表，按顺序飞行。必填。
	Points []FlyToWaypoint `json:"points"`
}

// FlyToWaypoint 飞行航点坐标。
type FlyToWaypoint struct {
	// Latitude 航点纬度，取值范围 [-90, 90]。必填。
	Latitude float64 `json:"latitude"`
	// Longitude 航点经度，取值范围 [-180, 180]。必填。
	Longitude float64 `json:"longitude"`
	// Height 航点高度，单位米，相对于起飞点的海拔高度。必填。
	Height float64 `json:"height"`
}

// DrcStickControlData 与 DRC 杆量控制 stick_control 的 data 体一致。
// 由 SendDrcStickControl 放入 DrcDownMessage 后在 drc/down 上发布；seq 位于 DrcDownMessage 顶层。
type DrcStickControlData struct {
	Roll        float64 `json:"roll"`
	Pitch       float64 `json:"pitch"`
	Throttle    float64 `json:"throttle"`
	Yaw         float64 `json:"yaw"`
	GimbalPitch float64 `json:"gimbal_pitch"`
}

// ReturnHomeData 一键返航数据，无额外参数。
type ReturnHomeData struct{}

// ReturnHomeCancelData 取消返航数据，无额外参数。
type ReturnHomeCancelData struct{}

// DroneEmergencyStopData 无人机紧急停机数据，无额外参数。
type DroneEmergencyStopData struct{}

// ==================== 五、相机/云台控制（Camera & Gimbal） ====================

// CameraModeSwitchData 相机模式切换数据。
type CameraModeSwitchData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// CameraMode 相机模式，0: 拍照, 1: 录像。必填。
	CameraMode int `json:"camera_mode"`
}

// CameraPhotoTakeData 相机拍照数据。
type CameraPhotoTakeData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
}

// CameraRecordingStartData 相机开始录像数据。
type CameraRecordingStartData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
}

// CameraRecordingStopData 相机停止录像数据。
type CameraRecordingStopData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
}

// CameraFocalLengthSetData 相机焦距设置数据。
type CameraFocalLengthSetData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// CameraType 相机类型，0: 广角, 1: 变焦, 2: 红外。必填。
	CameraType int `json:"camera_type"`
	// ZoomFactor 变焦倍数，取值范围因镜头型号而异。必填。
	ZoomFactor float64 `json:"zoom_factor"`
}

// GimbalResetData 云台重置数据。
type GimbalResetData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// ResetMode 重置模式，0: 回中, 1: 朝下, 2: 偏航回中。必填。
	ResetMode int `json:"reset_mode"`
}

// CameraAimData 相机对焦/瞄准数据。
type CameraAimData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// CameraType 相机类型，0: 广角, 1: 变焦, 2: 红外。必填。
	CameraType int `json:"camera_type"`
	// Locked 是否锁定目标。必填。
	Locked bool `json:"locked"`
	// X 瞄准点 X 坐标，归一化值，取值范围 [0, 1]，左上角为原点。必填。
	X float64 `json:"x"`
	// Y 瞄准点 Y 坐标，归一化值，取值范围 [0, 1]，左上角为原点。必填。
	Y float64 `json:"y"`
}

// CameraLookAtData 相机看向指定地理坐标数据。
type CameraLookAtData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// Latitude 目标纬度，取值范围 [-90, 90]。必填。
	Latitude float64 `json:"latitude"`
	// Longitude 目标经度，取值范围 [-180, 180]。必填。
	Longitude float64 `json:"longitude"`
	// Height 目标高度，单位米。必填。
	Height float64 `json:"height"`
}

// CameraPointFocusActionData 相机指点对焦数据。
type CameraPointFocusActionData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// CameraType 相机类型，0: 广角, 1: 变焦, 2: 红外。必填。
	CameraType int `json:"camera_type"`
	// X 对焦点 X 坐标，归一化值，取值范围 [0, 1]。必填。
	X float64 `json:"x"`
	// Y 对焦点 Y 坐标，归一化值，取值范围 [0, 1]。必填。
	Y float64 `json:"y"`
}

// CameraScreenSplitData 相机画面分屏数据。
type CameraScreenSplitData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// Enable 是否开启分屏，true: 开启, false: 关闭。必填。
	Enable bool `json:"enable"`
}

// CameraStorageSetData 照片/视频存储设置数据。
type CameraStorageSetData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// StorageType 存储位置，0: 机载存储, 1: SD 卡。必填。
	StorageType int `json:"storage_type"`
}

// CameraScreenDragData 相机画面拖动数据。
type CameraScreenDragData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// CameraType 相机类型，0: 广角, 1: 变焦, 2: 红外。必填。
	CameraType int `json:"camera_type"`
	// X 拖动起点 X 坐标，归一化值，取值范围 [0, 1]。必填。
	X float64 `json:"x"`
	// Y 拖动起点 Y 坐标，归一化值，取值范围 [0, 1]。必填。
	Y float64 `json:"y"`
}

// CameraIrMeteringPointData 红外点测温数据。
type CameraIrMeteringPointData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// X 测温点 X 坐标，归一化值，取值范围 [0, 1]。必填。
	X float64 `json:"x"`
	// Y 测温点 Y 坐标，归一化值，取值范围 [0, 1]。必填。
	Y float64 `json:"y"`
}

// CameraIrMeteringAreaData 红外区域测温数据。
type CameraIrMeteringAreaData struct {
	// PayloadIndex 负载设备索引，格式为 "机型-挂载位置"。必填。
	PayloadIndex string `json:"payload_index"`
	// X 区域起点 X 坐标，归一化值，取值范围 [0, 1]。必填。
	X float64 `json:"x"`
	// Y 区域起点 Y 坐标，归一化值，取值范围 [0, 1]。必填。
	Y float64 `json:"y"`
	// Width 区域宽度，归一化值，取值范围 [0, 1]。必填。
	Width float64 `json:"width"`
	// Height 区域高度，归一化值，取值范围 [0, 1]。必填。
	Height float64 `json:"height"`
}

// ==================== 六、直播管理（Live） ====================

// LiveStartPushData 直播推流启动数据。
type LiveStartPushData struct {
	// URLType 推流协议类型，0: AGORA, 1: RTMP, 2: RTSP, 3: GB28181。必填。
	URLType int `json:"url_type"`
	// URL 推流地址。必填。
	URL string `json:"url"`
	// VideoID 视频流 ID，格式为 "SN/机型-挂载位置/镜头类型"。必填。
	VideoID string `json:"video_id"`
	// VideoQuality 视频质量，0: 自适应, 1: 流畅, 2: 标清, 3: 高清, 4: 超清。必填。
	VideoQuality int `json:"video_quality"`
}

// LiveStopPushData 直播推流停止数据。
type LiveStopPushData struct {
	// VideoID 视频流 ID，与启动时一致。必填。
	VideoID string `json:"video_id"`
}

// LiveSetQualityData 直播画质设置数据。
type LiveSetQualityData struct {
	// VideoID 视频流 ID。必填。
	VideoID string `json:"video_id"`
	// VideoQuality 视频质量，0: 自适应, 1: 流畅, 2: 标清, 3: 高清, 4: 超清。必填。
	VideoQuality int `json:"video_quality"`
}

// LiveLensChangeData 直播镜头切换数据。
type LiveLensChangeData struct {
	// VideoID 视频流 ID。必填。
	VideoID string `json:"video_id"`
	// VideoType 镜头类型，0: 广角, 1: 变焦, 2: 红外。必填。
	VideoType int `json:"video_type"`
}

// LiveCameraChangeData 直播相机切换数据。
// 对应 DJI Cloud API method: live_camera_change 的 data 载荷。
type LiveCameraChangeData struct {
	// VideoID 视频流 ID。必填。
	VideoID string `json:"video_id"`
	// CameraIndex 目标相机索引，格式 type-subtype-gimbalIndex。必填。
	CameraIndex string `json:"camera_index"`
}

// MediaUploadFlighttaskMediaPrioritizeData 优先上传航线任务媒体请求数据。
type MediaUploadFlighttaskMediaPrioritizeData struct {
	// FlightID 航线任务 ID。必填。
	FlightID string `json:"flight_id"`
}

// MediaFastUploadData 快速上传媒体文件请求数据。
type MediaFastUploadData struct {
	// FileID 媒体文件 ID。必填。
	FileID string `json:"file_id"`
}

// MediaHighestPriorityUploadFlighttaskData 最高优先级上传航线任务媒体请求数据。
type MediaHighestPriorityUploadFlighttaskData struct {
	// FlightID 航线任务 ID。必填。
	FlightID string `json:"flight_id"`
}

// RemoteLogFileListData 远程日志文件列表查询请求数据。
type RemoteLogFileListData struct {
	// DeviceSN 目标设备序列号。可选。
	DeviceSN string `json:"device_sn,omitempty"`
	// Module 日志模块名称。可选。
	Module string `json:"module,omitempty"`
}

// RemoteLogFileUploadStartData 远程日志文件上传启动请求数据。
type RemoteLogFileUploadStartData struct {
	// Files 待上传的远程日志文件列表。必填。
	Files []RemoteLogFile `json:"files"`
}

// RemoteLogFileUploadUpdateData 远程日志文件上传更新请求数据。
type RemoteLogFileUploadUpdateData struct {
	// Files 更新后的远程日志文件列表。必填。
	Files []RemoteLogFile `json:"files"`
}

// RemoteLogFileUploadCancelData 远程日志文件上传取消请求数据。
type RemoteLogFileUploadCancelData struct {
	// Files 待取消上传的远程日志文件列表。必填。
	Files []RemoteLogFile `json:"files"`
}

// RemoteLogFileUploadResultEvent 远程日志文件上传结果事件数据。
type RemoteLogFileUploadResultEvent struct {
	// Files 远程日志文件上传结果列表。必填。
	Files []RemoteLogFileUploadResult `json:"files"`
}

// RemoteLogFileUploadProgressEvent 远程日志文件上传进度事件数据。
type RemoteLogFileUploadProgressEvent struct {
	// Files 远程日志文件上传进度列表。必填。
	Files []RemoteLogFileUploadProgress `json:"files"`
}

// RemoteLogFile 远程日志文件信息。
type RemoteLogFile struct {
	// DeviceSN 设备序列号。可选。
	DeviceSN string `json:"device_sn,omitempty"`
	// Module 日志模块名称。可选。
	Module string `json:"module,omitempty"`
	// Key 日志文件唯一标识。可选。
	Key string `json:"key,omitempty"`
	// Name 日志文件名称。可选。
	Name string `json:"name,omitempty"`
	// URL 日志文件上传地址或访问地址。可选。
	URL string `json:"url,omitempty"`
	// Size 日志文件大小，单位字节。可选。
	Size int64 `json:"size,omitempty"`
}

// RemoteLogFileUploadResult 远程日志文件上传结果。
type RemoteLogFileUploadResult struct {
	// DeviceSN 设备序列号。可选。
	DeviceSN string `json:"device_sn,omitempty"`
	// Module 日志模块名称。可选。
	Module string `json:"module,omitempty"`
	// Key 日志文件唯一标识。可选。
	Key string `json:"key,omitempty"`
	// Result 上传结果码。必填。
	Result int `json:"result"`
	// Output 上传结果扩展输出。可选。
	Output any `json:"output,omitempty"`
}

// RemoteLogFileUploadProgress 远程日志文件上传进度。
type RemoteLogFileUploadProgress struct {
	// DeviceSN 设备序列号。可选。
	DeviceSN string `json:"device_sn,omitempty"`
	// Module 日志模块名称。可选。
	Module string `json:"module,omitempty"`
	// Key 日志文件唯一标识。可选。
	Key string `json:"key,omitempty"`
	// Progress 上传进度，取值范围 [0, 100]。必填。
	Progress int `json:"progress"`
}

// ConfigUpdateData 设备配置更新请求数据。
type ConfigUpdateData struct {
	// ConfigScope 配置作用域。可选。
	ConfigScope string `json:"config_scope,omitempty"`
	// Config 配置键值内容。必填。
	Config map[string]any `json:"config"`
}

// ==================== 八、固件管理（Firmware） ====================

// OtaCreateData OTA 固件升级创建数据。
type OtaCreateData struct {
	// Devices 待升级的设备列表。必填。
	Devices []OtaDevice `json:"devices"`
}

// OtaDevice OTA 升级设备信息。
type OtaDevice struct {
	// SN 设备序列号。必填。
	SN string `json:"sn"`
	// ProductVersion 目标固件版本号。必填。
	ProductVersion string `json:"product_version"`
	// FirmwareUpgradeType 固件升级类型，1: 普通升级, 2: 一致性升级。必填。
	FirmwareUpgradeType int `json:"firmware_upgrade_type"`
}

// ==================== 九、设备管理（Device Management） ====================

// TopoUpdateData 设备拓扑更新数据，用于设备上线/下线通知。
type TopoUpdateData struct {
	// Type 设备类型，参考 DJI 设备类型枚举。必填。
	Type int `json:"type"`
	// SubType 设备子类型，参考 DJI 设备子类型枚举。必填。
	SubType int `json:"sub_type"`
	// DeviceSecret 设备密钥，用于身份验证。必填。
	DeviceSecret string `json:"device_secret"`
	// NonceSig 随机签名，用于防重放攻击。可选。
	NonceSig string `json:"nonce_sig,omitempty"`
	// Timestamp 签名时间戳，单位毫秒。可选。
	Timestamp int64 `json:"timestamp,omitempty"`
	// Version 设备版本信息，包含固件和硬件版本。可选。
	Version TopoVersion `json:"version,omitempty"`
	// SubDevices 子设备列表，如无人机挂载的负载设备。可选。
	SubDevices []TopoSubDevice `json:"sub_devices,omitempty"`
}

// TopoVersion 设备版本信息。
type TopoVersion struct {
	// FirmwareVersion 固件版本号，如 "01.00.0001"。必填。
	FirmwareVersion string `json:"firmware_version"`
	// HardwareVersion 硬件版本号，如 "1.0"。必填。
	HardwareVersion string `json:"hardware_version"`
}

// TopoSubDevice 拓扑子设备信息。
type TopoSubDevice struct {
	// SN 子设备序列号。必填。
	SN string `json:"sn"`
	// Type 子设备类型。必填。
	Type int `json:"type"`
	// SubType 子设备子类型。必填。
	SubType int `json:"sub_type"`
	// Index 子设备索引标识。必填。
	Index string `json:"index"`
	// Version 子设备版本信息。可选。
	Version TopoVersion `json:"version,omitempty"`
}

// ==================== 设备主动上报事件（方向 up: 设备->云平台） ====================
// 以下结构体对应 thing/product/{gateway_sn}/events topic 中，设备主动推送的事件数据。
// 这些事件不由网关发起请求触发，而是设备在特定条件下自动上报。

// CustomDataFromPsdkEvent PSDK 自定义数据上报事件。
// 方向 up：设备→云平台。
// 对应 method: custom_data_transmission_from_psdk。
// 当 PSDK 负载设备有自定义数据需要上报时，通过 events topic 主动推送此消息。
type CustomDataFromPsdkEvent struct {
	// Value 自定义消息内容，长度小于 256 字符。必填。
	Value string `json:"value"`
}

// FlightTaskProgressEvent 航线任务进度上报事件。
// 方向 up：设备→云平台。
// 对应 method: flighttask_progress。
// 机巢在执行航线任务过程中，主动定频上报当前任务执行进度。
type FlightTaskProgressEvent struct {
	// Ext 扩展内容，包含航线执行的详细进度信息。必填。
	Ext FlightTaskProgressExt `json:"ext"`
}

// FlightTaskProgressExt 航线任务进度扩展信息。
type FlightTaskProgressExt struct {
	// CurrentWaypointIndex 当前执行到的航点索引。必填。
	CurrentWaypointIndex int `json:"current_waypoint_index"`
	// WaylineMissionState 航线任务状态。必填。
	// 0: 断连, 1: 不支持该航点, 2: 航线准备状态, 3: 航线文件上传中,
	// 4: 准备状态（已触发开始）, 5: 进入航线到第一个航点, 6: 航线执行中,
	// 7: 航线中断（暂停/异常）, 8: 航线恢复, 9: 航线停止
	WaylineMissionState int `json:"wayline_mission_state"`
	// MediaCount 本次航线任务执行产生的媒体文件数量。必填。
	MediaCount int `json:"media_count"`
	// TrackID 航迹 ID。必填。
	TrackID string `json:"track_id"`
	// FlightID 任务 ID。必填。
	FlightID string `json:"flight_id"`
	// BreakPoint 航线断点信息，用于断点续飞。可选。
	BreakPoint *FlightTaskBreakPoint `json:"break_point,omitempty"`
}

// FlightTaskBreakPoint 航线断点信息，用于断点续飞场景。
type FlightTaskBreakPoint struct {
	// Index 断点序号。必填。
	Index int `json:"index"`
	// State 断点状态，0: 在航段上, 1: 在航点上。必填。
	State int `json:"state"`
	// Progress 当前航段进度，取值范围 [0, 1.0]。必填。
	Progress float64 `json:"progress"`
	// WaylineID 航线 ID。必填。
	WaylineID int `json:"wayline_id"`
	// BreakReason 中断原因。可选。
	// 0: 无异常, 1: Mission ID 不存在, 2: 不常见错误,
	// 257: 航线已开始不能再次开始, 258: 当前状态无法中断,
	// 259: 航线未开始不能结束, 513: 航线执行到达最大飞行距离
	BreakReason int `json:"break_reason,omitempty"`
}

// FlightTaskReadyEvent 任务就绪通知事件。
// 方向 up：设备→云平台。
// 对应 method: flighttask_ready。
// 当机巢中有任务满足就绪条件时，主动上报可执行的任务 ID 列表。
type FlightTaskReadyEvent struct {
	// FlightIDs 满足任务就绪条件的任务 ID 集合。必填。
	FlightIDs []string `json:"flight_ids"`
}

// ReturnHomeInfoEvent 返航信息事件。
// 方向 up：设备→云平台。
// 对应 method: return_home_info。
// 设备在返航时主动上报规划的返航路径和相关信息。
type ReturnHomeInfoEvent struct {
	// PlannedPathPoints 规划的返航轨迹点列表。必填。
	PlannedPathPoints []PathPoint `json:"planned_path_points"`
	// LastPointType 返航路径最后一个点的类型。必填。
	// 0: 轨迹最后一个点在返航点上空, 1: 轨迹最后一个点不在返航点上空
	LastPointType int `json:"last_point_type"`
	// FlightID 任务 ID。必填。
	FlightID string `json:"flight_id"`
	// HomeDockSn 返航目标机场的 SN。可选，蛙跳任务场景下上报。
	HomeDockSn string `json:"home_dock_sn,omitempty"`
	// MultiDockHomeInfo 蛙跳任务机场返航信息列表。可选，普通任务无该字段。
	MultiDockHomeInfo []DockHomeInfo `json:"multi_dock_home_info,omitempty"`
}

// PathPoint 轨迹坐标点。
type PathPoint struct {
	// Latitude 纬度，角度值，取值范围 [-90, 90]，精度到小数点后 6 位。必填。
	Latitude float64 `json:"latitude"`
	// Longitude 经度，角度值，取值范围 [-180, 180]，精度到小数点后 6 位。必填。
	Longitude float64 `json:"longitude"`
	// Height 高度，单位米，椭球高。必填。
	Height float64 `json:"height"`
}

// DockHomeInfo 蛙跳任务机场返航信息。
type DockHomeInfo struct {
	// SN 机场序列号。必填。
	SN string `json:"sn"`
	// PlanStatus 路径规划状态。必填。
	// 0: 规划失败或正在规划中, 1: 规划路径不可达,
	// 2: 规划路径因电量不可达, 3: 目标可达
	PlanStatus int `json:"plan_status"`
	// EstimatedBatteryConsumption 预估电量消耗，取值范围 [0, 100]。必填。
	EstimatedBatteryConsumption int `json:"estimated_battery_consumption"`
	// HomeDistance 到 home 点的距离，单位米。必填。
	HomeDistance float64 `json:"home_distance"`
}

// HmsEventData HMS 健康告警事件数据。
// 方向 up：设备→云平台。
// 对应 method: hms。
// 设备上报健康管理系统的告警和状态事件。
type HmsEventData struct {
	// List HMS 告警列表。必填。
	List []HmsItem `json:"list"`
}

// HmsItem HMS 健康告警条目。
type HmsItem struct {
	// HmsID 告警 ID，全局唯一标识。必填。
	HmsID string `json:"hms_id"`
	// Level 告警级别。必填。
	// 0: 通知, 1: 提示, 2: 警告, 3: 严重
	Level int `json:"level"`
	// Module 告警模块标识。必填。
	// 0: 飞行器, 3: 机场
	Module int `json:"module"`
	// InTheSky 告警发生时飞行器是否在空中。必填。
	// 0: 地面, 1: 空中
	InTheSky int `json:"in_the_sky"`
	// Code 告警码，格式为 "fpv_tip_xxxxxxxxx"。必填。
	Code string `json:"code"`
	// Imminent 是否为紧急告警。必填。
	Imminent bool `json:"imminent"`
	// Key 告警标识 Key。必填。
	Key string `json:"key"`
}
