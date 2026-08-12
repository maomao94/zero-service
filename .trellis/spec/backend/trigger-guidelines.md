# Trigger 调度规范

## 适用范围

修改 `app/trigger` 的异步任务、计划任务、节假日规则、CronJob RPC/Store，或任何服务调用这些能力时读取。通用 Cron 调度器规则见 [crontask-guidelines.md](./crontask-guidelines.md)。

## 三类能力不得混用

- asynq 异步任务：`SendTrigger` / `SendProtoTrigger` 将一次性或延时回调写入 Redis 队列，队列成功不等于业务回调成功。
- 计划任务：`Plan -> Batch -> ExecItem` 是数据库持久化生命周期，CronService claim 到期 ExecItem，业务通过 `CallbackPlanExecItem` 回报结果并聚合上级状态。
- CronJob：`Create/Enable/Disable/Delete/Run/List` 使用 `common/crontask`，面向稳定 task code 的周期或人工执行。

调用方必须先确定需要的是队列任务、可管理计划，还是周期 handler；不要只因都叫“定时任务”就复用状态或表。

依据：`app/trigger/trigger.proto`、`docs/trigger.md`、`app/trigger/internal/logic`、`app/trigger/internal/cronjob`。

## 契约源与边界

- RPC 字段、状态和控制方法以 `app/trigger/trigger.proto` 为源；修改后执行 `app/trigger/gen.sh`。
- asynq payload 与 callback target 在入队前完成校验和序列化；worker 负责调用和返回错误，让 asynq 决定重试，不在 Logic 伪造完成。
- 计划任务的创建、控制、claim、执行回调和状态聚合由各自 Logic/store 拥有。直接改表绕过状态机可能破坏父子状态。
- `exec_id` 标识一次执行项，`item_id` 是业务项标识，Plan/Batch ID 也各有所有权；不能用单个通用 ID 替换。

## 计划与时间规则

- 计划有效范围由 `startTime` / `endTime` 与 recurrence rule 共同决定；连续月计划使用时间范围和日规则表达。
- `month` 是可选的 `BYMONTH` 过滤；只有需要显式限制月份时才传，字段为空不能被当作缺少连续月信息。
- 日期展开必须受有效范围和现有跨度限制约束，保持时区与日历规则一致；节假日查询通过 Trigger 的 holiday 能力，不在调用服务复制日历数据。
- ExecItem 只有通过 store 的所有权条件被当前 worker claim 后才能下发；回调按 `execResult` 和 delay 配置驱动状态，不用 HTTP/gRPC 成功替代业务结果。
- Plan/Batch/ExecItem 与 CronJob 是不同状态机。两者可以复用 `CompileSchedule` 这类纯规则编译函数，但 Plan 仍在创建时用 `set.All()` 展开 Batch/ExecItem，不得接入 `TaskConfig`、lease 或 `TaskStore`。
- `Plan.RRuleStr` 只保存创建时用于展开日期的完整 Set 快照，供审计、详情描述和核对 Batch 日期；它不是 Plan 的运行时调度来源。
- Plan 与 CronJob 的共享编译结果遵循 `(RRULE ∪ specified_times) - excluded_times - expanded(exclude_dates)`；同秒去重且排除优先，整日排除同时删除当天的 RRULE 与 RDATE 候选。`rule` 仍为必填，不支持纯 RDATE-only 输入或时间范围排除。
- Plan 使用 3 年上限的 `CompileSchedule` 并预展开最终 Set；`CreatePlanTask` 在 `skip_time_filter=false` 时过滤过去候选，且 `候选数 × exec_items 数` 不得超过 5000。CronJob 使用 100 年上限的专用编译入口并按需推进，不预展开 occurrence。

## Scenario: Plan/Batch 终止与 ExecItem Claim

### 1. Scope / Trigger

- 修改 `TerminatePlan`、`TerminatePlanBatch`、`LockTriggerItem` 或执行回执状态更新时适用。

### 2. Signatures

```go
func CountRunningExecItemsByPlan(ctx context.Context, db *gorm.DB, planPk string) (int64, error)
func CountRunningExecItemsByBatch(ctx context.Context, db *gorm.DB, batchPk string) (int64, error)
func UpdatePlanTerminated(ctx context.Context, db *gorm.DB, id, reason, updateUser string, now time.Time) (int64, error)
func UpdatePlanBatchTerminated(ctx context.Context, db *gorm.DB, id, reason, updateUser string, now time.Time) (int64, error)
func LockTriggerItem(ctx context.Context, db *gorm.DB, dbType gormx.DatabaseType, expireIn time.Duration) (*PlanExecItem, error)
```

### 3. Contracts

