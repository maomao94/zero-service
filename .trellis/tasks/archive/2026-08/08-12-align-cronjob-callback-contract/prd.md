# Align CronJob callback contract

## Goal

将 Trigger CronJob 的业务编码容量提升到 128，并让 Trigger 到 StreamEvent 的 `HandleCronJobEventReq` 对回调字段执行与上游持久化契约一致的校验，避免任务创建成功后在数据库、查询或回调边界失败。

## Background

- `CreateCronJobReq.task_code` 与 `SubmitCronJobReq.task_code` 当前最大 64，用户已要求调整为 128。
- `ListCronJobsReq.task_code` 过滤条件仍最大 64，`cron_job.task_code` GORM 列为 `size:64`，必须同步调整才能形成端到端契约。
- Trigger 的 `HandleCronJobEvent` 调用原会发送 JobId、TaskCode、TaskName、Priority、Payload、Extra、ScheduledTime、Type、GroupId、Description、Ext1-5 和 DeptCode，其中 `extra` 从未被下游使用。
- 回调只需要关键业务字段，不需要调度器内部运行字段（claim 后 `next_run` 已被清零），因此回调字段集收敛为扁平关键字段；字段号顺延对齐，不保留兼容。

## Requirements

- 将 `CreateCronJobReq.task_code`、`SubmitCronJobReq.task_code` 和 `ListCronJobsReq.task_code` 的最大长度统一调整为 128；响应和管理视图不重复请求校验。
- 将 `gormmodel.CronJob.TaskCode` 调整为 `size:128`，保留唯一索引和字段语义。开发/测试 AutoMigrate 继续随模型更新；生产环境通过现有发布迁移流程扩列。
- 直接覆盖 `HandleCronJobEventReq`，字段号顺延对齐（不保留旧字段号兼容），只保留回调必需的关键业务字段：
  - `job_id`、`task_code`、`task_name`、`priority`、`payload`、`scheduled_time`、`type`、`group_id`、`description`、`ext1`-`ext5`、`dept_code`，序号从 1 连续排列。
  - 删除 `extra`：`TaskConfig.Extra` 仅作为 Trigger 内部模型适配载体，不对下游回调暴露。
  - 不引入嵌套 CronJob 管理模型，不使用 PGV validation（`facade/streamevent/streamevent.proto` 不导入 `validate/validate.proto`）。
- `scheduled_time` 是回调请求的执行上下文；claim 后回调模型中的 `next_run` 已被清零，因此回调不传递调度器内部运行字段，本次原计划点只读取 `scheduled_time`。
- 分别运行 `app/trigger/gen.sh` 与 `facade/streamevent/gen.sh`，不手工修改生成代码。
- 增加边界测试，覆盖 128/129 字符 task code 的请求校验与模型声明、扁平回调字段映射。

## Acceptance Criteria

- [ ] 128 rune 的 TaskCode 可通过 Create、Submit 和 List 请求校验，129 rune 被拒绝。
- [ ] CronJob 模型声明可存储 128 长度 TaskCode，唯一索引语义不变。
- [ ] `HandleCronJobEventReq` 字段号连续对齐且仅含关键业务字段，无 `extra`、无 validate 引用；生成脚本成功且生成物与源 Proto 一致。
- [ ] Trigger handler 通过 `ParseExtra` 构造扁平回调请求，回调测试断言身份、业务扩展、机构编码与本次 `scheduled_time` 均对齐，原有回执处理不变。

## Out Of Scope

- 不修改 JobId、GroupId、Type、DeptCode、Ext 字段的产品容量。
- 不改变 CronJob 调度、RDATE/EXDATE、lease、回执或重试语义。
- 不给 Payload 引入新的大小限制或 JSON schema。
- 不在本任务中新增生产数据库迁移文件；发布时需按现有流程将 `cron_job.task_code` 扩为 128。

## Risks

- 已存在的生产数据库列仍可能是 varchar(64)；应用上线前必须先完成扩列。
- 字段号顺延对齐属于破坏性契约变更，旧客户端需与服务端同版本发布后再启用。
