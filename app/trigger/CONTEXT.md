# Trigger Context

Trigger 上下文包含按业务规则展开的层级计划，以及不展开层级的独立周期任务。两者共享调度能力，但不是同一种业务对象。

## Language

**Plan**:
按业务规则生成批次与执行项的层级计划。
_Avoid_: Cron Job

**Plan Batch**:
Plan 在某个计划触发点下形成的一组执行项。
_Avoid_: Plan, Execution Item

**Plan Execution Item**:
Plan 中最小的可调度、回调和状态流转单位。
_Avoid_: Plan Batch, Cron Job

**Cron Job**:
独立的 RRULE 周期任务，不展开为 Plan Batch 或 Plan Execution Item。
_Avoid_: Plan

**Job ID**:
Trigger 为 Cron Job 生成的内部身份。
_Avoid_: Task Code
