package djisdk

import "fmt"

// ==================== Thing Topic ====================
//
// Thing Topic 用于设备物模型相关的消息通信，包括遥测数据上报、
// 云端服务下发、设备事件上报、设备属性请求等。
// 基础路径格式: thing/product/{gateway_sn}/{channel}

// OsdTopic 返回设备遥测数据（OSD）上报 Topic。
// 路径格式: thing/product/{device_sn}/osd
// 方向: 设备 → 云平台
// 用途: 设备定期推送实时遥测数据（飞行姿态、GPS 坐标、电池电量等）至云平台。
func OsdTopic(deviceSn string) string {
	return fmt.Sprintf("thing/product/%s/osd", deviceSn)
}

// OsdTopicPattern 返回 OSD Topic 的通配订阅模式。
// 路径格式: thing/product/+/osd
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有设备的遥测数据，"+" 匹配任意设备 SN。
func OsdTopicPattern() string {
	return "thing/product/+/osd"
}

// StateTopic 返回设备状态上报 Topic。
// 路径格式: thing/product/{device_sn}/state
// 方向: 设备 → 云平台
// 用途: 设备上报自身状态信息（固件版本、在线状态、设备能力集等），
// 与 OSD 不同，state 侧重于设备元信息而非实时飞行数据。
func StateTopic(deviceSn string) string {
	return fmt.Sprintf("thing/product/%s/state", deviceSn)
}

// StateTopicPattern 返回 State Topic 的通配订阅模式。
// 路径格式: thing/product/+/state
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有设备的状态变更通知。
func StateTopicPattern() string {
	return "thing/product/+/state"
}

// ServicesTopic 返回云平台下发服务调用的 Topic。
// 路径格式: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备
// 用途: 云平台向网关设备下发服务指令（如航线任务下发、设备控制、固件升级等）。
func ServicesTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/services", gatewaySn)
}

// ServicesReplyTopic 返回设备响应服务调用的 Topic。
// 路径格式: thing/product/{gateway_sn}/services_reply
// 方向: 设备 → 云平台
// 用途: 设备在处理完云平台下发的 services 指令后，通过此 Topic 返回执行结果。
func ServicesReplyTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/services_reply", gatewaySn)
}

// ServicesReplyTopicPattern 返回 ServicesReply Topic 的通配订阅模式。
// 路径格式: thing/product/+/services_reply
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有网关设备的服务调用响应。
func ServicesReplyTopicPattern() string {
	return "thing/product/+/services_reply"
}

// EventsTopic 返回设备事件上报 Topic。
// 路径格式: thing/product/{gateway_sn}/events
// 方向: 设备 → 云平台
// 用途: 设备主动上报事件通知（如任务进度、HMS 健康告警、文件上传进度等）。
func EventsTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/events", gatewaySn)
}

// EventsTopicPattern 返回 Events Topic 的通配订阅模式。
// 路径格式: thing/product/+/events
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有网关设备的事件上报。
func EventsTopicPattern() string {
	return "thing/product/+/events"
}

// EventsReplyTopic 返回云平台对设备事件的应答 Topic。
// 路径格式: thing/product/{gateway_sn}/events_reply
// 方向: 云平台 → 设备
// 用途: 云平台在收到设备 events 消息后，通过此 Topic 回复确认，通知设备事件已接收。
func EventsReplyTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/events_reply", gatewaySn)
}

// RequestsTopic 返回设备自定义请求上报 Topic。
// 路径格式: thing/product/{gateway_sn}/requests
// 方向: 设备 → 云平台
// 用途: 设备主动向云平台发起请求（如配置拉取、临时凭证获取等），
// 需要云平台通过 requests_reply 进行响应。
func RequestsTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/requests", gatewaySn)
}

// RequestsTopicPattern 返回 Requests Topic 的通配订阅模式。
// 路径格式: thing/product/+/requests
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有网关设备的请求消息。
func RequestsTopicPattern() string {
	return "thing/product/+/requests"
}

