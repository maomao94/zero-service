# Add Trigger RRULE specified execution dates

## Goal

先为 Trigger 的 Plan 与 CronJob 两种 RRULE 模式统一增加精确时间点的纳入与排除能力，再更新多场景 API 指南，支持 RRULE、RDATE 和 EXDATE 的完整集合调度。

## Background

- Plan 模式由 `CalcPlanTaskDate` / `CreatePlanTask` 使用 `PlanRulePb`，创建时预展开 Plan、Batch 和 ExecItem，显式时间范围最多 3 年。
- CronJob 模式由 `CreateCronJob` / `SubmitCronJob` 使用同一个 `PlanRulePb`，不预展开执行数据，按需计算下一计划点，显式时间范围最多 100 年。
- 两种模式共用 `freq/month/day/week/hours/minutes`、`start_time/end_time` 和 `exclude_dates` 的规则语义。
- `rrule-go.Set` 已支持 `RDATE`，并按 `RRULE ∪ RDATE - EXDATE` 计算候选；`common/crontask.NextAfter`、预览和描述均消费完整 Set，Plan 的日期展开也消费 Set。
- `rrule-go.Set` 不会使用 RRULE 的 `DTSTART/UNTIL` 裁剪 `RDATE`；Trigger 必须在编译边界显式校验每个指定时间均落在规范化后的 `start_time/end_time` 闭区间内。
- 当前 Trigger 请求没有 RDATE 字段，`CompileSchedule` / `CompileCronJobSchedule` 只写入 RRULE 和 EXDATE。
- 现有 `docs/trigger-rrule-api-guide.md` 已包含每分钟、每 10 分钟、每小时、固定季度月份、每月、每周、固定单次、节日、补触发、预览和人工执行等场景，但文档入口和示例外层主要面向 CronJob。

## Requirements

- 在 `CalcPlanTaskDateReq`、`CreatePlanTaskReq`、`CreateCronJobReq`、`UpdateCronJobReq` 和 `SubmitCronJobReq` 统一增加：
  - `repeated string specified_times`：额外纳入的精确执行时间，内部编译为 RFC 5545 `RDATE`。
  - `repeated string excluded_times`：额外排除的精确执行时间，内部编译为 RFC 5545 `EXDATE`。
- 两个字段元素格式均为 `yyyy-MM-dd HH:mm:ss`，按 `Asia/Shanghai` 解析；每个列表最多 1000 项。CronJob 详情/管理视图同步返回原始配置。
- 保留现有 `exclude_dates` 作为整日排除便捷接口；它会排除当天所有 RRULE 和 `specified_times` 候选。
- 最终候选集合为 `RRULE ∪ specified_times - excluded_times - expanded(exclude_dates)`；相同时间只产生一个候选。
- 每个 `specified_times` 和 `excluded_times` 值必须满足 `start_time <= value <= end_time`；边界本身允许命中，区间外时间返回参数错误，不依赖 `rrule-go` 的宽松集合行为。
- `excluded_times` 是精确时间点列表，不表示时间范围；本任务不新增区间起止排除。
- Trigger CronJob 持久化新增指定时间与精确排除时间列表字段，并纳入 Create/Update/Submit 配置所有权和清空语义；Plan 保存完整 `rrule_str` 快照并按现有模型决定是否需要单独管理视图字段。
- 保持 Plan 最大 3 年、CronJob 最大 100 年；明确校验指定时间与规则范围的关系。
- 修改 `trigger.proto` 后运行 `app/trigger/gen.sh`，不手工编辑生成文件。
- 为 Set 编译、Plan 日期计算/创建、CronJob Create/Update/Submit、预览和执行推进补测试。
- 功能完成后，将 `docs/trigger-rrule-api-guide.md` 调整为同时面向 Plan 和 CronJob 的 Trigger RRULE 场景指南；同步所有仓库内相对链接。
- 在文档开头明确两种 RRULE 模式的 API、执行模型、持久化方式和最大时间跨度。
- `PlanRulePb` 字段、时间区间、排除日期和周期场景只维护一套规则说明，不复制两份相同规则 JSON。
- 为共享规则示例说明如何嵌入两种请求：
  - Plan：`CalcPlanTaskDateReq` 用于预览，`CreatePlanTaskReq` 还必须提供 `plan_id`、`plan_name`、`type`、`dept_code` 和至少一个 `exec_items`。
  - CronJob：`CreateCronJobReq` / `SubmitCronJobReq` 提供任务身份、调度参数和 payload。
