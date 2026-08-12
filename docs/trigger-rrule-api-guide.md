# Trigger Plan/CronJob RRULE API 场景指南

本文面向 Trigger Plan 和 CronJob 对接方，说明周期规则、指定时间与排除条件如何组成同一个 RRULE Set，并给出可直接用于构造 gRPC JSON 请求的示例。RPC 字段与校验以 [`trigger.proto`](../app/trigger/trigger.proto) 为准，服务能力总览见 [Trigger 服务](./trigger.md)。

> 约定：示例 JSON 一律使用 snake_case 字段名，与 proto 字段名一致（如 `task_code`、`specified_times`、`exclude_dates`）。

## 统一候选集合

Plan 与 CronJob 使用同一套编译语义：

```text
最终候选 = (RRULE ∪ specified_times) - excluded_times - expanded(exclude_dates)
```

| 输入 | Set 表示 | 语义 |
| --- | --- | --- |
| `rule` | `RRULE` | 必填的周期规则；当前不支持省略 RRULE、只提交 RDATE |
| `specified_times` | `RDATE` | 向周期候选并集精确加入一个或多个带完整日期的时间点；每项只加入该秒，不会变成每日重复的时钟时间 |
| `excluded_times` | 精确 `EXDATE` | 删除同一秒的候选，无论候选来自 RRULE 还是 `specified_times` |
| `exclude_dates` | 展开的 `EXDATE` | 删除该上海自然日内由 RRULE 和 `specified_times` 产生的全部候选 |

`specified_times` 和 `excluded_times` 均严格使用 `yyyy-MM-dd HH:mm:ss`，按 `Asia/Shanghai` 解析，每个列表最多 1000 项。每个时间必须位于补齐默认值后的 `[start_time, end_time]` 闭区间内，因此等于开始或结束边界是合法的。`specified_times` 的每一项都是带年月日的单个 occurrence，例如 `2027-08-13 17:30:00` 只加入这一天的 17:30，不表示每天 17:30。所有候选精确到秒，同一秒重复出现只保留一次；排除始终优先，某一秒同时出现在 RRULE、`specified_times` 和排除输入中时，最终不会执行。

`exclude_dates` 只表示整日排除，不支持时间段排除，也不会产生替代执行日。要排除单个时刻使用 `excluded_times`；要加入不规则时刻使用 `specified_times`。

### Plan 与 CronJob 的运行差异

| 项目 | Plan | CronJob |
| --- | --- | --- |
| API | `CalcPlanTaskDate`、`CreatePlanTask` | `CreateCronJob`、`UpdateCronJob`、`SubmitCronJob`、`GetCronJob`、`ListCronJobs`、`PreviewCronJobSchedule` |
| 显式范围上限 | 3 年 | 100 年 |
| 候选处理 | 创建时展开完整 Set，生成 Batch 和 ExecItem | 持久化完整 Set，按需计算首次 `next_run`、完成后的下一次时间和预览结果 |
| 过去时间 | `CreatePlanTask` 在 `skip_time_filter=false` 时删除过去候选；为 `true` 时保留范围内候选 | `skip_time_filter` 只影响首次 `next_run`，为 `true` 时最多选择一个过去候选 |
| 数量限制 | `最终候选数 × exec_items 数 <= 5000` | 不预展开，不使用 5000 执行项限制 |

Plan 的 `rrule_str` 是创建时展开所用 Set 的审计快照，运行时由 ExecItem 驱动。CronJob 的持久化 `rrule_str` 则是运行时权威来源：首次执行时间、Enable、成功完成后的推进和 Preview 都消费该 Set。

## Plan 请求示例

以下示例以每天 09:00 的 RRULE 为基础，额外加入 17:30 和次日 14:15，精确排除 09:00，并整日排除 8 月 14 日。最终只保留 `2027-08-13 17:30:00`；8 月 14 日的 RRULE 与指定时间都会被删除。

### 预览 `CalcPlanTaskDate`

```json
{
  "start_time": "2027-08-13 00:00:00",
  "end_time": "2027-08-14 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": ["2027-08-14"],
  "specified_times": ["2027-08-13 17:30:00", "2027-08-14 14:15:00"],
  "excluded_times": ["2027-08-13 09:00:00"]
}
```

响应同时返回最终 `plan_dates`、由同一个 Set 生成的 `schedule_description` 和完整 `rrule_str`。建议先预览再创建，但创建时仍会独立校验并编译请求。

### 创建 `CreatePlanTask`

```json
{
  "plan_id": "demo-irregular-plan-202708",
  "plan_name": "不规则指定时间计划",
  "type": "demo",
  "group_id": "rrule-guide",
  "description": "周期候选与精确时间合并示例",
  "start_time": "2027-08-13 00:00:00",
  "end_time": "2027-08-14 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": ["2027-08-14"],
  "interval_time": 0,
  "interval_type": 0,
  "exec_items": [
    {
      "item_id": "order-demo-001",
      "item_type": "order",
      "item_name": "示例执行项",
      "payload": "{\"orderId\":\"demo-001\"}",
      "request_timeout": 60000
    }
  ],
  "batch_num_prefix": "RRULE",
  "skip_time_filter": false,
  "specified_times": ["2027-08-13 17:30:00", "2027-08-14 14:15:00"],
  "excluded_times": ["2027-08-13 09:00:00"],
  "dept_code": "DEMO"
}
```

