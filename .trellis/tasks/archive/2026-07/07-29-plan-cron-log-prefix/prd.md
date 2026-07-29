# 统一 Plan Cron 日志前缀

## Goal

为 Trigger 的 Plan/Batch/ExecItem 定时扫表链路增加统一、可直接检索的正文标记 `[cron-plan]`，便于在纯文本日志中与通用 `crontask` 及未来其他 cron 业务区分。

## Requirements

- `app/trigger/cron` 直接产生的生命周期、扫表、下游调用、结果处理和收尾日志，正文必须以 `[cron-plan] ` 开头。
- 经 `planscope` 输出的 cron 日志也必须使用同一正文前缀，包括 `execdelay.LogWarnings` 在 cron 调用路径产生的日志。
- gRPC 主动操作和 gRPC callback 日志不增加 `[cron-plan]`；`RunPlanExecItem` 的“立即执行”日志保持原样。
- `common/crontask` 已有的 `[crontask]` 前缀保持不变，二者不得合并或重复标记。
- 保留现有结构化字段、日志级别和业务语义，只调整日志正文格式。

## Acceptance Criteria

- [x] `app/trigger/cron` 的日志正文统一以 `[cron-plan] ` 开头。
- [x] cron 路径的延期/进行中告警带 `[cron-plan]`，callback 路径的相同告警不带该前缀。
- [x] `RunPlanExecItem`、`CallbackPlanExecItem` 和 `common/crontask` 的现有正文标记不受影响。
- [x] 相关单元测试通过，并通过格式检查与变更范围检查。

## Out Of Scope

- 调整 Plan/Batch/ExecItem 状态机、数据库更新或并发控制。
- 修改日志结构化字段 `entry`、`tag`、`ref` 等既有契约。
- 修改 GORM 日志级别、编码格式或 SQL 输出策略。

## Notes

- 本任务为轻量日志格式改动，采用 PRD-only 流程。
