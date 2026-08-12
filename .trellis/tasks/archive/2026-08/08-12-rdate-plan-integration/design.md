# Plan Exact-Time Integration Design

## Scope

本子任务仅把公共 `specified_times` / `excluded_times` 编译能力接入 Plan 预览与创建，不修改 CronJob 持久化或文档。

## Data Flow

`CalcPlanTaskDateReq` 将规则、范围、整日排除和两个精确时间列表传给 `CompileSchedule`。响应中的 `plan_dates`、`schedule_description` 和 `rrule_str` 来自同一个最终 Set。

`CreatePlanTaskReq` 将两个列表透传给 Calc。Create 继续消费最终 `plan_dates` 创建 Batch/ExecItem，并将完整 Set 保存到 `plan.rrule_str`。

## Constraints

Plan 仍限制 3 年。`skip_time_filter=false` 时过去的 RRULE/RDATE 候选一并过滤；`true` 时保留。最终日期数量继续受 `5000 / len(exec_items)` 限制。

## Tests

覆盖预览并集、精确/整日排除、去重、创建展开、过去时间过滤和数量上限。
