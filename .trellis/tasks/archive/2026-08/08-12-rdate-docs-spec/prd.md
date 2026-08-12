# Document unified exact-time scheduling

## Goal

为 Trigger 对接方和后续维护者提供一套与已实现行为一致的 Plan/CronJob RRULE Set 指南，明确如何用周期规则、指定时间和排除条件构造最终执行候选。

## Background

- `specified_times` 将精确到秒的候选加入 RRULE Set，内部表示为 `RDATE`。
- `excluded_times` 从合并候选中排除精确到秒的时间，内部表示为 `EXDATE`。
- `exclude_dates` 排除整个上海自然日，包括 RRULE 和 `specified_times` 产生的候选。
- Plan 创建时预展开最终 Set，显式范围最多 3 年；CronJob 持久化完整 Set 并按需推进，显式范围最多 100 年。
- 原 CronJob 场景指南仍将若干精确时间组合描述为必须拆分 CronJob，已与新能力不符。

## Requirements

- 将现有 CronJob 场景指南升级并重命名为 Plan/CronJob 共用的 RRULE API 场景指南，更新 `docs/README.md`、`docs/trigger.md` 及仓库内相关链接。
- 文档说明统一集合语义：`(RRULE union specified_times) - excluded_times - expanded(exclude_dates)`，排除优先且同一秒去重。
- 说明两个精确时间字段均使用 `yyyy-MM-dd HH:mm:ss`、`Asia/Shanghai`、闭区间范围校验，每个列表最多 1000 项。
- 提供 Plan 的 `CalcPlanTaskDate`、`CreatePlanTask` 示例，以及 CronJob 的 Create、Update/Submit 清空、Get/List 回显和 Preview 示例。
- 明确 Plan 预展开、过去时间过滤、5000 执行项上限和 3 年范围；明确 CronJob 完整 Set 持久化、首次 `next_run`、完成推进、预览、在途更新拒绝和 100 年范围。
- 修正旧指南中“09:00 与 17:30 必须拆成两个 CronJob”等已过时结论：RRULE 本身仍按字段笛卡尔积，但可通过 `specified_times` 精确加入不规则时间；多个独立业务 Handler/身份仍需拆任务。
- 更新 `.trellis/spec/backend/trigger-guidelines.md`、`crontask-guidelines.md`、`contract-generation.md` 和 `gormx-guidelines.md`，记录共享编译、Set 权威来源、Proto JSON 字段、可空 JSON 列、替换/清空和在途原子更新契约。

## Acceptance Criteria

- [ ] 共用指南能分别构造 Plan 和 CronJob 的多个不规则指定时间任务。
- [ ] 指南准确描述 RDATE/EXDATE、整日排除、排除优先、同秒去重、上海时区和闭区间边界。
- [ ] Plan 示例与 3 年、过去过滤、5000 项及预展开语义一致。
- [ ] CronJob 示例与持久化回显、列表清空、首次时间、推进、预览和在途更新语义一致。
- [ ] 旧文件名和旧标题的仓库内链接全部更新，不遗留与新能力冲突的限制说明。
- [ ] 所有 JSON 示例可解析，Markdown 相对链接存在，不包含模板占位文本，`git diff --check` 通过。

## Out Of Scope

- 不修改 Proto、Go 实现、数据库模型或生成文件。
- 不新增时间范围排除、动态节假日替代日、无 RRULE 的纯 RDATE 请求或跨 CronJob 去重。
- 不承诺 Exactly Once；业务消费者仍需按稳定业务键幂等。

## Dependency

- `08-12-rdate-contract-compiler` 和 `08-12-rdate-plan-integration` 已完成。
- `08-12-rdate-cronjob-integration` 必须完成独立检查、提交和归档后，本任务才能启动。

## Technical Notes

- 文档中的 JSON 字段遵循现有 gRPC JSON 示例风格；字段事实以 `app/trigger/trigger.proto` 和最终实现为准。
- 重命名目标为 `docs/trigger-rrule-api-guide.md`，以反映 Plan/CronJob 共用范围。
