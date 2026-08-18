# Trigger 服务

Trigger 是基于 go-zero 的统一任务调度服务，对外契约见 [`trigger.proto`](../../app/trigger/trigger.proto)。服务同时提供 asynq 异步任务、Plan 计划任务和 RRULE CronJob；三者分别解决队列投递、可管理计划生命周期和稳定周期 Handler 调度问题，存储、状态与回调语义互不混用。

## 能力选择

| 能力 | 调度基础 | 持久化 | 适用场景 |
| --- | --- | --- | --- |
| 异步任务 | asynq + Redis | Redis 队列及任务状态 | 一次性或延时的 HTTP/gRPC 回调，需要队列重试与任务查询 |
| 计划任务 | 数据库扫表 + `Plan -> Batch -> ExecItem` | `plan`、`plan_batch`、`plan_exec_item`、`plan_exec_log` | 一个计划需要拆分为日期批次和多个执行项，并支持分层暂停、恢复、终止及结果聚合 |
| CronJob | `common/crontask` + RRULE lease 调度 | `cron_job` | 以稳定 `taskCode` 注册周期 Handler，或人工执行已注册任务；不预展开批次或执行项 |

选择原则：

- 只需要投递一次回调，选择异步任务。
- 需要管理计划、批次、设备或路线执行项的完整生命周期，选择计划任务。
- 只需要按 RRULE 周期触发同一个业务 handler，选择 CronJob。
- 三类能力不能共用状态字段或数据表；RPC 请求受理、队列入队或 gRPC 调用成功，也不等于业务处理已经完成。

## 异步任务

异步任务基于 asynq 和 Redis。Worker 默认并发数为 20，队列权重为 `critical:6`、`default:3`、`low:1`，任务失败后由 asynq 按任务配置执行重试。

```text
客户端 gRPC -> Redis 队列 -> asynq Worker -> HTTP/gRPC 回调 -> 业务系统
```

- `SendTrigger` 发送 HTTP POST JSON 回调。
- `SendProtoTrigger` 调用指定的 gRPC Proto 方法。
- 入队响应表示 Trigger 已接受任务，不表示目标业务系统已经处理成功。

### 异步任务 API

| 方法 | 说明 |
| --- | --- |
| `SendTrigger` | 发送 HTTP 回调任务 |
| `SendProtoTrigger` | 发送 gRPC 回调任务 |
| `Queues` | 获取队列列表 |
| `GetQueueInfo` | 获取队列信息 |
| `GetTaskInfo` | 获取任务详情 |
| `ArchiveTask` / `DeleteTask` | 归档或删除任务 |
| `RunTask` | 立即运行队列任务 |
| `HistoricalStats` | 获取任务历史统计 |
| `ListActiveTasks` / `ListPendingTasks` / `ListScheduledTasks` / `ListRetryTasks` / `ListArchivedTasks` / `ListCompletedTasks` / `ListAggregatingTasks` | 按 asynq 状态查询任务 |
| `DeleteAllCompletedTasks` / `DeleteAllArchivedTasks` | 批量删除已完成或已归档任务 |

<div align="center">
  <img src="../images/trigger-flow.png" alt="Trigger 异步任务回调流程" style="max-width: 80%; height: auto;" />
</div>

## 计划任务

计划任务使用三级模型：

```text
Plan（计划）
  +-- Batch 1（一个计划执行时间）
  |     +-- ExecItem 1（业务执行单元）
  |     +-- ExecItem 2
  +-- Batch 2
        +-- ExecItem 3
```

创建时，Trigger 根据 `PlanRulePb`、`startTime`、`endTime`、`specifiedTimes`、`excludedTimes` 和 `excludeDates` 生成 RFC 5545 RRULE Set，再将有效候选预展开为 Batch 和 ExecItem；规则有效期跨度不能超过 3 年，且最终候选数与执行项数的乘积不能超过 5000。`skipTimeFilter=false` 时创建会过滤过去候选。`plan.rrule_str` 保存创建时的规则快照，实际运行仍由 ExecItem 的 `next_trigger_time` 驱动。

Plan 与 CronJob 共用 `(RRULE ∪ specifiedTimes) - excludedTimes - expanded(excludeDates)` 集合语义：指定时间编译为 RDATE，精确时间和整日排除编译为 EXDATE，排除优先且同一秒去重。完整字段与请求示例见 [Trigger Plan/CronJob RRULE API 场景指南](./trigger-rrule-api-guide.md)。

