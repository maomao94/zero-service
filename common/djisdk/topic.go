package djisdk

import "fmt"

// MQTT Topic 与方向说明以 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html) 为准。
// 本包运行于**上云/云平台侧**（或对接云平台的网关服务）：对 thing/product/{gateway_sn} 的 **Publish** 多为**云 → 设备** 下发；**通配订阅 +** 多为**收设备 → 云** 的上行。
// 物模型**属性设置**（property/set）仅表示**云平台向设备写入可写属性**；**不是**设备向云「设置」属性。设备侧执行结果经 **property/set_reply** 回云（与 services 的 reply 成对思想一致）。
// **DRC 通道**（见下方 DRC 小节）：**drc/down** 为云→设备高频飞控/杆量，**drc/up** 为设备→云通道内状态/回传；**drc_mode_enter / drc_mode_exit** 在协议上走 **thing/.../services** + **services_reply**，不是 drc/* 子路径。

// ==================== Thing Topic ====================
//
// Thing Topic 用于设备物模型相关的消息通信，包括遥测数据上报、
// 云端服务下发、设备事件上报等。
// 基础路径格式: thing/product/{gateway_sn}/{channel}

// OsdTopic 返回设备遥测数据（OSD）上报 Topic。
// 路径格式: thing/product/{device_sn}/osd
// 方向: 设备 → 云平台
// 用途: 设备定期推送 DJI 物模型中 pushMode=0 的定频数据，通常包括飞行姿态、GPS 坐标、电池电量等实时属性。
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
// 用途: 设备在状态变化时推送 DJI 物模型中 pushMode=1 的状态数据，通常包括固件/硬件版本、设备能力集、机巢/负载状态等非定频属性。
// 本服务不使用 state 上报刷新在线状态，在线状态以有效 osd 上行为准。
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

// ==================== Organization / Requests（设备主动请求，组织见文档） ====================
// 大疆上云 [Requests 说明](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/organization.html)。
// 与 thing/.../services 云下发行不同；此处为**设备**经 requests 要数据/要能力，云经 requests_reply 回包（data 字段以协议与 method 为准）。

// RequestsTopicPattern 返回通配订阅 device→cloud 的 requests 上行主题。
// 路径格式: thing/product/+/requests
func RequestsTopicPattern() string {
	return "thing/product/+/requests"
}

// RequestsReplyTopic 返回 cloud→device 的 requests 应答主题。
// 路径格式: thing/product/{gateway_sn}/requests_reply
// 见协议中 requests_reply 报文体（与 ServiceReply 等常见同形，以官方示例为准）。
func RequestsReplyTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/requests_reply", gatewaySn)
}

// ==================== Property（物模型属性，Dock3） ====================
// [Properties 文档](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/properties.html) 与 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html) 中「属性设置」：
//   - **property/set**：仅 **云平台/本服务 → 目标设备**（`gateway_sn` 为机场/网关等），**Publish** 下发可写物模型键值对。
//   - **property/set_reply**：**设备 → 云平台**，设备对此次属性写入的执行结果；云平台侧用 **通配 +** **Subscribe** 接收（与发 set 的同一「云」身份）。

// PropertySetTopic 返回**云平台向设备**下发属性修改的 Topic（仅云→设备，勿反向理解）。
// 路径格式: thing/product/{gateway_sn}/property/set
// 由 SetProperty 使用 ServiceRequest+MethodPropertySet 发布，载荷为待写入属性，见 SetProperty。
func PropertySetTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/property/set", gatewaySn)
}

// PropertySetReplyTopicPattern 返回**设备对 property/set 的应答**通配（设备 → 云，云侧订阅）。
// 路径格式: thing/product/+/property/set_reply
func PropertySetReplyTopicPattern() string {
	return "thing/product/+/property/set_reply"
}

// ==================== Sys Topic（设备/系统级状态，见设备文档） ====================
//
// 大疆上云 [Status / 设备与上下线等](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html)。
// 与 thing 下物模型通道并列；在离线、拓扑等走 sys/product/.../status，云回 status_reply。
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

// ==================== DRC（指令飞行 / 实时控制） ====================
// DRC 使用独立于 services 的实时通道：drc/down 为云平台下发，drc/up 为设备回传。
// drc_mode_enter、drc_mode_exit、飞行控制权等仍走 services + services_reply。

// DrcUpTopic 返回设备经 drc/up 上报的 Topic。
// 路径格式: thing/product/{gateway_sn}/drc/up
func DrcUpTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/drc/up", gatewaySn)
}

// DrcUpTopicPattern 返回 drc/up 通配订阅模式。
// 路径格式: thing/product/+/drc/up
func DrcUpTopicPattern() string {
	return "thing/product/+/drc/up"
}

// DrcDownTopic 返回云平台经 drc/down 下发实时控制消息的 Topic。
// 路径格式: thing/product/{gateway_sn}/drc/down
func DrcDownTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/drc/down", gatewaySn)
}
