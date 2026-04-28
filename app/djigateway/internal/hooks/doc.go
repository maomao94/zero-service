// Package hooks 将 DJI 上云 MQTT 上行按 Topic 类拆分，避免「事件 / 遥测 / 系统」混在同一文件里难读。
//
// 划分与 DJI [Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html) 一致：
//
//   - 事件类（thing/.../events，method 分流，部分需 events_reply 由 djisdk 回包）：
//     见 event_flighttask_up.go、event_notify_up.go
//   - 遥测/物模型状态类（thing/.../osd、thing/.../state，高频/元信息，无云侧 *reply）：
//     见 telemetry_up.go
//   - 系统状态类（sys/.../status，在离线/拓扑，需 status_reply）：
//     见 sys_status_up.go
//   - 设备主动请求（thing/.../requests → requests_reply，需 OnRequest）：
//     见 mqtt_request_up.go
//   - 物模型**属性**（**云→设备** 写可写项：thing/.../property/set，设备→云 set_reply；**不是**设备向云「设置」属性；与 state/osd 只读上报不同）：
//     应答与 services_reply 同形，由 djisdk SetProperty 经 pending 收 set_reply，无应用层独立 hook（与 SendCommand 一致）
//   - **DRC**（[Topic 总览](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)：**drc/down** 云发杆量/心跳等、**drc/up** 避障/回传等以 [DRC 上云](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html) 分节为准；**drc_mode_enter** 等走 **services** / **services_reply**）：
//     杆量/虚拟摇杆 (stick_control) 的实时下行由 gRPC **SendDrcStickControl** 经 djisdk 发 drc/down；[避障信息上报](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html#drc-%E9%81%BF%E9%9A%9C%E4%BF%A1%E6%81%AF%E4%B8%8A%E6%8A%A5) 等不在该 RPC
//
// 向 djisdk.Client 注册时优先使用本包 RegisterDjiClient，见 register.go。

package hooks
