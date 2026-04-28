// Package djisdk 封装大疆上云（Dock/设备与 MQTT）侧能力：Topic 表、Client 与消息体。
//
// 官方资料（实现与注释以此为准，版本升级时请逐条 diff）：
//   - [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)（各 topic 方向以表中「云/设备」为准；本 SDK 为**云平台侧**客户端）
//   - 设备与系统状态 [Status | device](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html)：sys/product/.../status、status_reply；在代码中见 StatusTopic*、StatusHandler、HandleStatus、OnStatus
//   - 组织与设备主动请求 [Requests | organization](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/organization.html)：thing/product/.../requests、requests_reply；在代码中见 RequestsTopic*、RequestHandler、HandleRequests、OnRequest
//   - 物模型**属性** [Properties 上云](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/properties.html)：**仅** 云经 property/set 向**设备**下写可写属性，设备经 property/set_reply 回执；**不是**设备向云写属性
//   - 指令飞行 **DRC** [DRC | Dock3](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html) 与 [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)：
//     **drc/down** 云→设备（杆量用 SendDrcStickControl；同 topic 另有 heart_beat 等，见 DRC 文档分节），**drc/up** 设备→云；**drc_mode_enter** / **drc_mode_exit** 等为 **thing/.../services** 的 method，**services_reply** 应答，与 **drc/*** 的 topic 对是两套路径
//   - DRC **避障信息上报**（多经 **drc/up**，设备→云，非本 gRPC 杆量 RPC；与杆量分节不同）: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html#drc-%E9%81%BF%E9%9A%9C%E4%BF%A1%E6%81%AF%E4%B8%8A%E6%8A%A5
//   - DRC **杆量/虚拟摇杆** (MQTT method **stick_control**，载荷如 DrcStickControlData): https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html#drc-%E6%9D%86%E9%87%8F%E6%8E%A7%E5%88%B6
//   - 其余 topic（services、events、osd、state 等）见本包 topic.go 与 Client
//
// 行为约定（简要）：
//   - 简单 result 码：0=成功；1=云侧错误/未实现等；**2=与大疆约定对齐时表示超时**（见 protocol.PlatformResult*，勿将 2 挪作非超时含义）
//   - **通知型 events**：常见由 tryDispatchEventNotify 对应强类型 On* 解析，need_reply 见设备侧；与「兜底」OnEvent 关系见 client.go
//   - **status**：设备上云后平台须按文档回 status_reply，result 由 StatusHandler 决定
//   - **requests**：设备主动向云要能力/数据，云经 requests_reply 回 result 与 data.output（体例与协议一致，常用 ServiceReply 同形 Envelope）
//   - **property**：**云平台/本服务** 向 `gateway_sn` 标识的**设备/机场** **Publish** `property/set`；设备 **Publish** `set_reply` 到云、云通配**订阅**收。与**只读**物模型上报（如 state 里部分字段）路径不同
//   - **drc**：`drc/down` 为云→设备（杆量/心跳等分节见 [DRC 上云](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html)）；`drc/up` 为设备→云。进入/退出 DRC 走 `services` / `services_reply`，与 **drc/*** 的 topic 对区分
//   - DRC 杆量 (stick_control) 以官方「DRC-杆量控制」为准: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html#drc-%E6%9D%86%E9%87%8F%E6%8E%A7%E5%88%B6
//   - DRC **避障信息上报** 与 drc/up 以官方「DRC-避障信息上报」为准: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html#drc-%E9%81%BF%E9%9A%9C%E4%BF%A1%E6%81%AF%E4%B8%8A%E6%8A%A5
package djisdk
