# Trigger CronJob

修改 `app/trigger` 的 CronJob 创建、更新、提交、预览、到点回调或执行日志时读取。底层 lease、claim 与 `RunNow` 语义见[调度器契约](../scheduling.md)，RRULE Set 算法见[RRULE 契约](../rrule.md)。

## CronJob 适配

- `specified_times` 与 `excluded_times` 是 Plan/CronJob 共用的精确时间输入。二者均按 `Asia/Shanghai` 的 `yyyy-MM-dd HH:mm:ss` 解析、必须位于规范化后闭区间内且各最多 1000 项；分别编译为 `RDATE` 与精确 `EXDATE`。最终候选为 `RRULE ∪ RDATE - EXDATE - expanded(exclude_dates)`，同秒去重且排除优先。`exclude_dates` 还必须排除当天的 RDATE。
- Store 遵循 lease/complete CAS、SQL NULL 终止时间和字段所有权，详见公共调度规范。
- `scheduled_time` 表示原计划执行点，重试时保持不变；attempt/实际开始时间另行记录。
- 到点回调 `HandleCronJobEventReq` 只携带扁平关键业务字段，不传 `extra`、不携带调度器内部运行字段，详见「Scenario: CronJob 到点回调契约」。
- `RunCronJob` 触发人工执行，不改变周期 `next_run` 或启停状态。
- CronJob Handler 注册集中在 `ServiceContext`/cronjob 组装边界，业务服务通过 task code 与 payload 解耦。
- CronJob 详情/列表的 `rruleStr` 和 `scheduleDescription` 必须来自持久化 `TaskConfig.RRuleStr`，不能从业务 JSON 重新编译。
- `PreviewCronJobSchedule` 按 `job_id` 查询现有任务，并通过同一个 `CronJobScheduler.PreviewNextRuns` 预览未来时间；禁用任务仍可预览。
- 预览必须消费持久化完整 RRULE Set，不能用管理视图中的 `rule/start_time/end_time/exclude_dates` 重新编译。创建时默认补齐的时间范围和 `EXDATE` 已固化在 `RRuleStr` 中，重新编译可能改变实际调度语义。
- `count=0` 由 Logic 默认成 10，Proto 限制最大 100；返回数量不足只表示规则或过滤器已耗尽，不是错误。
- 预览严格只读：不得修改 `next_run`、`scheduled_time`、启停状态或运行历史，不得调用 Handler、`RunNow` 或写 `CronExecLog`。
- 预览时间来自当前请求时刻之后的规则候选，不以数据库 `next_run` 为游标；`next_run` 在 claim 期间可能承载 lease 截止时间，不等同于纯规则 occurrence。
- 使用 `NewLoggingEventHandler(db, client)` 装饰 Handler：每次 cron job 执行后自动写入 `CronExecLog` 记录（字段：job_id、task_code、task_name、scheduled_time、start_time、end_time、cost_ms、status、message、error_message）。gRPC 无 transport error 且响应非空时，`message` 保存原始业务回执；`error_message` 保存 Handler 最终错误。执行日志写入失败不影响 handler 返回结果。`CronExecLog` 已在 `ServiceContext` 的 Dev/Test 模式下通过 `db.MustAutoMigrate` 自动建表。
- 一个 CronJob 只编译一个 `PlanRulePb` / RRULE；多个 OR 条件拆为多个 CronJob，并用稳定 `group_id` 分组。Trigger 不对同组同一计划点自动去重。
- Create/Submit 新建在 `group_id` 为空时生成 UUID；Update/Submit 更新已有任务保留原 `group_id`。Create/Update/Submit 响应返回最终 `group_id`。
- CronJob 不预展开 occurrence，显式生效范围最多 100 年；传统 Plan 和 `CalcPlanTaskDate` 创建时展开日期，继续保持 3 年上限。省略 `end_time` 时两者仍默认到开始年份年末。

### Exact-Time Persistence

- `cron_job.specified_times` 与 `cron_job.excluded_times` 是可空 JSON 文本列；空或 nil 列表写 SQL `NULL`，读取后在 `CronJobPb` 表示空列表。
- `CronJobExtra` 是 TaskConfig 运行时载体。模型转换必须在写入时平铺两个列表、读取时重建两个列表，Get/List 只能由同一转换链路回显，不能从 `rrule_str` 反推原输入。
- Create、Update 和 Submit 的新建/更新分支都必须将两个列表传给 CronJob 专用编译入口。完整 Set 仍是首次 `next_run`、Enable、完成推进与 Preview 的唯一运行时权威来源。
- 两个 JSON 列与 `rrule_str` 属于同一配置更新单元：Store 白名单更新和 `scheduled_time IS NULL` 条件必须在同一事务中完成。任务在途时零行更新返回 `ErrUpdate`，不得部分替换 JSON 列或覆盖 lease。

依据：`app/trigger/trigger.proto`、`app/trigger/internal/logic/previewcronjobschedulelogic.go`、`app/trigger/internal/logic/cronjoblogic_test.go`、`common/crontask/crontask.go`。

## Scenario: CronJob 更新与幂等提交

