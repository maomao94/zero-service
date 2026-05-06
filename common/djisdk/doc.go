// Package djisdk 封装大疆上云（Dock/设备与 MQTT）协议能力，提供 Topic 构造、云侧 Client、消息结构与常用载荷模型。
//
// 官方资料（实现与注释以此为准，版本升级时请逐条 diff）：
//   - [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)：各 topic 方向以表中「云/设备」为准；本 SDK 按云平台侧客户端建模
//   - [Status | device](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html)：sys/product/.../status、status_reply；对应 StatusTopic*、StatusHandler、HandleStatus、OnStatus
//   - [Requests | organization](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/organization.html)：thing/product/.../requests、requests_reply；对应 RequestsTopic*、RequestHandler、HandleRequests、OnRequest
//   - [Properties 上云](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/properties.html)：云经 property/set 向设备写入可写属性，设备经 property/set_reply 回执；不要把它理解成设备向云写属性
//   - [DRC | Dock3](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html)：drc/down 为云→设备实时控制通道，drc/up 为设备→云状态与回执通道；drc_mode_enter、drc_mode_exit 等仍走 thing/.../services + services_reply
//   - DRC 杆量/虚拟摇杆 method 为 stick_control，载荷见 DrcStickControlData；DRC 避障、时延、OSD 等上报多经 drc/up，载荷见 protocol_drc.go
//   - 其余 topic（services、events、osd、state 等）见 topic.go 与 Client
//
// 行为约定：
//   - 简单 result 码：0 表示成功；1 表示云侧未实现/内部错误；2 按大疆常见约定表示超时，禁止挪作非超时占位
//   - events：通知型 method 优先由 tryDispatchEventNotify 分发到强类型 On* handler；未命中预置分支时才进入 OnEvent 兜底
//   - status：设备上报 sys/product/.../status 后，云平台按 ReplyOptions 决定是否回 status_reply，result 由 StatusHandler 返回
//   - requests：设备通过 thing/product/.../requests 主动向云拉取平台侧数据，云通过 requests_reply 返回 result 与 data.output
//   - property：云平台向 gateway_sn 标识的目标设备 Publish property/set，设备 Publish property/set_reply 回云；只读物模型上报仍走 osd/state
//   - drc：drc/down 与 drc/up 是实时控制专用通道；进入/退出 DRC 模式、飞行控制权等服务调用走 services/services_reply，二者不要混用
package djisdk
