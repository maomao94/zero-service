# DJI 上云前置服务 -- 产品待办列表

> 本文件是所有需求的唯一入口。老板通过「需求输入」提交想法，AI 产品经理自动梳理并写入产品待办表。

<!--
=== 使用说明 ===

【老板（用户）- 需求输入方式】

老板是技术出身的项目开发者，需求输入支持两种方式：

方式一：快捷便签（适合简短想法）
- 在「老板便签」区域直接写，一句话、关键词均可
- 梳理后标记「已梳理 → B-XXX」

方式二：需求文档（适合复杂需求）
- 在项目目录下创建 `需求输入.md`，支持标准 Markdown 格式
- 可包含大段描述、接口定义、代码片段、架构图、外部链接等
- Sprint Planning 时 PM 自动读取并梳理
- 梳理完成后 PM 在文档底部追加处理记录

【老板（用户）- 补充文档】
- 老板可在需求输入中引用补充材料，PM 负责串联消化
- 支持的引用类型见「补充文档」区域

【AI - 需求梳理流程】
- Sprint Planning 时按顺序读取：① 需求输入.md ② 老板便签 ③ 补充文档引用
- 对模糊想法执行需求澄清，按以下维度结构化提问：
  · WHY — 业务目标、解决什么问题、不做的后果
  · WHO — 使用方（人/系统/设备）、上下游依赖
  · WHAT — 功能边界、输入输出、核心数据结构、非目标
  · HOW — 技术约束、协议选型、性能要求、已有可复用组件
- 信息不足以拆解时，向老板发起 3-5 个关键问题（按维度分组，附带选项/示例，一次问清减少迭代）
- 将需求结构化为 Epic → Story，写入产品待办表
- 便签原文标记「已梳理 → B-XXX」，需求文档底部追加处理记录

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

## 需求输入

> 复杂需求请使用项目目录下的 `需求输入.md` 文件，支持标准 Markdown，可包含接口定义、代码片段、架构图等。
> 简单想法可直接写在下方便签区。

### 老板便签

> 快捷输入区，一句话、关键词均可，PM 自动梳理。

- [x] received event: tid=e4889c0f-378b-47af-b18f-ec9472baea89 method=live_stop_push need_reply=0 gateway=  app=djigateway.rpc      caller=djisdk/client.go:103   trace=4a024174515dc42624642b09be0152b3  span=69556ab2f8eb628b   client=dji-gateway-001
规范钩子函数的打印日志 → 已梳理: B-008

### 补充文档

> 老板可在此引用与需求相关的补充材料，PM 在需求分析时自动串联消化。

| 类型 | 引用 | 说明 |
| --- | --- | --- |
| 接口定义 | `app/djigateway/djigateway.proto` | DJI 上云 gRPC 接口定义 |
| 外部文档 | https://developer.dji.com/doc/cloud-api-tutorial/cn/ | DJI Cloud API 官方文档 |

---

## 路线图

> 从开发计划（架构设计文档）中提取模块依赖关系，规划迭代顺序。

### 依赖分析

| 层级 | 模块 | 依赖 | 说明 |
| --- | --- | --- | --- |
| 地基模块 | {模块名} | 无 | {基础设施，优先开发} |
| 核心业务 | {模块名} | {依赖的地基模块} | {核心功能，第二优先级} |
| 锦上添花 | {模块名} | {依赖的核心模块} | {增强功能，最后迭代} |

### 里程碑规划

| 里程碑 | 目标 | 关键交付物 | 状态 |
| --- | --- | --- | --- |
| Milestone 1 MVP | {最小可用版本} | {交付物列表} | 未开始 |
| Milestone 2 完整版 | {功能完整} | {交付物列表} | 未开始 |
| Milestone 3 增强版 | {体验优化与扩展} | {交付物列表} | 未开始 |

---

## 产品待办

| 编号 | 优先级 | 需求描述 | 来源 | 状态 | Sprint | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| B-014 | Must | gRPC 错误码优化 — 引入 DJIErrorCode 枚举，SendCommand 返回设备错误时在 CommonRes 中携带原始错误码 + 枚举中文描述，调用方可感知具体 DJI 错误 | 需求输入.md | 开发中 | S7 | SDK 层改造 SendCommand + errorcode 包 |
| B-015 | Must | 机巢在线状态管理 — SDK 订阅 sys/product/+/status 主题 + onStatus 钩子；业务层用 go-zero cache 标记机巢在线（Status 主判断 + OSD 心跳辅助超时）；下发命令前校验在线状态，离线快速拒绝 | 需求输入.md | 开发中 | S7 | Status 主题 + OSD 辅助心跳 + 在线拦截 |

---

## 已完成条目归档

> 已完成的 Backlog 条目归档至此，保持上方待办表简洁。

| 编号 | 需求描述 | 完成 Sprint | 完成日期 |
| --- | --- | --- | --- |
| B-001 | 接入大疆 DRC 指令飞行控制（drc_mode_enter / drc_mode_exit / drone_control），在 djigateway.proto 新增 11 个 gRPC 接口，Logic 层全部实现 | S3 | - |
| B-002 | 补 HMS 健康告警上报钩子（protocol.go 新增 HmsEventData 结构体，client.go 新增类型化钩子 + dispatchTypedEvent 分支，hooks/hms_event.go 实现） | S3 | - |
| B-003 | 补设备属性上报钩子（OSD / State 上报，HandleOsd / HandleState，SubscribeAll 增加订阅，hooks/osd.go + state.go 实现） | S3 | - |
| B-004 | 敏感接口配置拦截（DroneEmergencyStop 默认关闭，DangerousOpsConfig 配置结构 + Logic 层拦截检查） | S3 | - |
| B-005 | 接口归类修正（按上云 API 官方分类，return_home 系列归入航线管理，DRC 按子分组拆分） | S3 | - |
| B-006 | 修复 MQTT 消息分发 topicTemplate 匹配错误：`topicTemplateFromMsg` 返回完整主题而非订阅通配模板，导致 dispatcher 匹配失败，所有通配订阅消息被静默丢弃 | S4 | - |
| B-007 | PSDK 透传接口规范化：按 DJI Cloud API 官方文档规范 gRPC 接口命名与注释，proto/Logic/SDK 方法全面规范化，Logic 层新增 value 长度校验 | S5 | - |
| B-008 | 钩子日志规范化：SDK 层 HandleEvents 移除空值 gateway 字段，业务钩子层 device=/gateway= 统一为 sn=，7 个钩子文件全部对齐 | S5 | - |
| B-009 | Dock3 全量 gRPC 接口暴露 — 远程调试（15个 RPC） | S6 | 2026-04-24 |
| B-010 | Dock3 全量 gRPC 接口暴露 — 相机/云台控制（6个 RPC） | S6 | 2026-04-24 |
| B-011 | Dock3 全量 gRPC 接口暴露 — 直播管理（3个 RPC） | S6 | 2026-04-24 |
| B-012 | Dock3 全量 gRPC 接口暴露 — 航线管理补充（4个 RPC） | S6 | 2026-04-24 |
| B-013 | Dock3 全量 gRPC 接口暴露 — 通用属性设置（1个 RPC：SetProperty） | S6 | 2026-04-24 |