### 1. Scope / Trigger

- 管理端按 Trigger `job_id` 更新 CronJob，或上游按稳定 `task_code` 重复提交完整配置时适用。

### 2. Signatures

```protobuf
rpc UpdateCronJob(UpdateCronJobReq) returns (UpdateCronJobRes);
rpc SubmitCronJob(SubmitCronJobReq) returns (SubmitCronJobRes);
```

### 3. Contracts

- `UpdateCronJobReq` 用 `job_id` 定位任务，不接收 `task_code`；服务端保留原 `task_code`、状态和运行历史。
- `SubmitCronJobReq` 用 `task_code` 定位：有效记录存在时更新并保留 `job_id`，不存在时创建。
- Create/Update/Submit 响应均返回最终 `group_id`；Update/Submit 还返回 `job_id`、`task_code`、`next_run`。
- 写入响应直接使用本次编译得到的 `NextRun`，成功后不为组装响应再次查询数据库。
- 共享配置编译使用 Logic helper 的传输中立内部数据对象，不让 Update/Submit 构造其他 RPC 的 PB 请求。
- Create/Update/Submit 必须通过 CronJob 专用编译入口生成 `RRuleStr`、`RuleJSON` 和首次 `NextRun`，不能分别计算或从业务 JSON 反推；Plan 继续使用 3 年上限的单规则入口。
- CronJob 编译使用 `Asia/Shanghai`，生成完整的 `DTSTART + RRULE + EXDATE` Set，并固定 `BYSECOND=0`；`skip_time_filter` 只影响首次 `NextRun`，最多补一个过去计划点。
- Store 配置更新只拥有任务名称、描述、RRULE、规则范围、排除日期、优先级、超时、payload、业务扩展和 ext 字段；`task_code/group_id/dept_code/type`、状态、软删除、执行历史和 lease 不属于配置更新。
- Store 普通配置 UPDATE 使用 `scheduled_time IS NULL` 原子拒绝在途任务，零行返回 `ErrUpdate`；成功后再更新 `next_run`，不得接受配置更新后让旧 worker 用旧 RRULE 回写下一计划点。

### 4. Validation & Error Matrix

- 请求校验、JSON 或 RRULE 无效 -> `PARAM_INVALID`。
- Update/Submit 请求的稳定身份字段与原值冲突，或任务正在执行 -> `PARAM_INVALID`。
- Update 的 `job_id` 不存在或已删除 -> `RECORD_NOT_EXIST`。
- Create 重复 `task_code` -> `RECORD_ALREADY_EXIST`。
- Submit 插入唯一冲突后二次查询仍无有效记录 -> `RECORD_ALREADY_EXIST`，表示软删除历史占用编码。
- Submit 对相同 `task_code` 和相同配置重复提交 -> 取决于数据库 changed-rows 语义；配置 UPDATE 零行时返回 Store 更新失败。
- 其他 Store 错误 -> `DB`。

### 5. Good/Base/Bad Cases

- Good: Submit 同一 `task_code` 更新原 `job_id`，禁用任务保持禁用。
- Base: Submit 新 `task_code` 创建新任务；Update 按 `job_id` 修改规则并返回原 `task_code`。
- Bad: Update 暴露可修改的 `task_code`，或写入成功后仅为返回 `next_run` 再查一次数据库。
- Bad: 配置 Update 无条件覆盖 `next_run`，导致正在执行任务的 lease token 丢失。

### 6. Tests Required

- 断言 Create 重复仍返回冲突。
- 断言 Update 保留 `task_code/group_id/dept_code/type`、状态和执行历史；在途任务更新失败且配置与 lease 均不变。
- 断言 Submit 创建、更新及软删除编码冲突。
- 断言 Submit 更新已有任务时保留原 `job_id`；零行更新按 Store `ErrUpdate` 契约处理。
- 使用合法 `PlanRulePb` 验证生成的完整 Set 可通过 `rrulex.Validate`，并断言时区、`EXDATE` 和 `next_run`。

### 7. Wrong vs Correct

#### Wrong

```go
task, err := buildCronJobTask(&trigger.CreateCronJobReq{/* copy another RPC */})
updated, err := store.GetByID(ctx, task.ID) // only for response
```

#### Correct

```go
task, err := buildCronJobTask(cronJobTaskData{/* key configuration */})
nextRun := carbonx.FormatDateTime(task.NextRun)
```

## Scenario: CronJob 到点回调契约

### 1. Scope / Trigger

- 修改 Trigger 到 StreamEvent 的 `HandleCronJobEventReq`（`facade/streamevent/streamevent.proto`）、handler 回调构造或回调字段映射时适用。

### 2. Signatures

```protobuf
rpc HandleCronJobEvent (HandleCronJobEventReq) returns (HandleCronJobEventRes);

message HandleCronJobEventReq {
  string job_id = 1 [json_name = "jobId"];
  string task_code = 2 [json_name = "taskCode"];
  string task_name = 3 [json_name = "taskName"];
  int32 priority = 4;
  string payload = 5;
  string scheduled_time = 6 [json_name = "scheduledTime"];
  string type = 7;
  string group_id = 8 [json_name = "groupId"];
  string description = 9;
  string ext1 = 10;
  string ext2 = 11;
  string ext3 = 12;
  string ext4 = 13;
  string ext5 = 14;
  string dept_code = 15 [json_name = "deptCode"];
}
```

