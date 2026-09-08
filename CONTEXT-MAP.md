# 领域上下文地图

zero-service 包含多个相对独立的业务上下文。本地图只帮助定位领域语言；实现约束与任务入口见 [Agent 知识库](./docs/agent/README.md)。

## 上下文

- [ISP](./common/isp/CONTEXT.md)：定义 ISP 连接、对端身份与设备点位语言
- [Scheduling](./common/crontask/CONTEXT.md)：定义通用任务调度及其时间语义
- [Trigger](./app/trigger/CONTEXT.md)：定义层级计划与独立周期任务
- [Live](./app/live/CONTEXT.md)：定义会议与 LiveKit 媒体资源之间的身份
- [Oryx Relay](./app/oryxserver/CONTEXT.md)：定义外部拉流到 SRS/Oryx 目标的中继活动

## 关系

- **ISP -> Scheduling**：ISP 巡检任务使用 Scheduling 的周期与执行时间语言。
- **Trigger -> Scheduling**：Trigger 的 Cron Job 使用 Scheduling 的通用调度语义。
- **Live / Oryx Relay**：两者都是媒体相关上下文，但会议身份与中继身份彼此独立。