`skip_time_filter=false` 时，创建时刻之前的最终候选会被过滤；若过滤后没有候选则创建失败。`skip_time_filter=true` 会保留范围内的过去候选并为其创建执行项，不等于 CronJob 的“最多补一个过去计划点”。

## CronJob 精确时间生命周期

### 创建不规则时间任务

一个 CronJob 可以用 RRULE 表达稳定基线，再用 `specified_times` 加入无法由字段笛卡尔积准确表达的带日期时间点。下面的 RRULE 在有效范围内每天 09:00 执行，并且只为 8 月 13 日和 14 日指定加入 17:30；它不会因此变成每天 17:30，也不是纯 RDATE 任务。

```json
{
  "task_code": "demo-daily-irregular-times",
  "task_name": "每天不规则时间示例",
  "type": "demo",
  "group_id": "rrule-guide",
  "description": "每天 09:00，并为两个日期指定加入 17:30",
  "start_time": "2027-08-01 00:00:00",
  "end_time": "2027-08-31 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": ["2027-08-14"],
  "specified_times": ["2027-08-13 17:30:00", "2027-08-14 17:30:00"],
  "excluded_times": ["2027-08-13 09:00:00"],
  "priority": 10,
  "payload": "{\"source\":\"rrule-guide\"}",
  "lock_timeout": 60000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

首次 `next_run` 从编译后的完整 Set 计算。8 月 13 日的 09:00 被精确排除，8 月 14 日的 09:00 和 17:30 均被整日排除。若多个带日期的指定 occurrence 具有同一个 Handler 和任务身份，可放在同一个 CronJob 的 `specified_times`；只有 Handler、`task_code`、状态或管理身份需要独立时才拆成多个 CronJob。

### Update/Submit 替换或清空列表

`UpdateCronJob` 和 `SubmitCronJob` 接收完整配置，列表采用替换语义。传空数组会清空已保存的精确时间列表，并重新编译 Set；省略 repeated 字段在 gRPC JSON 中也表示空列表，不是“保持原值”。例如按 `job_id` 清空两个列表：

```json
{
  "job_id": "0198-demo-cron-job-id",
  "task_name": "每天不规则时间示例",
  "description": "清空精确时间列表，仅保留周期规则",
  "start_time": "2027-08-01 00:00:00",
  "end_time": "2027-08-31 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "specified_times": [],
  "excluded_times": [],
  "priority": 10,
  "payload": "{\"source\":\"rrule-guide\"}",
  "lock_timeout": 60000,
  "max_delay": 1800,
  "skip_time_filter": false
}
```

`SubmitCronJob` 使用同样的 `specified_times: []` 和 `excluded_times: []` 清空列表，但以稳定 `task_code` 定位；已有任务保留原 `job_id`、`group_id` 和启停状态，不存在时创建。Update/Submit 会在同一事务内替换规则、两个列表和 `next_run`。只要 `scheduled_time` 非空，说明任务正在执行或重试，更新会被原子拒绝，不会部分清空列表或覆盖 lease；调用方应等待本次执行完成后重试。

例如按 `task_code` 提交完整配置并清空两个列表：

```json
{
  "task_code": "demo-daily-irregular-times",
  "task_name": "每天不规则时间示例",
  "type": "demo",
  "group_id": "rrule-guide",
  "description": "提交后清空精确时间列表，仅保留周期规则",
  "start_time": "2027-08-01 00:00:00",
  "end_time": "2027-08-31 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "specified_times": [],
  "excluded_times": [],
  "priority": 10,
  "payload": "{\"source\":\"submit-demo\"}",
  "lock_timeout": 60000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

Create、Update 和 Submit 成功响应都直接返回本次编译得到的 `next_run`；Update/Submit 还返回最终 `job_id`、`task_code` 和 `group_id`，不会为了组装响应再次从数据库读取旧配置。下例展示清空之前 Get/List 中单个 `CronJobPb` 的精确时间字段；清空之后两个字段均为 `[]`：

```json
{
  "job_id": "0198-demo-cron-job-id",
  "task_code": "demo-daily-irregular-times",
  "task_name": "每天不规则时间示例",
  "status": 1,
  "next_run": "2027-08-01 09:00:00",
  "group_id": "rrule-guide",
  "specified_times": ["2027-08-13 17:30:00", "2027-08-20 17:30:00"],
  "excluded_times": ["2027-08-13 09:00:00"]
}
```

### Get/List 回显

`GetCronJob` 请求：

```json
{
  "job_id": "0198-demo-cron-job-id"
}
```

`ListCronJobs` 请求：

```json
{
  "page_size": 20,
  "page_num": 1,
  "task_code": "demo-daily-irregular-times",
  "status": [0, 1],
  "group_id": "rrule-guide"
}
```

两者返回的 `CronJobPb` 都直接回显持久化的 `specified_times` 和 `excluded_times`；数据库用可空 JSON 文本列保存原始列表，未配置或已清空时写入 SQL `NULL`，API 统一返回空列表。`rrule_str` 与 `schedule_description` 来自持久化完整 Set，不会从回显列表重新推导。Preview 请求和响应示例见[预览未来计划时间](#预览未来计划时间)，其结果自然包含 RDATE 并应用精确与整日 EXDATE，且严格只读。

## API 边界

| API | 定位方式 | 用途 | 关键行为 |
| --- | --- | --- | --- |
| `CreateCronJob` | 请求中的全局唯一 `task_code` | 创建一个新 CronJob | `task_code` 已存在时返回重复记录错误；成功后返回 Trigger 生成的 `job_id`、最终 `group_id` 和首次 `next_run` |
| `UpdateCronJob` | 已创建任务的 `job_id` | 更新规则和可变业务配置 | 保留 `task_code`、`group_id`、`dept_code`、`type`、启停状态和运行历史；任务正在执行时拒绝更新 |
| `SubmitCronJob` | 稳定 `task_code` | 幂等提交配置 | 有效任务不存在时创建，存在时更新可变配置并保留原 `job_id`、`group_id` 和启停状态 |
| `RunCronJob` | 已创建任务的 `job_id` | 人工异步执行一次 | 不创建任务，不修改周期 `next_run`、启停状态或 `last_scheduled_run`；返回 `trace_id` 用于追踪 |
| `PreviewCronJobSchedule` | 已创建任务的 `job_id` | 预览未来计划时间 | 直接读取持久化 RRULE Set，应用排除日期和调度器时间过滤；不修改任务或执行 Handler |

本文的周期示例均使用 `CreateCronJob` 请求体。需要由上游反复提交同一个业务任务配置时，可以将相同配置字段用于 `SubmitCronJob`；两者都会通过同一规则编译逻辑生成 RRULE Set 和 `next_run`。

创建成功只表示任务已经持久化并返回首次计划时间，不表示异步业务回调已经成功。调度 lease 过期、网络错误等情况可能使同一计划点再次下发，下游应结合 `job_id`、`task_code` 和计划时间实现幂等。

### 单规则与任务分组

一个 CronJob 只包含一个 `rule`，并编译成一条 RRULE。需要“每月 3 日或每周五”这类无法由单条 RRULE 表达的 OR 条件时，应创建多个 CronJob，并为它们设置相同的 `group_id`：

```text
group_id = monthly-or-friday
├── CronJob A：每月 3 日 09:00
└── CronJob B：每周五 09:00
```

同组 CronJob 仍然拥有独立的 `job_id`、`task_code`、状态和 `next_run`，Trigger 不会合并或去重同一秒命中的任务。业务要求同组同一计划点只处理一次时，下游应使用 `group_id + scheduled_time` 等稳定业务键实现幂等。

创建时未传 `group_id` 或传空字符串，Trigger 会生成 UUID 并在 Create/Submit 响应中返回。Update 和 Submit 更新已有任务时保留原 `group_id`；`task_code`、`group_id`、`dept_code`、`type` 和状态不能通过配置更新修改。

## 规则字段

### `PlanRulePb`

| 字段 | 是否必填 | 取值与语义 |
| --- | --- | --- |
| `freq` | 是 | `0-YEARLY`、`1-MONTHLY`、`2-WEEKLY`、`3-DAILY`、`4-HOURLY`、`5-MINUTELY` |
| `month` | 否 | 月份过滤，元素范围 `1-12`；同一数组内多个值表示多个固定月份 |
| `day` | 否 | 月中的日期，元素范围 `1-31` 或 `-1` 至 `-31`，不能为 `0`；`-1` 表示当月最后一天 |
| `week` | 否 | 星期过滤，`1-7` 分别表示周一至周日 |
| `hours` | 是 | 至少一个元素，范围 `0-23` |
| `minutes` | 是 | 至少一个元素，范围 `0-59` |
| `interval` | 否 | 周期步进，缺省 `0` 或 `1` 表示每个周期都执行；`N>1` 表示每隔 `N` 个 `freq` 周期执行一次，相位从任务开始时间起算。例如 `{"freq":1,"day":[5],"hours":[9],"minutes":[0],"interval":3}` 表示从开始时间所在周期起每 3 个月的 5 号 09:00 执行 |

Trigger 会把以上字段映射为 RRULE 的 `INTERVAL`、`BYMONTH`、`BYMONTHDAY`、`BYDAY`、`BYHOUR` 和 `BYMINUTE`，并固定加入 `BYSECOND=0`。因此所有示例都在第 0 秒执行，请求中没有秒字段。

`month`、`day`、`week`、`hours` 和 `minutes` 都是过滤条件。要表达全天每分钟或每小时执行，需要像下文示例一样覆盖全部小时；只传 `hours: [0]` 会把执行时间限制在 00 点。

`interval` 不是"从任意位置开始滚 N 个周期"，它的步进锚定任务的 `start_time`。例如 `freq=1`（MONTHLY）、开始时间 2027 年 1 月 5 日、`interval=3` → 固定落在 1、4、7、10 月的 5 号。要表达"每年固定 1、4、7、10 月"应直接用 `freq=0`（YEARLY）的固定 `month` 集合，两者语义不同。

`interval` 在各频次中的效果：

| freq | interval 行为 | 注意 |
|---|---|---|
| YEARLY / MONTHLY / WEEKLY / DAILY | 每天/周/月/年隔 interval 个周期执行 | 直接生效，`freq=3, interval=2` = 隔天 |
| HOURLY | 每隔 interval 个小时执行 | `minutes` 照常过滤；`minutes:[30], interval:2` = 每 2 小时的 :30 |
| MINUTELY | 每隔 interval 个分钟**从 start_time 分钟起跳**，但 `minutes` 是必填过滤条件 | ⚠️ 步长落点**必须落在 `minutes` 列表内**，否则该步被跳过 |

MINUTELY + interval 的三种实际效果（以 `start_time` 分钟为 00 为例）：

| `minutes` | `interval` | 实际发生时间 | 说明 |
|---|---|---|---|
| `[0,10,20,30,40,50]` | `10` | 每 10 分钟：:00、:10、:20… | 步长 10 分钟的落点全在列表里 |
| `[0]` | `10` | 每小时**仅 :00 一次** | 步长 10 分钟，但只有 :00 落在列表里，其余被丢弃 |
| `[15]` | `10` | **空规则，永不执行** | 步长 10 分钟的落点永远是 00,10,20…，一个都不在列表里 |

因此"每 N 分钟"直接用 `minutes` 列表表达，不要叠加 `interval`。

### 开始与结束时间

规则统一使用 `Asia/Shanghai` 时区。

| 项目 | 说明 |
| --- | --- |
| 字段 | `start_time`、`end_time` 均可省略或传空字符串 |
| 格式 | `yyyy-MM-dd HH:mm:ss` |
| 生效边界 | 开始和结束时间都包含在规则范围内 |
| 相等时间 | `start_time == end_time` 合法，可构造只有一个候选点的固定时间单次任务 |
| 错误顺序 | `end_time < start_time` 时请求失败 |
| 最大跨度 | CronJob 的 `end_time` 不能晚于 `start_time` 加 100 年 |

省略字段时按以下方式补齐：

| `start_time` | `end_time` | 编译 RRULE 时使用的范围 |
| --- | --- | --- |
| 空 | 空 | 当前上海时间所在年份的 `01-01 00:00:00` 至 `12-31 23:59:59` |
| 有值 | 空 | 指定开始时间至该开始时间所在年份的 `12-31 23:59:59` |
| 空 | 有值 | 当前上海年份的 `01-01 00:00:00` 至指定结束时间；若结束时间早于默认开始时间则失败 |
| 有值 | 有值 | 使用调用方指定范围，并执行顺序与 100 年跨度校验 |

默认值会写入实际用于调度的 RRULE Set，但管理视图中的 `start_time`、`end_time` 保留调用方原始输入；调用方传空字符串时，查询结果仍可能为空，而 `rrule_str` 已包含补齐后的边界。生产配置建议显式填写两个字段，避免任务范围随创建年份变化。下文所有完整创建示例均显式填写范围。

100 年是调用方显式配置的最大有效范围，不是默认有效期。CronJob 不会预先展开整个范围，只在调度和预览时按需计算后续 occurrence。传统 PlanTask 会创建时展开计划日期，仍保持 3 年限制。

### 创建配置字段

| 字段 | 语义 |
| --- | --- |
| `task_code` | 调用方提供的全局唯一稳定业务编码，最长 64 字符；与 Trigger 生成的 `job_id` 不是同一个标识 |
| `task_name` | 任务名称，必填，最长 128 字符 |
| `type` | 业务任务类型，必填，最长 64 字符 |
| `group_id` | 稳定业务分组 ID，最长 64 字符；创建时为空由 Trigger 生成 UUID，更新已有任务时不可修改 |
| `description` | 可选描述，最长 200 字符 |
| `dept_code` | 机构编码，必填，最长 64 字符 |
| `exclude_dates` | 排除日期数组，格式 `yyyy-MM-dd`；Trigger 会排除该日规则中全部小时和分钟组合 |
| `specified_times` | 指定执行时间数组，编译为 RDATE；格式、时区、数量和范围见[统一候选集合](#统一候选集合) |
| `excluded_times` | 精确排除时间数组，编译为 EXDATE；排除同一秒的 RRULE/RDATE 候选 |
| `priority` | 非负整数；到期任务选择时数字越大优先级越高 |
| `payload` | 传给业务 Handler 的参数字符串；为空表示无参数，非空时必须是合法 JSON 字符串 |
| `lock_timeout` | 单次调度 lease 超时，单位毫秒；`0` 使用调度器默认值，实际有效锁时长不会低于 30 秒 |
| `max_delay` | 最大延迟容忍，单位秒；`0` 使用调度器默认值，超过该延迟时跳过本次并计算下一计划点 |
| `skip_time_filter` | 只影响首次 `next_run`；`true` 时最多选择一个已经发生的计划点用于尽快补触发，不追赶全部历史周期；该计划点仍受 `max_delay` 检查 |

`payload` 的 proto 类型是字符串，因此 JSON 请求中需要把业务 JSON 转义为字符串，例如 `"payload": "{\"orderId\":\"demo-001\"}"`，不能直接传一个 JSON 对象。

### 更新字段边界

`UpdateCronJob` 只接收可变配置字段（名称、描述、规则及范围、排除日期、优先级、payload、超时策略和 `ext1` 至 `ext5`），不包含 `task_code`、`group_id`、`dept_code`、`type` 和 `status`。`SubmitCronJob` 更新已有任务时委托 Update 处理，同样不修改身份字段。任务存在正在执行或重试的 `scheduled_time` 时，Update 会失败，调用方应在本次执行完成后重试。

## 创建示例

以下日期仅用于展示规则。实际接入时应替换 `task_code`、业务字段和生效区间，并确保区间没有过期。

### 每分钟第 0 秒执行一次

`hours` 覆盖全天，`minutes` 覆盖每一分钟。

```json
{
  "task_code": "demo-every-minute",
  "task_name": "每分钟执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每天每分钟第 0 秒执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2027-12-31 23:59:59",
  "rule": {
    "freq": 5,
    "month": [],
    "day": [],
    "week": [],
    "hours": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23],
    "minutes": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59]
  },
  "exclude_dates": [],
  "priority": 10,
  "payload": "{\"source\":\"cron-guide\"}",
  "lock_timeout": 60000,
  "max_delay": 120,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每 10 分钟第 0 秒执行一次

`MINUTELY` 的 `interval` 与 `minutes` 不应叠加（原因见[规则字段](#规则字段)中 MINUTELY + interval 的说明），因此用固定分钟列表表达：

```json
{
  "task_code": "demo-every-10-minutes",
  "task_name": "每 10 分钟执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每天每小时的 0、10、20、30、40、50 分执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2027-12-31 23:59:59",
  "rule": {
    "freq": 5,
    "month": [],
    "day": [],
    "week": [],
    "hours": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23],
    "minutes": [0, 10, 20, 30, 40, 50]
  },
  "exclude_dates": [],
  "priority": 10,
  "payload": "{\"source\":\"cron-guide\"}",
  "lock_timeout": 60000,
  "max_delay": 300,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每小时整点执行一次

```json
{
  "task_code": "demo-hourly",
  "task_name": "每小时整点执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每天每小时的 00 分 00 秒执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2027-12-31 23:59:59",
  "rule": {
    "freq": 4,
    "month": [],
    "day": [],
    "week": [],
    "hours": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 10,
  "payload": "{\"source\":\"cron-guide\"}",
  "lock_timeout": 60000,
  "max_delay": 300,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每年固定季度月份 1 日 09:00 执行

该示例固定在每年 1、4、7、10 月执行，不随任务开始时间变化。需要"从任务开始时间起滚动每 3 个月"时使用 `interval: 3`，见下一个示例；两者语义不同，不能混用。

```json
{
  "task_code": "demo-quarter-months",
  "task_name": "固定季度月份执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每年 1、4、7、10 月 1 日 09:00 执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2029-12-31 23:59:59",
  "rule": {
    "freq": 0,
    "month": [1, 4, 7, 10],
    "day": [1],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"reportType\":\"quarterly\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每 3 个月 5 号 09:00 执行一次（10 年滚动示例）

`interval: 3` 从任务开始时间所在周期起滚动步进，适合"每季度一次"且不固定日历月份的业务。下面示例跨度 10 年（2027-2036），开始时间所在月份为 1 月，因此发生在每年 1、4、7、10 月的 5 号 09:00，共 40 个候选点；若开始时间改为 2 月 5 日，则依次落在 2、5、8、11 月。

```json
{
  "task_code": "demo-rolling-quarter-10y",
  "task_name": "滚动每 3 个月执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "从 2027 年 1 月起每 3 个月 5 号 09:00 执行，共 40 次",
  "start_time": "2027-01-05 00:00:00",
  "end_time": "2036-12-31 23:59:59",
  "rule": {
    "freq": 1,
    "month": [],
    "day": [5],
    "week": [],
    "hours": [9],
    "minutes": [0],
    "interval": 3
  },
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"reportType\":\"rolling-quarterly\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

编译得到的 RRULE 为 `FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=5;BYHOUR=9;BYMINUTE=0;BYSECOND=0`，中文描述为"以开始时间为基准，按 3 个月间隔，第 5 天 09:00 执行"。候选点依次为 2027-01-05、2027-04-05、2027-07-05、2027-10-05……最后一个为 2036-10-05。`end_time` 只要覆盖范围内即可，超出最后一个完整季度不影响已生成的候选。

### 每年元旦 09:00 执行

元旦是公历固定日期，可以直接使用 YEARLY 规则：

```json
{
  "task_code": "demo-new-year-day",
  "task_name": "元旦固定日期任务",
  "type": "holiday",
  "group_id": "major-holidays",
  "description": "每年 1 月 1 日 09:00 执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2126-12-31 23:59:59",
  "rule": {
    "freq": 0,
    "month": [1],
    "day": [1],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"holiday\":\"new-year-day\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每年国庆节 09:00 执行

国庆节 10 月 1 日也是公历固定日期：

```json
{
  "task_code": "demo-national-day",
  "task_name": "国庆固定日期任务",
  "type": "holiday",
  "group_id": "major-holidays",
  "description": "每年 10 月 1 日 09:00 执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2126-12-31 23:59:59",
  "rule": {
    "freq": 0,
    "month": [10],
    "day": [1],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"holiday\":\"national-day\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

这里的规则只表示每年公历 10 月 1 日，不代表每年实际公布的国庆放假和调休区间。需要 10 月 1 日至 7 日每天执行时，可以使用 `day: [1,2,3,4,5,6,7]`；需要跟随官方调休安排时，应使用节假日数据生成当年的具体任务。

### 一个任务覆盖 2026-2030 年的元旦、国庆、中秋、端午

公历固定节日（元旦、国庆）可以用 RRULE 表达，农历浮动节日（中秋、端午）的公历日期每年变化，需要先从节假日数据查询，再通过 `specified_times` 逐项加入。同一 Handler 和任务身份下，两类日期可以混在同一个 CronJob。元旦 1 月 1 日和国庆 10 月 1 日都是每月 1 日，`month: [1,10]` 与 `day: [1]` 的笛卡尔积恰好只有这两个交点：

```json
{
  "task_code": "demo-major-holidays-2026-2030",
  "task_name": "2026-2030 主要节假日任务",
  "type": "holiday",
  "group_id": "major-holidays",
  "description": "2026-2030 年元旦、国庆 09:00 及当年中秋、端午 09:00 执行",
  "start_time": "2026-01-01 00:00:00",
  "end_time": "2030-12-31 23:59:59",
  "rule": {
    "freq": 0,
    "month": [1, 10],
    "day": [1],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "specified_times": [
    "2026-06-19 09:00:00",
    "2026-09-25 09:00:00",
    "2027-06-09 09:00:00",
    "2027-09-15 09:00:00",
    "2028-05-28 09:00:00",
    "2028-10-03 09:00:00",
    "2029-06-16 09:00:00",
    "2029-09-22 09:00:00",
    "2030-06-05 09:00:00",
    "2030-09-12 09:00:00"
  ],
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"holidays\":[\"new-year-day\",\"national-day\",\"mid-autumn\",\"dragon-boat\"],\"years\":\"2026-2030\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

该请求产生 20 个候选点：RRULE 每年 1 月 1 日、10 月 1 日各一次（5 年共 10 个），`specified_times` 为 2026-2030 年中秋和端午的 10 个公历日期。`month` 与 `day` 是独立过滤数组，会按笛卡尔积组合，上面的 `month: [1,10]`、`day: [1]` 恰好精确；一旦日期不在同一天，例如 2027 年中秋 9 月 15 日与国庆 10 月 1 日写成 `month: [9,10]`、`day: [1,15]`，会多出 9 月 1 日和 10 月 15 日。规则和 `specified_times` 都应限定在已查询节假日数据的年份内，不能把中秋的公历日期作为跨年固定规则复用；只有 Handler、`task_code` 或状态需要独立管理时才拆成多个共享 `group_id` 的 CronJob。

### 中秋等农历浮动节日

中秋节是农历八月十五，对应的公历日期每年变化，不能写成固定的 `month: [9]`、`day: [15]`。该写法只表示每年公历 9 月 15 日。

调用方应先通过 `ListHolidayFestivals` 或 `GetHolidayFestival` 获取目标年份中秋节的公历日期，再创建一个有界单次 CronJob。例如节假日数据返回某年中秋节为 `2027-09-15` 时：

```json
{
  "task_code": "demo-mid-autumn-2027",
  "task_name": "2027 中秋节任务",
  "type": "holiday",
  "group_id": "major-holidays",
  "description": "按节假日数据在 2027 年中秋节 09:00 执行一次",
  "start_time": "2027-09-15 09:00:00",
  "end_time": "2027-09-15 09:00:00",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"holiday\":\"mid-autumn\",\"year\":2027}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

下一年度应根据新的节假日数据创建或提交对应年份的任务，不要用固定公历日期长期复用。

### 每月 1 日 09:00 执行一次

```json
{
  "task_code": "demo-monthly-day-1",
  "task_name": "每月 1 日执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每月 1 日 09:00 执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2028-12-31 23:59:59",
  "rule": {
    "freq": 1,
    "month": [],
    "day": [1],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 10,
  "payload": "{\"reportType\":\"monthly\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每月最后一个日历日 23:30 执行一次

`day: [-1]` 表示当月最后一个日历日，会随月份长度匹配 28、29、30 或 31 日。它包含周末和法定节假日，不表示最后一个工作日。

```json
{
  "task_code": "demo-month-end",
  "task_name": "每月最后一日执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每月最后一个日历日 23:30 执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2028-12-31 23:59:59",
  "rule": {
    "freq": 1,
    "month": [],
    "day": [-1],
    "week": [],
    "hours": [23],
    "minutes": [30]
  },
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"reportType\":\"month-end\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每周一、周三、周五 09:00 执行一次

```json
{
  "task_code": "demo-weekdays-135",
  "task_name": "每周一三五执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每周一、周三、周五 09:00 执行",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2027-12-31 23:59:59",
  "rule": {
    "freq": 2,
    "month": [],
    "day": [],
    "week": [1, 3, 5],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 10,
  "payload": "{\"shift\":\"morning\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每周一至周五 09:00 执行一次

该规则按日历星期匹配周一至周五，不识别法定工作日：工作日中的法定节假日仍会命中，周末调休工作日不会命中。需要按节假日调整时，应由调用方维护具体排除日期或在外部业务中处理。

```json
{
  "task_code": "demo-calendar-weekdays",
  "task_name": "周一至周五执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "每周一至周五 09:00 执行，不含法定工作日语义",
  "start_time": "2027-01-01 00:00:00",
  "end_time": "2027-12-31 23:59:59",
  "rule": {
    "freq": 2,
    "month": [],
    "day": [],
    "week": [1, 2, 3, 4, 5],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 10,
  "payload": "{\"shift\":\"calendar-weekday-morning\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每天多个固定时间

`hours` 与 `minutes` 按笛卡尔积组合。下面的规则片段每天只在 09:30 和 17:30 执行：

```json
{
  "freq": 3,
  "month": [],
  "day": [],
  "week": [],
  "hours": [9, 17],
  "minutes": [30]
}
```

如果配置 `hours: [9, 17]`、`minutes: [0, 30]`，实际会得到 09:00、09:30、17:00、17:30 四个时间点。要在一组已知日期精确表达 09:00 和 17:30 两个不同分钟的时间对，可以用 RRULE 生成 09:00，再通过 `specified_times` 逐项加入这些日期的 17:30。`specified_times` 不是每日时钟模板；若要在很长范围内每天都执行这两个不规则时间对，应调整可表达的规则或拆分任务。仅当两个时刻需要不同 Handler、任务身份或状态时，才必须拆成两个 CronJob。

### 更多受支持的规则变体

以下内容是 `rule` 字段的紧凑写法；用于创建时仍需放入完整请求，并提供非空 `task_code`、`task_name`、`type`、`dept_code`、显式生效区间及其他创建字段。

| 场景 | `rule` 片段 | 说明 |
| --- | --- | --- |
| 每年 12 月 31 日 18:00 | `{"freq":0,"month":[12],"day":[31],"week":[],"hours":[18],"minutes":[0]}` | 只在请求生效区间内匹配 |
| 每季度末最后一个日历日 23:00 | `{"freq":0,"month":[3,6,9,12],"day":[-1],"week":[],"hours":[23],"minutes":[0]}` | 固定选择季末月份；若要从开始时间起滚动每 3 个月，应改用 `{"freq":1,"day":[-1],"interval":3}` 形态；也不是最后一个工作日 |
| 每 3 个月 5 号 09:00（从开始时间起算） | `{"freq":1,"month":[],"day":[5],"week":[],"hours":[9],"minutes":[0],"interval":3}` | 发生月份序列锚定任务开始时间，与固定 `month` 集合语义不同 |
| 每月 1 日和 15 日 10:00 | `{"freq":1,"month":[],"day":[1,15],"week":[],"hours":[10],"minutes":[0]}` | 两个日期分别形成候选点 |
| 每周六、周日 08:00 | `{"freq":2,"month":[],"day":[],"week":[6,7],"hours":[8],"minutes":[0]}` | 表示日历周末，不包含节假日判断 |

日期过滤不会自动修正不存在的日期。例如 `day: [31]` 在 4 月没有候选点，会直接跳过，而不会顺延或收敛到 4 月 30 日；每年 2 月 29 日也只在闰年命中，不会平移到 2 月 28 日。确实需要每月最后一天时应使用 `day: [-1]`。

### 指定日期区间内每天 09:00 执行

该示例还通过 `exclude_dates` 排除 2027-05-03 的候选点。

```json
{
  "task_code": "demo-bounded-daily",
  "task_name": "限定区间每日执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "2027-05-01 至 2027-05-07 每天 09:00 执行，排除 5 月 3 日",
  "start_time": "2027-05-01 00:00:00",
  "end_time": "2027-05-07 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": ["2027-05-03"],
  "priority": 10,
  "payload": "{\"campaignId\":\"demo-week\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 在固定时间只执行一次

将 `start_time` 与 `end_time` 设为同一秒，并使规则命中该秒。下面的有界 DAILY 规则只有 `2027-06-18 14:30:00` 一个候选执行点，完成后不再产生下一周期。

```json
{
  "task_code": "demo-one-time",
  "task_name": "固定时间单次执行示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "仅在 2027-06-18 14:30:00 执行一次",
  "start_time": "2027-06-18 14:30:00",
  "end_time": "2027-06-18 14:30:00",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [14],
    "minutes": [30]
  },
  "exclude_dates": [],
  "priority": 10,
  "payload": "{\"action\":\"one-time-demo\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

如果该唯一候选点早于创建时间且 `skip_time_filter` 为 `false`，规则已经耗尽，响应中的 `next_run` 为空。需要创建后补触发这个过去候选点时，可以设置 `skip_time_filter: true`，并将 `max_delay` 配置为足以覆盖候选点到创建时刻的延迟；否则调度器可能跳过 Handler 并直接耗尽规则。

## 不直接支持的调度语义

下列需求无法由当前 `PlanRulePb` 直接表达，需要拆分 CronJob、预先计算具体日期，或交给外部业务逻辑处理：

| 需求 | 当前限制与处理建议 |
| --- | --- |
| 从任意锚点滚动每 N 天、周、月、年、小时 | 已支持 `interval: N`（锚定开始时间）；每 N 分钟仍用固定 `minutes` 列表（`interval` 与 `minutes` 交互见[规则字段](#规则字段)） |
| 精确执行 N 次后停止 | 请求没有 `count`；只能用包含边界的 `end_time` 限定区间，不能保证生成数量恰为 N |
| 直接填写序号星期，或从候选集合取第 N 个 | `week` 只接收普通星期值，没有序号或 `BYSETPOS` 字段。部分固定序号星期可改用 `day` 与 `week` 的交集表达，例如每月第一个周一使用 `day: [1,2,3,4,5,6,7]` 和 `week: [1]`；通用候选排序与选位仍不支持 |
| 法定工作日、动态节假日、最后一个工作日 | CronJob 规则不接入动态节假日日历；`exclude_dates` 只是调用方提交的静态日期，不会识别调休工作日 |
| 节假日或周末自动前移、后移 | 排除日期只会删除原候选点，不会生成替代日期；替代执行日需由外部计算和管理 |
| 只用 `rule` 字段精确表达 09:00 和 17:30 | `hours` 与 `minutes` 会形成笛卡尔积；可用 RRULE 表达一个稳定基线，并通过 `specified_times` 逐项加入已知日期的另一个精确时刻 |
| 不提供 RRULE、只提交精确时间 | `rule` 必填；当前不支持纯 RDATE-only 请求，应提供合法 RRULE 并用 `specified_times` 补充候选 |
| 排除一个连续时间段 | 当前只支持 `excluded_times` 的单秒排除和 `exclude_dates` 的上海自然日排除，不支持任意时间范围排除 |
| 将每月 29、30、31 日自动收敛到月末 | 不存在的日期会跳过；仅当业务语义就是最后一个日历日时使用 `day: [-1]` |
| 单条规则覆盖超过 100 年或永久不设结束时间 | CronJob 生效区间最多 100 年；省略 `end_time` 只会默认到开始时间所在年份年末，不会生成无界规则 |

## 创建后立即补触发一次

“创建后立即执行”不是调用 `RunCronJob`，而是在 `CreateCronJob` 请求中设置 `skip_time_filter: true`。Trigger 会在规则范围内选择不晚于当前时间的最近一个候选作为首次 `next_run`；最多补一个已经发生的计划点，不会追赶全部历史周期。该计划点进入调度时仍会检查 `max_delay`：若相对当前时间的延迟超过有效容忍值，调度器会跳过 Handler 并推进到下一周期。因此需要按规则频率配置足以覆盖最近候选点的 `max_delay`。补触发完成后，任务仍按原 RRULE 继续执行。

下面示例在 2026 年每天 09:00 执行。若创建时范围内已有过去的 09:00 候选，首次调度会选择最近一个过去候选尽快补触发；若还没有过去候选，则仍使用未来的首次计划点。示例将 `max_delay` 设为 86400 秒，可覆盖每日规则中距当前时间不到 24 小时的最近候选点。

```json
{
  "task_code": "demo-create-and-catch-up",
  "task_name": "创建后补触发示例",
  "type": "demo",
  "group_id": "cron-guide",
  "description": "创建后最多补一个过去计划点，之后每天 09:00 继续执行",
  "start_time": "2026-01-01 00:00:00",
  "end_time": "2026-12-31 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": [],
  "priority": 20,
  "payload": "{\"action\":\"catch-up-demo\"}",
  "lock_timeout": 120000,
  "max_delay": 86400,
  "skip_time_filter": true,
  "dept_code": "DEMO"
}
```

创建响应示例：

```json
{
  "job_id": "0198-demo-cron-job-id",
  "next_run": "2026-08-11 09:00:00",
  "group_id": "cron-guide"
}
```

`next_run` 是本次规则编译得到的首次计划点。任务创建成功与 Handler 业务处理成功是两个不同阶段。

## 预览未来计划时间

`PreviewCronJobSchedule` 接收 `job_id` 和可选 `count`。`count` 为 `0` 时默认返回最多 10 条，显式值最大为 100；规则耗尽时 `execution_times` 为空。预览严格从请求处理时刻之后开始，禁用任务也可以查询。

响应中的 `rrule_str` 是当前持久化并用于调度的完整 RRULE Set，`schedule_description` 和 `execution_times` 均基于该字符串生成。预览会自然应用其中的 `DTSTART`、`UNTIL`、BY*、`RDATE` 和 `EXDATE`，不会从管理视图的 `rule`、`start_time`、`end_time`、`specified_times`、`excluded_times` 或 `exclude_dates` 重新编译，也不会修改 `next_run`、启停状态、运行历史或 lease。

请求示例：

```json
{
  "job_id": "0198-demo-cron-job-id",
  "count": 5
}
```

响应示例：

```json
{
  "job_id": "0198-demo-cron-job-id",
  "task_code": "demo-every-minute",
  "execution_times": ["2026-08-11 10:01:00", "2026-08-11 10:02:00"],
  "schedule_description": "按分钟生成候选，并在每分钟的 00 秒执行",
  "rrule_str": "DTSTART;TZID=Asia/Shanghai:20260101T000000\nRRULE:FREQ=MINUTELY;UNTIL=20261231T155959Z;BYSECOND=0"
}
```

## 已有任务人工执行一次

`RunCronJob` 必须使用已经创建且未删除任务的 `job_id`。它异步调用同一个 Handler，不创建新任务，也不改变原周期 `next_run`、启停状态或 `last_scheduled_run`；只有 Handler 成功时才更新 `last_run`。即使任务处于禁用状态，也可以人工执行。

请求：

```json
{
  "job_id": "0198-demo-cron-job-id"
}
```

响应：

```json
{
  "trace_id": "4f6f9b724a0e47f2aeb09f44df4f02de"
}
```

`trace_id` 用于追踪异步执行过程；未接入 OpenTelemetry 时可能为空字符串。RPC 返回表示异步执行请求已受理，不表示业务 Handler 已经成功完成。

## 对接检查

- 确认 `task_code` 在业务侧稳定且全局唯一，并保存创建响应中的 `job_id`。
- 需要组织多个单规则 CronJob 时复用同一个 `group_id`；未传时保存响应中 Trigger 自动生成的 UUID。
- 确认 `task_name`、`type`、`dept_code` 非空，`hours` 和 `minutes` 均至少包含一个合法元素。
- 显式填写 `start_time` 与 `end_time`，并检查上海时区、边界顺序和 CronJob 100 年跨度。
- 确认 `payload` 是空字符串或合法 JSON 字符串，而不是直接嵌套的 JSON 对象。
- 需要创建后补一次时设置 `skip_time_filter: true`；需要人工执行已有任务时调用 `RunCronJob`。
- 不把创建响应、人工执行响应或 gRPC 调用成功当作业务完成，下游按计划点实现幂等。
