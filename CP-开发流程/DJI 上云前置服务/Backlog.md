# DJI 上云前置服务 -- 产品待办列表

> 老板在便签区随意写想法，AI 产品经理自动梳理并写入产品待办表。

<!--
=== 使用说明 ===

【老板（用户）- 老板便签】
- 在「老板便签」区域随意写下想法，格式不限
- 一句话、关键词、甚至画个箭头都行，AI 产品经理会自动梳理
- 梳理后的条目会标记「已梳理 → B-XXX」，关联到产品待办表

【老板（用户）- 产品待办】
- 在「产品待办」表格中追加新需求，只需填写「需求描述」列，其余列可留空
- 格式不限，简要描述即可，AI 会自动润色补充
- 可随时追加新需求，无需关注排序

【AI - 老板便签梳理】
- Sprint Planning 时读取「老板便签」中的未梳理条目
- 对模糊想法执行苏格拉底式需求澄清（WHY/WHO/WHAT/HOW 四维度提问）
- 将澄清后的想法结构化为 Epic → Story，写入产品待办表
- 在便签原文后追加 → 已梳理: B-XXX，标记复选框为 [x]
- 信息不足以拆解时，向老板发起 1-2 个关键问题（附带选项/示例）

【AI - 产品待办管理】
- 读取新增条目后，润色需求描述，补充技术细节
- 使用 MoSCoW 方法评估并填写优先级（Must/Should/Could/Won't）
- 分配编号（B-001 递增），标注来源和状态
- 纳入 Sprint 时更新状态为「开发中」，完成后更新为「已完成」
- 已完成的条目定期移至「已完成条目归档」区域，保持待办表简洁

【优先级说明 - MoSCoW】
- Must:   必须做，缺少则项目无法交付
- Should: 应该做，重要但非阻塞
- Could:  可以做，锦上添花
- Won't:  暂不做，明确排除

【状态流转】
- 待开发 → 开发中 → 已完成
- 待开发 → 已搁置（附注原因）
-->

---

## 老板便签

> 老板的想法随便写，格式不限，AI 产品经理会自动梳理。

- [x] 解决hms 日志没有打印的问题，我用 mqttx工具订阅了，hms 一直在上班，但是日志没有看到 → 已梳理: B-006
- [] https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/psdk-transmit-custom-data.html 规范 Topic: thing/product/{gateway_sn}/events

Direction: up

Method: custom_data_transmission_from_psdk
你这个是特定的 psdk 接收，这个需要加一个钩子
下面还有一个发送特定的 psdk 接口  是平台发送给机巢的，grpc 定义的接口描述不行，规范接口，重新写功能

---

## 产品待办

| 编号 | 优先级 | 需求描述 | 来源 | 状态 | Sprint | 备注 |
| --- | --- | --- | --- | --- | --- | --- |

---

## 已完成条目归档

> 已完成的 Backlog 条目归档至此，保持上方待办表简洁。

| 编号 | 需求描述 | 完成 Sprint | 完成日期 |
| --- | --- | --- | --- |
| B-001 | 接入大疆 DRC 指令飞行控制（drc_mode_enter / drc_mode_exit / drone_control），在 djigateway.proto 新增 11 个 gRPC 接口，Logic 层全部实现 | S03 | - |
| B-002 | 补 HMS 健康告警上报钩子（protocol.go 新增 HmsEventData 结构体，client.go 新增类型化钩子 + dispatchTypedEvent 分支，hooks/hms_event.go 实现） | S03 | - |
| B-003 | 补设备属性上报钩子（OSD / State 上报，HandleOsd / HandleState，SubscribeAll 增加订阅，hooks/osd.go + state.go 实现） | S03 | - |
| B-004 | 敏感接口配置拦截（DroneEmergencyStop 默认关闭，DangerousOpsConfig 配置结构 + Logic 层拦截检查） | S03 | - |
| B-005 | 接口归类修正（按上云 API 官方分类，return_home 系列归入航线管理，DRC 按子分组拆分） | S03 | - |
| B-006 | 修复 MQTT 消息分发 topicTemplate 匹配错误：`topicTemplateFromMsg` 返回完整主题而非订阅通配模板，导致 dispatcher 匹配失败，所有通配订阅消息被静默丢弃 | S04 | - |
