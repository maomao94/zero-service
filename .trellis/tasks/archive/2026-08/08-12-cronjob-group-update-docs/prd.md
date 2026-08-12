# 完善 CronJob 分组更新与文档

## Goal

完善 Trigger CronJob 的分组身份、配置更新边界、长期规则范围和执行日志，使多个单规则 CronJob 可通过稳定 `group_id` 组织，并让 API 返回值、持久化行为、执行审计和接入文档保持一致。

## Background

- 当前 `rrule-go v1.8.2` 的 `Set` 只保存一个 RRULE，本任务保持“一个 CronJob 对应一个 `PlanRulePb` / RRULE”，不实现多 RRULE Set。
- `group_id` 已存在于 Create/Update/Submit 请求、`CronJobPb`、列表过滤、Extra 和数据库模型，但 Create/Update/Submit 响应未返回，空值也未自动生成。
- `UpdateCronJob` 与 `SubmitCronJob` 更新分支当前允许覆盖 `group_id/dept_code/type`，而这些字段应作为稳定身份或归属字段。
- CronJob 运行时按需计算下一个 occurrence，不预展开执行数据；传统 Plan 会在创建时展开日期，两者不应共享相同跨度上限。
- `HandleCronJobEventRes.message` 在 gRPC 无 transport error 时可用，但当前 `crontask.Handler` 只返回 error，成功回执 message 未写入 `cron_exec_log`。

## Requirements

- 保持单 CronJob 单规则模型；需要多个 OR 条件时创建多个 CronJob，并通过相同 `group_id` 分组。
- Create/Submit 新建时，调用方传入非空 `group_id` 则原样保存；为空时由 Trigger 生成 UUID。创建后持久化和响应中的 `group_id` 必须一致。
- Update/Submit 更新已有任务时保留原 `group_id`；请求中稳定身份字段为空或与原值一致可接受，与原值冲突时返回参数错误，不静默忽略。
- `task_code`、`group_id`、`dept_code`、`type`、`status`、软删除字段、运行历史和 lease 字段不得由配置更新覆盖。`status` 仍只由 Enable/Disable 修改。
- 配置更新只拥有规则和可变业务配置：任务名称、描述、规则及范围、排除日期、优先级、payload、业务 extra、超时策略和 ext1-ext5。
- 在途任务存在 `scheduled_time` 时拒绝 Update/Submit 配置更新，避免旧 worker 按旧 RRULE 回写下一次计划。
- Create/Update/Submit 响应均返回最终持久化的 `group_id`；Get/List 继续通过 `CronJobPb.group_id` 返回，List 继续支持按组过滤。
- CronJob 显式生效区间最大跨度从 3 年提高到 100 年；传统 Plan 和 `CalcPlanTaskDate` 继续保持 3 年限制。省略结束时间时仍默认到开始年份年末，不自动创建 100 年任务。
- `cron_exec_log` 增加 `message` 字段。gRPC 调用无 transport error 且响应非空时，记录 `HandleCronJobEventRes.message`，无论业务 receipt 最终映射为成功、删除还是普通错误；`error_message` 继续记录 Handler 最终错误。
- 执行日志写入失败不得改变原 Handler 返回结果。
- 更新 CronJob API 指南，说明分组模型、默认 UUID、响应字段、更新字段边界、100 年跨度，以及公历固定节日和农历浮动节日的配置方式。
- 文档提供元旦、国庆等公历固定日期示例；明确中秋等农历节日不能用固定公历 `month/day` 跨年表达，应先查询节假日数据并按当年公历日期创建有界单次 CronJob。

## Out of Scope

- 不修改或 fork `rrule-go`，不支持一个 CronJob 内多个 RRULE。
- 不新增组级 Enable/Disable/Delete/Run/Preview 原子接口，也不实现同组 occurrence 自动去重。
- 不改变传统 PlanTask 的单规则、日期展开或 3 年跨度行为。
- 不执行生产数据库 DDL；代码只更新 GORM 模型，生产 `cron_exec_log.message` 列由发布流程单独迁移。

## Acceptance Criteria

- [ ] Create 和 Submit 新建在 `group_id` 为空时生成非空 UUID，并在数据库、Get/List 和写入响应中保持一致；调用方指定值时原样保留。
- [ ] Update 和 Submit 更新无法修改 `task_code/group_id/dept_code/type/status`，冲突身份值返回参数错误，稳定身份与运行历史保持不变。
- [ ] Update 和 Submit 更新可修改并清空其拥有的可变配置字段；任务在途时配置更新失败且 lease 与配置均不被部分覆盖。
- [ ] Create/Update/Submit 响应 proto 和生成代码包含 `group_id`，相关测试覆盖返回值。
- [ ] CronJob 接受显式不超过 100 年的规则范围，超过 100 年报参数错误；Plan 的超过 3 年规则仍报错。
- [ ] gRPC SUCCESS、TASK_NOT_FOUND 和 UNKNOWN 等无 transport error 响应均把原始 message 写入 `cron_exec_log.message`；RPC error/nil response 的 message 为空，最终错误写入 `error_message`。
- [ ] Trigger proto 生成物和 Swagger 通过 `app/trigger/gen.sh` 更新，无手工修改生成文件。
- [ ] CronJob、Store、Logic 和 Handler 相关测试通过，且 `git diff --check` 通过。
- [ ] `docs/trigger-rrule-api-guide.md` 清楚说明分组与更新语义、100 年跨度、公历固定节日示例和中秋等农历节日限制。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