### 执行流程

1. `CreatePlanTask` 创建 Plan，并预展开 Batch 和 ExecItem。
2. CronService 扫描到期 ExecItem，通过带状态、版本和时间条件的更新抢占执行项。
3. Trigger 调用 StreamEvent 的 `HandlerPlanTaskEvent` 下发业务任务。
4. 下游可直接返回结果，也可通过 `CallbackPlanExecItem` 回报后续结果。
5. Trigger 更新 ExecItem，写入 `plan_exec_log`，并在全部执行项结束后聚合 Batch 和 Plan 的完成时间。

### ExecItem 状态

```text
WAITING(0) / DELAYED(10)
          |
          | 到期并成功抢占
          v
      RUNNING(100)
       |    |    |
       |    |    +-- terminated --> TERMINATED(300)
       |    +------- completed  --> COMPLETED(200)
       +------------ failed/delayed --> DELAYED(10)

WAITING(0) / DELAYED(10) -- pause --> PAUSED(150) -- resume --> WAITING(0)
```

| 状态 | 值 | 说明 |
| --- | ---: | --- |
| `WAITING` | 0 | 等待扫表触发 |
| `DELAYED` | 10 | 失败退避或业务延期，等待再次触发 |
| `RUNNING` | 100 | 已下发，等待直接结果、后续回调或再次检查 |
| `PAUSED` | 150 | 人工暂停，不参与扫表 |
| `COMPLETED` | 200 | 已完成终态 |
| `TERMINATED` | 300 | 人工终止、业务终止或超过失败重试上限 |

### 执行结果

| `execResult` | 状态变化 | 调度语义 |
| --- | --- | --- |
| `completed` | `RUNNING -> COMPLETED` | 本执行项完成 |
| `failed` | `RUNNING -> DELAYED`，超过上限后终止 | 根据 `trigger_count` 计算退避时间 |
| `delayed` | `RUNNING -> DELAYED` | 使用 `delayConfig.nextTriggerTime`；缺失或无效时默认延后 5 分钟 |
| `ongoing` | 保持 `RUNNING` | 可更新下次检查时间；缺失或无效时默认延后 5 分钟 |
| `terminated` | `RUNNING -> TERMINATED` | 本执行项不再触发 |

当前失败退避以 10 秒为基础并带少量抖动，后续逐步指数增长且最高为 30 分钟；`trigger_count` 达到 500 时自动终止。业务系统仍应根据 `execId` 做幂等处理，不能把数据库抢占等同于下游 Exactly Once。

### 计划任务 API

| 方法 | 说明 |
| --- | --- |
| `CalcPlanTaskDate` | 预计算执行日期、规则描述和 RRULE 原文 |
| `CreatePlanTask` | 创建并展开计划任务 |
| `PausePlan` / `ResumePlan` / `TerminatePlan` | 计划级控制 |
| `PausePlanBatch` / `ResumePlanBatch` / `TerminatePlanBatch` | 批次级控制 |
| `PausePlanExecItem` / `ResumePlanExecItem` / `RunPlanExecItem` / `TerminatePlanExecItem` | 执行项级控制 |
| `CallbackPlanExecItem` | 回报执行项结果 |
| `GetPlan` / `ListPlans` | 查询计划 |
| `GetPlanBatch` / `ListPlanBatches` | 查询批次 |
| `GetPlanExecItem` / `ListPlanExecItems` | 查询执行项 |
| `GetPlanExecLog` / `ListPlanExecLogs` | 查询执行流水 |
| `GetExecItemDashboard` | 获取执行项统计 |

### 计划任务数据表

| 表 | 关键字段 | 用途 |
| --- | --- | --- |
| `plan` | `plan_id`、`recurrence_rule`、`rrule_str`、`start_time`、`end_time`、`status`、`finished_time` | 计划定义、规则快照和计划级生命周期 |
| `plan_batch` | `plan_pk`、`batch_id`、`plan_trigger_time`、`status`、`finished_time` | 一个具体计划时间对应的批次 |
| `plan_exec_item` | `exec_id`、`item_id`、`next_trigger_time`、`trigger_count`、`status` | 扫表、下发、延期和执行项状态 |
| `plan_exec_log` | `exec_id`、`trigger_time`、`trace_id`、`exec_result`、`message`、`reason` | 每次执行结果流水 |