- 保留并复核以下场景：每分钟、每 10 分钟、每小时整点、固定季度月份 09:00、固定时间一次、每月 1 日、每周一三五 09:00，以及现有其他受支持场景。
- 明确“每三个月”仍是固定月份集合，例如 `[1,4,7,10]`，不是从任意起点滚动的 `INTERVAL=3`。
- CronJob 专属章节继续保留分组、Update/Submit、创建后补触发、预览、RunNow、lease 和执行幂等语义。
- Plan 专属章节说明预展开、`exec_items`、3 年限制及 Plan/Batch/ExecItem 生命周期，不把 CronJob 的 group/lease/RunNow 语义套到 Plan。
- 更新 `docs/trigger.md` 和 `docs/README.md` 的文档名称、摘要和链接，使入口明确覆盖两种 RRULE 模式。
- 不修改 Proto、调度实现、数据库模型或生成代码。
- 文档阶段统一解释周期规则与多个指定时间的并集，并提供 Plan/CronJob 两种请求示例。

## Acceptance Criteria

- [ ] 对接方能从统一指南判断应选择 Plan 还是 CronJob。
- [ ] 两种 RRULE 模式都能接收 `specified_times` 和 `excluded_times`，并生成包含 RDATE/EXDATE 的完整 Set。
- [ ] RRULE、指定时间、精确排除时间和整日排除的并集/排除/去重行为有可观察测试。
- [ ] 两类精确时间均使用 `yyyy-MM-dd HH:mm:ss` 和 `Asia/Shanghai`；等于开始/结束边界合法，区间外或格式错误返回参数错误。
- [ ] 两个列表都允许为空、各最多 1000 项；第 1001 项在请求校验阶段被拒绝。
- [ ] `excluded_times` 只排除同一秒；`exclude_dates` 排除当天所有候选。
- [ ] `exclude_dates` 能排除当天任意 RDATE，而不只排除规则中 `hours × minutes` 对应的时间。
- [ ] Plan 计算和创建会展开指定时间，CronJob 创建、推进与预览会返回指定时间候选。
- [ ] CronJob Update/Submit 可替换或清空指定时间列表，且遵循现有在途更新保护。
- [ ] Proto、生成物、模型、转换、Logic、测试和文档字段一致。
- [ ] 所有周期规则示例只维护一套 `PlanRulePb` 语义，并明确可用于两种模式。
- [ ] 文档包含两种请求外层的最小合法结构，Plan 示例不遗漏 `exec_items`，CronJob 示例不遗漏必填身份字段。
- [ ] Plan 的 3 年预展开限制与 CronJob 的 100 年按需调度限制描述准确。
- [ ] 用户指定的每分钟、每 10 分钟、固定季度月份、固定单次、每月 1 日和每周一三五场景均可直接定位。
- [ ] CronJob 专属的分组、更新、补触发、预览和人工执行内容没有被误写成 Plan 通用能力。
- [ ] `docs/README.md`、`docs/trigger.md` 和指南之间的相对链接有效，不存在旧标题或死链。
- [ ] JSON 示例可解析，文档无 TBD，`git diff --check` 通过。

## Out Of Scope

- 不改变 Plan 或 CronJob 除候选集合增加 RDATE 外的状态机、lease 和回执语义。
- 不新增时间范围排除、多 RRULE、`INTERVAL`、`COUNT` 或组级原子操作。
- 不重写 Plan 状态机、CronJob lease 或部署章节。

## Delivery Order

1. 公共请求契约与 RRULE Set 编译。
2. Plan 计算、创建与日期展开。
3. CronJob 持久化、管理和运行链路。
4. 统一场景文档与 Trellis Spec。


## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
