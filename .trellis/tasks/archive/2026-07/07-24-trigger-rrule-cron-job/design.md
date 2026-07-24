# Trigger RRULE Cron Job Design

## Architecture

数据流如下：

```text
CreateCronJobReq
  -> Trigger 规则编译与首次 NextRun 计算
  -> cron_job / Trigger DBStore
  -> common/crontask Scheduler claim
  -> Eventstream HandleCronJobEvent
  -> SUCCESS / TASK_NOT_FOUND / error
  -> Complete CAS / Delete / lease retry
```

`common/crontask` 保持业务无关。Trigger 私有转换层负责 `PlanRulePb`、时间边界和排除日期；Scheduler 只消费 `RRuleStr`、`NextRun`、Payload 和 Extra。

## Crontask Contract

- 删除 `TaskConfig.Version`。并发所有权由不可变的 `LockedUntil` token 表达，而不是未落库的版本号。
- `LockAndFetch` 返回 `TaskClaim{Task, LockedUntil}`。`Task.NextRun` 保留原计划时间，Eventstream 重试时始终收到该时间。
- `TaskConfig.LockTimeout` 是任务级单次 claim 锁超时。值大于 0 时覆盖 Scheduler 的 `LockExpire` 默认值；零值使用 Scheduler 默认值。通用层使用 `time.Duration`，支持该字段的 Proto 和数据库列使用毫秒。
- `Complete` 使用 `id + next_run=LockedUntil` CAS 更新下一次和上一次成功时间。禁用只阻止新扫描，不撤销已经执行中的 claim；CAS 丢失返回 `ErrNotFound`，旧执行不得覆盖新配置。
- Handler 普通 error 不调用 `Complete`，lease 到期后重新 claim。
- `ErrDeleteTask` 是控制信号。Scheduler 调用 Store 的软删除；删除成功或记录已不存在均终止当前处理，删除数据库错误则保留 lease 等待重试。
- `RunNow` 不复用周期完成路径；成功只写 `LastRun`，不修改 `NextRun/Status`。
- Store 以 `Enable(ctx,id)`、`Disable(ctx,id)` 暴露幂等生命周期操作；`List(ctx,ListCondition)` 使用状态数组过滤。

## Persistence

`cron_job` 使用 `gormx.LegacyStringBaseModel`，不嵌入 `VersionMixin`。通用列为 `task_code`、`task_name`、`rrule_str`、`priority`、`lock_timeout`、`payload`、`extra`、`status`、`next_run`、`last_run`。`lock_timeout` 使用毫秒且 0 表示未配置；`next_run/last_run` 使用 `sql.NullTime`。Trigger 另用 nullable `scheduled_time` 固化首次 claim 的计划时间，普通错误重试沿用，成功完成或重新启用后清空。

业务列为 `dept_code`、`type`、`group_id`、`description`、`start_time`、`end_time`、`rule`、`exclude_dates`、`ext_1..ext_5`。`rule` 使用 JSON 文本；可选的 `start_time`、`end_time` 和 `exclude_dates` 使用 nullable 类型，保留调用方未传的事实。

`CronJobExtra` 包含上述业务字段和调用方 `bizExtra`。写入时保存请求原值，不把规则编译时补齐的本年边界伪装成调用方输入；读取时以平铺列为真源重建 Extra，避免列与旧 JSON 漂移。

Trigger DBStore 的候选查询必须包含未删除、启用、`next_run IS NOT NULL`、`next_run <= now`；claim UPDATE 再校验启用状态和 SELECT 时观察到的 `next_run`。

`Enable(ctx,id)` 在 Store 内读取已保存 RRULE 并从当前时间重算未来 `NextRun`，已启用任务重复调用不重算；`Disable(ctx,id)` 只更新状态。

## Rule Semantics

- 创建请求不接受 `RRuleStr`，由 Trigger 复用现有 `PlanRulePb` 语义生成 RRULE set。
- 未传开始/结束时间时沿用 Plan 当前“默认本年”语义；所有输入按项目本地时区解析。
- 排除日期由 Trigger 业务转换层应用，不进入通用 TaskConfig 字段。
- `skipTimeFilter=false` 选择当前时间之后的首次计划。
- `skipTimeFilter=true` 最多产生一次立即补触发；首次成功后以当前时间为基准跳到未来计划，不逐次追赶历史。
- 规则 COUNT/UNTIL 耗尽时 `NextRun` 为 Go 零值和 SQL `NULL`，Status 保持 enabled。

## RPC Contracts

Trigger 新增 `CreateCronJob`、`EnableCronJob`、`DisableCronJob`、`DeleteCronJob`。`CreateCronJobReq.lockTimeout` 使用毫秒并要求非负；创建响应字段为 `jobId`、`nextRun`。生命周期请求仅使用 `jobId`。启停重复调用幂等，删除不存在记录也成功。

ISP `101-1` 任务下发 Item 不包含 `lock_timeout`，不得解析或覆盖该字段。ISP Store 仍以毫秒持久化任务级锁超时，任务配置查询以 `int64 lock_timeout` 返回持久化值；已有任务收到 `101-1` 配置更新时必须保留原值。

Eventstream 新增 `HandleCronJobEvent`。请求携带 Job/Task 标识、Priority、Payload、Extra、Trigger 业务字段和 `scheduledTime`。回执枚举为 UNKNOWN、SUCCESS、TASK_NOT_FOUND：UNKNOWN 和 RPC error 返回普通 error，TASK_NOT_FOUND 转换为 `ErrDeleteTask`。

## Compatibility And Rollback

- TaskStore 接口调整会同步修改 MemoryStore、ISP DBStore 和所有测试调用方，确保全仓编译。
- `TaskConfig.Version` 删除只影响 common/crontask 直接引用；现有 Plan/PlanExecItem 的 `VersionMixin` 保持不变。
- 开发与测试环境加入 `cron_job` AutoMigrate；生产环境按 GORM 模型部署等价表结构。
- 回滚时可停止注册新 Scheduler 和 RPC，现有 Plan 调度路径不受影响。