ExecItem 的核心扫表索引为 `(is_deleted, next_trigger_time, status)`。

## RRULE CronJob

CronJob 面向“一个稳定任务编码对应一个周期 Handler”的场景，适合固定频率回调、周期同步、巡检或报表等不需要预展开执行项的任务。它复用 `PlanRulePb` 表达周期，但不会创建 Plan、Batch 或 ExecItem，也没有计划任务的多级状态聚合。

`taskCode` 是调用方提供的全局唯一业务编码，供调度器和业务侧稳定识别任务；`jobId` 是 Trigger 创建的 `cron_job.id`，用于管理 RPC。两者不能混用。

常见周期、指定时间、精确排除、固定时间单次执行、创建后补触发和人工执行的完整请求见 [Trigger Plan/CronJob RRULE API 场景指南](./trigger-rrule-api-guide.md)。本节只保留能力边界和运行语义。

### 规则编译

`CreateCronJob` 将业务规则编译为带 `DTSTART`、`RRULE` 和可选 `EXDATE` 的 RFC 5545 Set：

| 输入 | 行为 |
| --- | --- |
| `startTime` | 可为空；为空时使用当前上海年份第一天，格式为 `yyyy-MM-dd HH:mm:ss` |
| `endTime` | 可为空；为空时使用补齐后开始时间所在年份最后一天；允许与开始时间相等，不能早于开始时间，跨度不能超过 100 年 |
| `rule` | 必填，支持频率、小时、分钟、星期、日期、可选月份过滤和可选 `interval` 周期步进 |
| `excludeDates` | 按 `yyyy-MM-dd` 排除当天规则中的全部小时和分钟组合 |
| `specifiedTimes` | 按上海时区的 `yyyy-MM-dd HH:mm:ss` 加入精确 RDATE 候选；最多 1000 项 |
| `excludedTimes` | 按上海时区的 `yyyy-MM-dd HH:mm:ss` 排除同一秒候选；最多 1000 项 |
| `skipTimeFilter` | 为 `true` 时，首次执行最多选择一个已经发生的计划点，不追赶全部历史周期；该计划点仍受 `maxDelay` 检查 |
| `priority` | 数字越大越优先参与到期任务选择 |
| `lockTimeout` | 单次 claim 的 lease 超时；0 使用调度器默认值，实际值不会低于 30 秒 |
| `group_id` | 创建时为空自动生成 UUID，响应中返回；同一分组下的多个 CronJob 独立调度，不会自动合并或去重 |

CronJob 不预展开执行数据，100 年仅是显式生效区间的上限。传统计划任务（Plan）创建时会展开日期，仍保持 3 年限制。

### 周期执行流程

```text
CreateCronJob
      |
      v
cron_job（RRULE + next_run）
      |
      | CronJobScheduler 扫描并通过 lease 条件更新抢占
      v
StreamEvent.HandleCronJobEvent
      |
      +-- SUCCESS ---------> 计算并提交下一次 next_run
      +-- TASK_NOT_FOUND --> 软删除 CronJob
      +-- UNKNOWN / RPC 错误 -> 保留 lease，过期后允许再次扫描
```

调度器默认每 2 秒扫描一次；抢占到任务后会快速继续扫描其他到期任务。默认 lease 为 5 分钟，任务可以通过 `lockTimeout` 覆盖。完成更新必须匹配本次 `locked_until`，避免旧 worker 覆盖新的 claim。

自动重试期间，`scheduled_time` 始终保存最初的计划执行时间，并随回调发送给业务系统。Handler 成功后才会更新 `last_run`、`last_scheduled_run` 并推进 RRULE；Handler 失败不会提前计算下一周期。lease 过期后同一计划点可能再次执行，因此业务消费者应结合 `jobId`、`taskCode` 和 `scheduledTime` 实现幂等。

### Eventstream 回执