### 3. Contracts

- 回调是**扁平关键业务字段**请求：身份、名称、优先级、payload、本次原计划时间、类型、分组、描述、扩展和机构编码。`type/group_id/description/ext1-5/dept_code` 由 handler 通过 `ParseExtra(task.Extra)` 解析后映射，不传递 `extra` 原文。
- `TaskConfig.Extra` 是 Trigger 内部为通用 Scheduler 重建业务模型列的运行时载体，不属于下游业务契约，管理视图与回调均不暴露。
- 回调不携带调度器内部运行字段（`next_run`、lease、rule、RDATE/EXDATE、`rrule_str` 等）：claim 后 `next_run` 已被清零，下游无法消费这些值；本次执行的原计划点只读 `scheduled_time`。
- `scheduled_time` 表示原计划执行点，重试期间保持不变，由 `carbonx.FormatDateTimeOrEmpty` 保证 `yyyy-MM-dd HH:mm:ss` 与零值空字符串；回调 PB 不引入 PGV validation。
- 字段号从 1 连续对齐，不保留历史字段号兼容（用户已明确允许覆盖）；RPC 方法名与 `CronJobReceiptPb` 回执枚举保持不变。
- 收到回执后：`CRON_JOB_RECEIPT_SUCCESS` 视为成功；`CRON_JOB_RECEIPT_TASK_NOT_FOUND` 返回 `crontask.ErrDeleteTask`；未知回执与传输错误按普通错误重试。

### 4. Validation & Error Matrix

- Create/Submit/List 的 `task_code` `max_len: 128`，`cron_job.task_code` GORM `size:128` + `uniqueIndex:uq_cron_job_task_code`；生产发布前必须按现有迁移流程扩列，模型声明不替代 DDL。
- 回调 PB 无校验规则，校验责任归属 Trigger 请求侧与 `carbonx.FormatDateTimeOrEmpty` 的受控时间来源。
- `ParseExtra` 解析失败 -> handler 返回错误，不发送回调。

### 5. Good/Base/Bad Cases

- Good: handler 以扁平字段发送完整关键业务字段（含 Type/GroupId/Description/Ext1-5/DeptCode），`scheduled_time` 为原计划时间，回执处理正确。
- Base: 无 Extra JSON 或空业务扩展字段时回调仍发送身份与 payload，下游按空值处理。
- Bad: 把 `TaskConfig.Extra` 原文字符串发给下游；或把 `next_run`、规则等调度器内部字段塞进回调模型。

### 6. Tests Required

- 断言 handler 构造的请求包含全部扁平字段（身份、业务扩展、机构编码、`scheduled_time`），且无 `extra` 字段。
- 断言 `task_code` 128 rune 通过 Create/Submit/List 校验、129 rune 被拒绝；模型 tag 声明 `size:128` 且唯一索引不变。
- 断言回执三种分支（成功、`ErrDeleteTask`、重试）行为不变。

### 7. Wrong vs Correct

#### Wrong

```go
response, err := client.HandleCronJobEvent(ctx, &streamevent.HandleCronJobEventReq{
	JobId: task.ID, TaskCode: task.TaskCode,
	Extra: string(task.Extra), // 把内部适配 JSON 泄露给下游
})
```

#### Correct

```go
extra, err := ParseExtra(task.Extra)
if err != nil {
	return "", err
}
response, err := client.HandleCronJobEvent(ctx, &streamevent.HandleCronJobEventReq{
	JobId: task.ID, TaskCode: task.TaskCode, TaskName: task.TaskName,
	Priority: int32(task.Priority), Payload: string(task.Payload),
	ScheduledTime: carbonx.FormatDateTimeOrEmpty(task.ScheduledTime),
	Type: extra.Type, GroupId: extra.GroupId, Description: extra.Description,
	Ext1: extra.Ext1, Ext2: extra.Ext2, Ext3: extra.Ext3, Ext4: extra.Ext4, Ext5: extra.Ext5,
	DeptCode: extra.DeptCode,
})
```

## 反模式

- 用 asynq task state 代替 Plan/ExecItem 状态，或反向共用表。
- 回调未到就把发送成功写成 ExecItem 完成。
- 整行更新 Plan/Batch/ExecItem，覆盖其他控制路径拥有的状态。
- 把 `month` 当作所有月度计划必填字段。
- 手工调用 CronJob handler 绕过 `Scheduler.RunNow`。

## 验证

- Proto 变更执行 `app/trigger/gen.sh` 并测试所有直接调用方。
- 计划测试覆盖创建展开、Pause/Resume/Terminate、claim 竞争、重复/迟到回调、delay/ongoing/failed/completed 和父级聚合。
- CronJob 运行 `go test ./common/crontask ./app/trigger/internal/cronjob`，覆盖 lease、人工执行和终止时间。
