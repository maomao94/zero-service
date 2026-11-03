# Trigger RRULE 规则与字段参考

本文说明 Trigger Plan 与 CronJob 的规则模型：候选集合如何编译、PlanRulePb 字段含义、开始/结束时间边界与更新字段限制。可直接复制到请求里的完整示例见 [场景示例](./trigger-rrule-api-guide.md)。

> 约定：示例 JSON 一律使用 snake_case 字段名，与 proto 字段名一致（如 `task_code`、`specified_times`、`exclude_dates`）。RPC 字段与校验以 [`trigger.proto`](../../app/trigger/trigger.proto) 为准。

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

两者返回的 `CronJobPb` 都直接回显持久化的 `specified_times` 和 `excluded_times`；数据库用可空 JSON 文本列保存原始列表，未配置或已清空时写入 SQL `NULL`，API 统一返回空列表。`rrule_str` 与 `schedule_description` 来自持久化完整 Set，不会从回显列表重新推导。Preview 请求和响应示例见[预览未来计划时间](./trigger-rrule-operations.md#预览未来计划时间)，其结果自然包含 RDATE 并应用精确与整日 EXDATE，且严格只读。

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

## 下一步

- 完整请求示例（分钟级、日历、节假日、区间、一次性等）：[场景示例](./trigger-rrule-api-guide.md)
- 服务能力与状态流转：[Trigger 服务](./trigger.md)