| 回执 | Trigger 行为 |
| --- | --- |
| `CRON_JOB_RECEIPT_SUCCESS` | 本次业务处理成功，提交成功时间并计算下一个 RRULE 时间 |
| `CRON_JOB_RECEIPT_TASK_NOT_FOUND` | 业务任务已经不存在，软删除当前 CronJob，停止后续触发 |
| `CRON_JOB_RECEIPT_UNKNOWN`、空响应或 RPC 错误 | 按执行失败处理，不推进 RRULE，等待 lease 过期后重试 |

### 创建、提交与人工执行

| 操作 | 行为 |
| --- | --- |
| 创建 | `CreateCronJob` 创建新任务，校验 JSON 和规则，生成 `jobId`、RRULE Set 与首次 `next_run`；`taskCode` 冲突返回重复记录错误；`group_id` 为空时自动生成 UUID 并在响应中返回 |
| 更新 | `UpdateCronJob` 按 `jobId` 只更新可变配置（名称、描述、规则、时间范围、排除日期、优先级、payload、超时策略和扩展字段），不修改身份字段和状态；在途任务更新失败 |
| 提交 | `SubmitCronJob` 按 `taskCode` 提交；存在时委托更新保留原 `jobId/groupId` 和状态，不存在时委托创建 |
| 启用 | 从当前时间按已保存 RRULE 重新计算未来 `next_run`；重复启用幂等 |
| 禁用 | 状态改为禁用，后续扫表不再选择；已经 claim 的在途执行仍可完成 |
| 删除 | 幂等软删除，不再参与查询和调度 |
| 创建后补触发 | 创建请求设置 `skipTimeFilter=true`，首次最多选择一个过去计划点；该点未超过 `maxDelay` 时调用 Handler，不追赶全部历史周期，后续仍按原 RRULE 执行 |
| 人工执行 | `RunCronJob` 基于已有 `jobId` 异步调用同一 Handler；只在成功后更新 `last_run`，不修改周期 `next_run`、启停状态或 `last_scheduled_run` |

“创建后立即补触发”与“人工执行已有任务”是两个不同场景：前者属于 Create/Submit 的首次计划时间选择，后者由 `RunCronJob` 触发。CronJob 的启停状态只控制周期扫描，禁用任务仍可通过 `RunCronJob` 人工触发。`RunCronJob` 返回 `traceId`，仅表示异步执行请求已受理，不表示 Eventstream 已经成功处理。

### CronJob API

| 方法 | 说明 |
| --- | --- |
| `CreateCronJob` | 创建 RRULE CronJob，返回 `jobId`、`group_id` 和首次 `nextRun` |
| `UpdateCronJob` | 按 `jobId` 只更新可变配置，不修改身份字段和状态 |
| `SubmitCronJob` | 按 `taskCode` 创建或更新，不存在时创建、存在时委托更新保留身份 |
| `EnableCronJob` | 启用任务并重新计算未来执行时间 |
| `DisableCronJob` | 禁用后续扫描 |
| `DeleteCronJob` | 幂等软删除任务 |
| `RunCronJob` | 不改变周期计划地异步执行一次 |
| `GetCronJob` | 按 `jobId` 获取详情、RRULE 原文和中文规则描述 |
| `ListCronJobs` | 按任务编码、名称、状态、机构、类型和分组分页查询 |
| `PreviewCronJobSchedule` | 预览从当前时间之后的计划执行时间，严格只读不修改状态 |

完整字段说明和 JSON 场景示例见 [Trigger Plan/CronJob RRULE API 场景指南](./trigger-rrule-api-guide.md)。

### `cron_job` 数据模型

| 字段 | 说明 |
| --- | --- |
| `id` | Trigger 生成的 `jobId` |
| `task_code` | 调用方提供的全局唯一业务任务编码 |
| `rrule_str` | 实际用于调度的 RFC 5545 RRULE Set |
| `status` | `0` 禁用，`1` 启用 |
| `priority` | 调度优先级，数字越大越优先 |
| `lock_timeout` | 任务级 lease 超时，单位毫秒 |
| `next_run` | 下次计划时间；claim 期间暂存 lease 截止时间，`NULL` 表示规则耗尽 |
| `scheduled_time` | 当前在途执行最初的计划时间，自动重试期间保持不变 |
| `last_run` | 最近一次 Handler 成功完成的实际时间，包括人工执行 |
| `last_scheduled_run` | 最近一次成功周期执行的原计划时间，人工执行不更新 |
| `payload` | 业务参数 JSON |
| `dept_code` / `type` / `group_id` | 机构、任务类型和业务分组（创建时确定，不可通过更新修改） |
| `start_time` / `end_time` / `rule` / `exclude_dates` / `specified_times` / `excluded_times` | 调用方规则输入的持久化视图；空精确时间列表以 SQL `NULL` 保存并在 API 回显为空列表 |
| `cron_exec_log` | 每次回调执行的审计日志，含 `traceId`、`scheduledTime`、`cost_ms`、`status`、`message` 和 `error_message` |

