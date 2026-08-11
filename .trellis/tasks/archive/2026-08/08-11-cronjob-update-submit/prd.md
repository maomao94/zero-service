# CronJob 更新与提交接口

## Goal

为 Trigger CronJob 增加明确更新和幂等提交能力，使管理方可以按 Trigger `job_id` 精确修改任务，上游可以按自己的唯一 `task_code` 重复提交最终配置，而不改变现有严格创建、软删除和唯一索引语义。

## Background

- `CreateCronJob` 当前只调用 `TaskStore.Insert`；`task_code` 重复时返回记录已存在。
- `common/crontask.TaskStore` 已提供 `GetByID`、`GetByCode`、`Insert` 和 `Update`，本任务不扩展公共 Store 接口。
- `cron_job.task_code` 保持全局唯一，软删除记录仍占用该编码；删除后上游必须使用新的 `task_code` 创建新任务。
- `TaskStore.Update` 已负责保留 `last_run`、`last_scheduled_run`、`scheduled_time` 等运行历史，并在任务在途时避免覆盖作为 lease token 的 `next_run`。

## Requirements

- 保留 `CreateCronJob` 的严格创建语义，重复 `task_code` 继续返回记录已存在。
- 新增 `UpdateCronJob` RPC，使用 `job_id` 定位当前未删除任务，并在独立、扁平的请求消息中接收完整 CronJob 配置。
- `UpdateCronJobReq` 不接收 `task_code`，服务端按 `job_id` 保留原任务编码。
- `UpdateCronJob` 保留原任务的 `job_id`、启用/禁用状态和运行历史；调度配置变化后按新规则计算未来 `next_run`。
- 新增 `SubmitCronJob` RPC，使用独立、扁平的 `SubmitCronJobReq/Res`，按 `task_code` 判断创建或更新。
- `SubmitCronJob` 查询到当前未删除任务时更新并保留原 `job_id` 和启用/禁用状态；查询不到时创建并生成新 `job_id`。
- `SubmitCronJob` 必须处理并发首次提交：插入发生唯一冲突后再次按 `task_code` 查询，若查到当前任务则转为更新。
- 若唯一冲突后二次查询仍找不到任务，视为编码已被软删除历史记录占用，返回记录已存在，不恢复旧记录，也不修改表结构或唯一索引。
- `UpdateCronJob` 与 `SubmitCronJob` 统一返回 `job_id`、`task_code` 和当前可展示的 `next_run`，不返回 created/updated 操作类型。
- 请求参数、JSON、RRULE 和扩展字段的校验行为与 `CreateCronJob` 保持一致。
- Proto 字段使用 `snake_case`；生成文件必须由 `app/trigger/gen.sh` 产生，不手工编辑。
- Create、Update、Submit 三种写请求的可变配置字段顺序、校验规则和业务注释必须逐项对齐；Update 使用 `job_id` 替代 `task_code` 作为身份字段。

## Out Of Scope

- 不修改 `common/crontask.TaskStore` 接口及调度器行为。
- 不修改 `cron_job` 表结构、`task_code` 唯一索引或软删除模型。
- 不支持恢复已删除 CronJob，也不允许删除后复用原 `task_code`。
- `UpdateCronJob` 不暴露 `task_code` 输入字段。
- 不新增部分更新、字段掩码或操作类型枚举。

## Acceptance Criteria

- [ ] `CreateCronJob` 的现有创建和重复冲突行为保持不变。
- [ ] `UpdateCronJob` 按有效 `job_id` 更新配置并返回相同 `job_id`；不存在或已删除时返回记录不存在类错误。
- [ ] `UpdateCronJob` 不接收 `task_code`，响应返回原任务的 `task_code`。
- [ ] 更新已禁用任务后仍为禁用状态，更新已启用任务后仍为启用状态。
- [ ] 更新不覆盖 `last_run`、`last_scheduled_run`、`scheduled_time`；任务在途时不覆盖 lease `next_run`。
- [ ] `SubmitCronJob` 在 `task_code` 不存在时创建，在有效记录存在时更新并返回原 `job_id`。
- [ ] 两个并发或竞争的首次 Submit 不产生两条记录；唯一冲突后二次查询可收敛为更新。
- [ ] `task_code` 只存在于软删除记录时，Submit 返回记录已存在，不创建或恢复任务。
- [ ] 非法 JSON、非法 RRULE 和缺少必填字段在三个写接口中得到一致的校验错误。
- [ ] `app/trigger/gen.sh`、Trigger 相关定向测试、`git diff --check` 均通过。