- `StatusRunning` 是唯一禁止终止父 plan/batch 的 ExecItem 状态；它覆盖刚下发尚未回调和 `last_result=ongoing` 两种情况。
- `waiting/delayed/paused/completed/terminated` 不阻止父级终止，且父级终止不得批量修改这些 ExecItem 状态。
- claim 的候选查询通过 JOIN 筛选 enabled 的 plan/batch，最终更新使用 ExecItem 的 `version/status/next_trigger_time` 乐观 CAS；claim 后重新加载 exec、plan 和 batch，在调用下游前补查父级状态，不使用父级 `FOR UPDATE` 或额外父级子查询。
- 终止入口在事务中查询 running 数量，父级更新只保护父级当前状态，不加入父级锁或 ExecItem `EXISTS` 子查询。
- claim 与终止竞争时，claim 后重新加载 ExecItem、Plan 和 Batch；父级非 enabled 或已有 finished time 时不得调用下游。该补查只保证“不再调用下游”，不保证 ExecItem 不会已被 CAS 为 running。
- 条件更新 `RowsAffected == 0` 表示状态竞争失败，不是成功。
- callback/cron 的 ExecItem 状态更新竞争失败时，不得写成功流水或继续聚合 finished 通知。

### 4. Validation & Error Matrix

- 作用域内存在 `StatusRunning` -> `BIZ_STATE`，父级、finished time、ExecItem 和通知均不变。
- 父级已 terminated/finished -> `BIZ_STATE`。
- claim 候选查询无可调度记录或父级非 enabled -> `model.ErrNotFound`；ExecItem CAS 竞争失败 -> `model.ErrNoRowsUpdate`。两者均不调用下游。
- callback 条件更新零行 -> `BIZ_STATE`，不写 `plan_exec_log`。

### 5. Good/Base/Bad Cases

- Good: claim 先完成并可见，ExecItem 进入 running，随后终止请求明确失败；若终止先提交且 cron 已取得候选，claim 后父级复查至少停止下游调用。
- Base: 只有 waiting/delayed/paused/completed/terminated，终止成功且 ExecItem 状态原样保留。
- Bad: claim 后直接调用下游，不重新加载并检查父级状态。

> **Warning**: 当前无父级锁、无 claim UPDATE 父级条件、无 CAS 补偿的设计仍有窗口：终止提交后，旧候选可能被 ExecItem CAS 改为 running，随后父级复查阻止下游。不要把“未调用下游”描述为“父级终止后不会产生 running”。若产品要求后者，必须增加共同同步点、父级条件或版本化补偿，并补真实并发测试。

### 6. Tests Required

- 覆盖所有 ExecItem 状态，断言只有 running 拒绝终止。
- 断言终止成功后 ExecItem 状态不变。
- 断言 plan 或 batch 非 enabled 时 claim 返回 `ErrNotFound`；均 enabled 时 claim 转为 running。
- 断言两个 claim 竞争同一 ExecItem 时只有匹配原 version/status/next_trigger_time 的 CAS 成功。
- 断言 completed/delayed/ongoing/terminated 等条件更新零行返回 `ErrNoRowsUpdate`，且调用路径不写成功流水。
- 对 model、logic 和 cron 相关包运行 race test。
- 并发验收分别断言“主状态不变化”和“下游副作用不发生”；只断言未调用下游不能证明没有遗留 running 状态。

### 7. Wrong vs Correct

#### Wrong

```go
if runningCount == 0 {
	db.Save(&plan)
}
```

#### Correct

```go
running, err := CountRunningExecItemsByPlan(ctx, tx, plan.ID)
if err != nil || running > 0 {
	// 数据库错误或已有 running item，均不得更新父级或发送 finished 通知。
}
rows, err := UpdatePlanTerminated(ctx, tx, plan.ID, reason, userID, now)
if err != nil || rows == 0 {
	// 数据库错误或父级状态冲突，均不得发送 finished 通知。
}
```

依据：`app/trigger/model/gormmodel/plan.go`、`app/trigger/model/gormmodel/plan_test.go`、`app/trigger/internal/logic/terminateplanlogic.go`、`app/trigger/internal/logic/terminateplanbatchlogic.go`、`app/trigger/cron/cronservice.go`、`app/trigger/internal/logic/callbackplanexecitemlogic.go`。

## Plan Cron 日志正文标记

- `app/trigger/cron` 的服务生命周期、扫表、下游调用、结果处理和收尾日志，正文统一以 `[cron-plan] ` 开头，便于在 plain 日志中直接检索。
- `planscope.Scope.LogMessage` 是带具体 Scope 日志的统一格式入口；只有 `EntryCron` 增加 `[cron-plan]`，RPC 与 callback 正文保持不变。
- 不依赖具体 Plan/ExecItem 的 cron 生命周期日志使用 `planscope.CronPlanLogMessage`，不得在各调用点重复硬编码前缀。
- `common/crontask` 继续使用 `[crontask]`，表示公共 Scheduler/CronJob/RunNow；同一正文不得同时添加 `[cron-plan]` 与 `[crontask]`。

