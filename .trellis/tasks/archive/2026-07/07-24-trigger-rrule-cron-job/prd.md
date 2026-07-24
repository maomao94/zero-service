# Trigger RRULE Cron Job

## Goal

在 Trigger 中新增基于 `common/crontask` 的独立 RRULE 周期任务能力。任务到点后调用 Eventstream；普通错误依赖 lease 过期自动重试，业务任务不存在时删除当前 Cron Job，避免继续触发。

## Requirements

- Cron Job 独立于现有 Plan/PlanBatch/PlanExecItem，不预生成日期批次或执行项。
- `common/crontask.TaskConfig` 只保留通用调度字段；不加入 Trigger 业务字段，并删除未参与 CAS 的 `Version`。
- Trigger 根据 `PlanRulePb`、开始/结束时间和排除日期生成 RFC 5545 RRULE 与首次执行时间。
- 创建请求包含任务编码、任务名称、优先级、单次调度锁超时、Payload、Extra、机构号、类型、分组、描述、规则、开始/结束时间、排除日期、首次时间过滤开关及五个扩展字段。
- 单次调度锁超时使用毫秒；任务配置值大于 0 时覆盖 Scheduler 默认锁超时，未指定或为 0 时使用默认值。Trigger 创建接口接受该字段；ISPAgent Store 应用已持久化值，但 `101-1` 下发报文不包含也不修改该字段。
- 创建后返回 Trigger 生成的 `JobId` 和首次 `NextRun`；`TaskCode` 仍为调用方提供的全局唯一业务编码。
- 支持按 `JobId` 启用、禁用和软删除。创建默认启用；重新启用从当前时间计算未来执行时间，不追赶禁用期间的历史周期。
- Trigger GORM 模型保存 crontask 通用字段，并将 Trigger 业务字段平铺为列；业务字段同时通过 `CronJobExtra` 重建到 `TaskConfig.Extra`。
- `skipTimeFilter` 只影响创建时首次执行时间，不持久化。
- 到点调用 Eventstream 独立回调接口。成功推进 RRULE；RPC 错误或未知回执自动重试；任务不存在回执删除当前 Cron Job。
- 调度语义为 At-Least-Once，不承诺 Exactly-Once；同一 claim 只有一个实例获得执行权。
- 所有新增导出 Go 类型、关键分支、RPC、Proto 消息、枚举和字段必须有准确中文注释。

## Acceptance Criteria

- [x] `TaskConfig.Version` 及其无效自增、Trace 和测试依赖已删除，其他 Plan 模型的版本字段不受影响。
- [x] claim 使用 `next_run` lease token，完成更新使用 CAS；普通回调错误不写 `LastRun/NextRun`。
- [x] `ErrDeleteTask` 直接或包装返回时执行软删除，不再推进 `NextRun`；删除失败保留重试能力。
- [x] 创建相同 `TaskCode` 返回重复错误；成功创建返回唯一 `JobId` 和正确首次执行时间。
- [x] start/end、excludeDates、规则耗尽和 `skipTimeFilter` 场景计算正确；无下次调度保存为 SQL `NULL`。
- [x] 禁用任务不会被扫描；启用任务从当前时间恢复；删除接口幂等。
- [x] Eventstream SUCCESS、UNKNOWN/RPC error、TASK_NOT_FOUND 三种路径行为符合约定，重试时 `scheduledTime` 保持原计划时间。
- [x] Trigger 与 Eventstream Proto 已通过各自 `gen.sh` 生成，生成代码没有手工编辑。
- [x] Trigger 创建接口可配置任务级锁超时；ISP Store 可持久化并应用任务值，但 `101-1` 下发报文不包含且不得覆盖该字段；0 使用 Scheduler 默认值。
- [x] 相关单测、race、定向 vet、全仓构建与 `git diff --check` 通过。

## Out Of Scope

- 不新增 Plan/Batch/ExecItem 或执行日志模型。
- 不支持 Eventstream 回执控制启用或禁用；生命周期只由 Trigger RPC 管理。
- 不提供 Exactly-Once、历史周期逐次补偿或分布式事务保证。
- 本期不新增 Cron Job 查询、修改规则或立即执行 RPC。