核心扫描索引为 `(status, next_run)`；任务选择还会按 `priority` 降序排序。

## 辅助能力

除三类调度能力外，Trigger 还提供以下公共 RPC。字段和校验规则以 [`trigger.proto`](../../app/trigger/trigger.proto) 为准。

| 能力 | 方法 | 说明 |
| --- | --- | --- |
| 节假日查询 | `QueryHoliday`、`ListHolidayFestivals`、`GetHolidayFestival`、`GetHolidayYearSummary`、`ListHolidayYears` | 查询中国大陆日期类型、节日详情、年度汇总和已配置年份 |
| 节假日源管理 | `ListHolidaySource`、`SaveHolidaySource`、`SetHolidaySourceEnabled` | 查询、保存和启停节假日数据源配置 |
| 唯一编码 | `NextId`、`BatchNextId` | 生成单个或批量有序业务编码，不是数据库自增序列 |
| 并发调用 | `Invoke` | 按请求定义并发调用多个目标并汇总结果 |

## 配置与依赖

Trigger 启动时会同时装配 asynq Worker、asynq Scheduler、Plan CronService 和 CronJobScheduler。

```yaml
Name: trigger.rpc
ListenOn: 0.0.0.0:21006
Timeout: 120000

Redis:
  Host: 127.0.0.1:6379
  Type: node
  Pass: ""

DB:
  DataSource: "postgres://user:pass@127.0.0.1:5432/dbname?sslmode=disable&TimeZone=Asia/Shanghai"

StreamEventConf:
  Endpoints:
    - 127.0.0.1:21009
  NonBlock: true
  Timeout: 120000
```

| 依赖 | 使用方 |
| --- | --- |
| Redis | asynq 队列、任务查询和 Trigger 的其他 Redis 能力 |
| MySQL、PostgreSQL/openGauss 或项目支持的关系数据库 | Plan 四表、`cron_job` 和节假日数据 |
| StreamEvent gRPC 服务 | Plan 的 `HandlerPlanTaskEvent` / `NotifyPlanEvent`，CronJob 的 `HandleCronJobEvent` |

## 部署与并发语义

```bash
cd app/trigger
go run . -f etc/trigger.yaml
```

多实例需要共享 Redis 和关系数据库。Plan 通过条件更新与版本字段抢占 ExecItem，CronJob 通过 `next_run` lease token 和完成 CAS 降低并发覆盖风险；这些机制不替代下游业务幂等，也不构成 Exactly Once、跨节点故障转移或可靠送达承诺。

## 监控与日志定位

- OpenTelemetry trace 覆盖任务执行链路。
- asynq 任务可通过队列、任务详情、状态列表和历史统计 RPC 查询。
- Plan 通过 `plan_exec_log` 和 `GetExecItemDashboard` 查询执行历史及统计。
- Plan 扫表链路的日志正文以 `[cron-plan]` 开头；CronJob 公共调度器日志以 `[crontask]` 开头。
- CronJob 日志包含 `task_code`、`task_id`、`scheduled_run` 和 `locked_until` 等定位字段；每次执行写入 `cron_exec_log` 记录成本、回执和错误。

## 参考

- [`trigger.proto`](../../app/trigger/trigger.proto) - Trigger RPC 契约
- [`streamevent.proto`](../../facade/streamevent/streamevent.proto) - Plan 与 CronJob 业务回调契约
- [`app/trigger/internal/cronjob`](../../app/trigger/internal/cronjob) - CronJob 规则编译、Store 和 Handler 适配
- [`common/crontask`](../../common/crontask) - 通用 RRULE 调度器与 lease 契约
- [Trigger Plan/CronJob RRULE API 场景指南](./trigger-rrule-api-guide.md) - 周期规则、指定时间、精确与整日排除及 Plan/CronJob 示例