```go
// Plan 扫表链路
scope.Logger(ctx).Info(scope.LogMessage("下游返回：执行完成（completed）"))

// gRPC 主动操作或 callback
scope.Logger(ctx).Info(scope.LogMessage("RPC 执行回调：收到下游回执"))
```

测试至少断言 cron Scope 会增加 `[cron-plan]`，RPC/callback Scope 不增加，并覆盖共享延期告警在两种入口下的差异。

### Common Mistake: 使用无业务命名空间的 `[cron]`

**Symptom**: Plan 扫表日志、公共 `common/crontask` 调度器日志和未来其他业务 cron 混在一起，无法只靠正文前缀定位一条业务链路。

**Cause**: 把“执行方式是 cron”误当成唯一日志分类，忽略了 cron 还需要业务命名空间。

**Fix**: Plan 扫表使用 `[cron-plan]`，公共调度器保持 `[crontask]`，未来业务按 `[cron-<business>]` 扩展；gRPC 主动操作不继承 cron 正文标记。

**Prevention**: 新增或调整 cron 日志时，先确认日志生产组件和业务命名空间，再通过 `planscope.Scope.LogMessage` 或对应组件的统一格式入口生成正文，并补充前缀边界测试。

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
- 使用合法 `PlanRulePb` 验证生成的完整 Set 可通过 `crontask.ValidateRRule`，并断言时区、`EXDATE` 和 `next_run`。

### 7. Wrong vs Correct

#### Wrong

```go
task, err := buildCronJobTask(&trigger.CreateCronJobReq{/* copy another RPC */})
updated, err := store.GetByID(ctx, task.ID) // only for response
```

#### Correct

```go
task, err := buildCronJobTask(cronJobTaskData{/* key configuration */})
nextRun := tool.CarbonFromTimeStartOfSecond(task.NextRun).ToDateTimeString()
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
- `scheduled_time` 表示原计划执行点，重试期间保持不变，由 `formatTime` 来源保证 `yyyy-MM-dd HH:mm:ss`；回调 PB 不引入 PGV validation。
- 字段号从 1 连续对齐，不保留历史字段号兼容（用户已明确允许覆盖）；RPC 方法名与 `CronJobReceiptPb` 回执枚举保持不变。
- 收到回执后：`CRON_JOB_RECEIPT_SUCCESS` 视为成功；`CRON_JOB_RECEIPT_TASK_NOT_FOUND` 返回 `crontask.ErrDeleteTask`；未知回执与传输错误按普通错误重试。

### 4. Validation & Error Matrix

- Create/Submit/List 的 `task_code` `max_len: 128`，`cron_job.task_code` GORM `size:128` + `uniqueIndex:uq_cron_job_task_code`；生产发布前必须按现有迁移流程扩列，模型声明不替代 DDL。
- 回调 PB 无校验规则，校验责任归属 Trigger 请求侧与 `formatTime` 来源。
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
	ScheduledTime: formatTime(task.ScheduledTime),
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

## Scenario: CalcPlanTaskDate 规则描述

### 1. Scope / Trigger

- `CalcPlanTaskDate` 同时返回日期预览和面向用户的规则描述时适用。

### 2. Signatures

```proto
message CalcPlanTaskDateRes {
  repeated string planDates = 1;
  string scheduleDescription = 2;
  string rruleStr = 3;
}
```

### 3. Contracts

- `scheduleDescription` 必须由展开 `planDates` 的同一个 `rrule.Set` 生成。
- `rruleStr` 返回该 Set 的 RFC 5545 原文，供排障和与持久化快照比对。
- Logic 在完成 DTSTART、RRULE 和 EXDATE 组装后调用 `crontask.DescribeRRule(set.String())`。
- proto 是契约源，修改后执行 `app/trigger/gen.sh`，不得手改生成文件。

### 4. Validation & Error Matrix

- 请求或 RRULE 生成失败 -> 参数错误。
- RRULE 描述失败 -> 参数错误，不返回只有日期而缺少描述的部分响应。
- 规则有效且可描述 -> 同时返回 `planDates` 与非空 `scheduleDescription`。

### 5. Good/Base/Bad Cases

- Good: 每天 09:30 且排除一天 -> 日期列表移除当天，描述包含同一排除时间。
- Base: 未传 start/end -> 使用 Logic 规范化后的本年边界生成日期和描述。
- Bad: 直接从 `PlanRulePb` 拼中文 -> 容易遗漏 DTSTART 默认值、UNTIL 时区和 EXDATE。

### 6. Tests Required

- 断言每天 09:30 的描述、规范化有效期和排除日期。
- 断言 `planDates` 数量与 EXDATE 后结果一致。
- 运行 `app/trigger/gen.sh` 后确认 descriptor 和 Go 生成类型均包含字段 2。

### 7. Wrong vs Correct

#### Wrong

```go
description := describePlanRule(in.Rule)
```

#### Correct

```go
description, err := crontask.DescribeRRule(set.String())
```
