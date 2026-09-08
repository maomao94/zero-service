# Trigger Plan 与 ExecItem

## 适用范围

修改 `app/trigger` 的 Plan、Batch、ExecItem、节假日规则或执行回调时读取。CronJob 见[Trigger CronJob](./cron-jobs.md)，通用调度器见[调度器契约](../scheduling.md)。

## 三类能力不得混用

- asynq 异步任务：`SendTrigger` / `SendProtoTrigger` 将一次性或延时回调写入 Redis 队列，队列成功不等于业务回调成功。
- 计划任务：`Plan -> Batch -> ExecItem` 是数据库持久化生命周期，CronService claim 到期 ExecItem，业务通过 `CallbackPlanExecItem` 回报结果并聚合上级状态。
- CronJob：`Create/Enable/Disable/Delete/Run/List` 使用 `common/crontask`，面向稳定 task code 的周期或人工执行。

调用方必须先确定需要的是队列任务、可管理计划，还是周期 handler；不要只因都叫“定时任务”就复用状态或表。

依据：`app/trigger/trigger.proto`、`docs/trigger/trigger.md`、`app/trigger/internal/logic`、`app/trigger/internal/cronjob`。

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