// RequestsReplyTopic 返回云平台对设备请求的响应 Topic。
// 路径格式: thing/product/{gateway_sn}/requests_reply
// 方向: 云平台 → 设备
// 用途: 云平台在收到设备 requests 消息后，通过此 Topic 下发响应数据。
func RequestsReplyTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/requests_reply", gatewaySn)
}

// ==================== Property Topic ====================
//
// Property Topic 用于设备属性的远程设置，
// 云平台可通过该通道修改设备可写属性（如夜航灯开关、避障距离等）。
// 基础路径格式: thing/product/{gateway_sn}/property/{channel}

// PropertySetTopic 返回云平台下发属性设置的 Topic。
// 路径格式: thing/product/{gateway_sn}/property/set
// 方向: 云平台 → 设备
// 用途: 云平台向网关设备下发属性修改指令，设置设备可写属性值。
func PropertySetTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/property/set", gatewaySn)
}

// PropertySetReplyTopic 返回设备响应属性设置的 Topic。
// 路径格式: thing/product/{gateway_sn}/property/set_reply
// 方向: 设备 → 云平台
// 用途: 设备在处理完属性设置指令后，通过此 Topic 返回设置结果。
func PropertySetReplyTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/property/set_reply", gatewaySn)
}

// PropertySetReplyTopicPattern 返回 PropertySetReply Topic 的通配订阅模式。
// 路径格式: thing/product/+/property/set_reply
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有网关设备的属性设置响应。
func PropertySetReplyTopicPattern() string {
	return "thing/product/+/property/set_reply"
}

// ==================== Sys Topic ====================
//
// Sys Topic 用于设备上下线管理和拓扑关系维护，
// 属于系统级通道，独立于物模型业务。
// 基础路径格式: sys/product/{gateway_sn}/{channel}

// StatusTopic 返回设备上下线状态上报 Topic。
// 路径格式: sys/product/{gateway_sn}/status
// 方向: 设备 → 云平台
// 用途: 网关设备上报自身及子设备的上线（online）/ 下线（offline）/ 拓扑更新状态。
func StatusTopic(gatewaySn string) string {
	return fmt.Sprintf("sys/product/%s/status", gatewaySn)
}

// StatusTopicPattern 返回 Status Topic 的通配订阅模式。
// 路径格式: sys/product/+/status
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有网关设备的上下线状态变更。
func StatusTopicPattern() string {
	return "sys/product/+/status"
}

// StatusReplyTopic 返回云平台对设备状态上报的应答 Topic。
// 路径格式: sys/product/{gateway_sn}/status_reply
// 方向: 云平台 → 设备
// 用途: 云平台在收到设备上下线通知后，通过此 Topic 回复确认。
func StatusReplyTopic(gatewaySn string) string {
	return fmt.Sprintf("sys/product/%s/status_reply", gatewaySn)
}

// ==================== DRC Topic ====================
//
// DRC（Device Remote Control）Topic 用于设备远程实时控制，
// 通过低延迟双向通道实现飞行器的实时遥控操作（如虚拟摇杆、云台控制等）。
// 基础路径格式: thing/product/{gateway_sn}/drc/{direction}

// DrcUpTopic 返回设备上行 DRC 消息 Topic。
// 路径格式: thing/product/{gateway_sn}/drc/up
// 方向: 设备 → 云平台
// 用途: 设备通过此 Topic 上报 DRC 通道的遥控响应数据和状态反馈（如控制权状态、心跳等）。
func DrcUpTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/drc/up", gatewaySn)
}

// DrcUpTopicPattern 返回 DRC Up Topic 的通配订阅模式。
// 路径格式: thing/product/+/drc/up
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有网关设备的 DRC 上行消息。
func DrcUpTopicPattern() string {
	return "thing/product/+/drc/up"
}

// DrcDownTopic 返回云平台下行 DRC 控制指令 Topic。
// 路径格式: thing/product/{gateway_sn}/drc/down
// 方向: 云平台 → 设备
// 用途: 云平台通过此 Topic 向设备下发实时遥控指令（如虚拟摇杆、云台角度、相机操作等）。
func DrcDownTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/drc/down", gatewaySn)
}
