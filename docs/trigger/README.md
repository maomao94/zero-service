# Trigger 服务

基于 go-zero 的统一任务调度服务，提供 asynq 异步任务、Plan 计划任务和 RRULE CronJob 三类能力。

## 文档

| 文档 | 内容 |
| --- | --- |
| [Trigger 服务](./trigger.md) | 异步任务、计划任务、RRULE CronJob、API 和状态流转 |
| [规则与字段参考](./trigger-rrule-syntax.md) | 候选集合编译、`PlanRulePb` 字段、时间边界、更新限制与不支持的调度语义 |
| [场景示例](./trigger-rrule-api-guide.md) | 分钟级、日历、节假日、区间、一次性等可直接复制的请求示例 |
| [运行与对接](./trigger-rrule-operations.md) | 创建后补触发、预览计划时间、人工执行与上线检查清单 |
