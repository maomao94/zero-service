# Trigger CronJob API 场景指南

本文面向 Trigger CronJob 对接方，给出可直接用于构造 gRPC JSON 请求的周期任务创建和人工执行示例。RPC 字段与校验以 [`trigger.proto`](../app/trigger/trigger.proto) 为准，服务能力总览见 [Trigger 服务](./trigger.md)。

## API 边界

| API | 定位方式 | 用途 | 关键行为 |
| --- | --- | --- | --- |
| `CreateCronJob` | 请求中的全局唯一 `task_code` | 创建一个新 CronJob | `task_code` 已存在时返回重复记录错误；成功后返回 Trigger 生成的 `job_id` 和首次 `next_run` |
| `SubmitCronJob` | 稳定 `task_code` | 幂等提交完整配置 | 有效任务不存在时创建，存在时更新并保留原 `job_id` 和启停状态 |
| `RunCronJob` | 已创建任务的 `job_id` | 人工异步执行一次 | 不创建任务，不修改周期 `next_run`、启停状态或 `last_scheduled_run`；返回 `trace_id` 用于追踪 |
| `PreviewCronJobSchedule` | 已创建任务的 `job_id` | 预览未来计划时间 | 直接读取持久化 RRULE Set，应用排除日期和调度器时间过滤；不修改任务或执行 Handler |

本文的周期示例均使用 `CreateCronJob` 请求体。需要由上游反复提交同一个业务任务配置时，可以将相同配置字段用于 `SubmitCronJob`；两者都会通过同一规则编译逻辑生成 RRULE Set 和 `next_run`。

创建成功只表示任务已经持久化并返回首次计划时间，不表示异步业务回调已经成功。调度 lease 过期、网络错误等情况可能使同一计划点再次下发，下游应结合 `job_id`、`task_code` 和计划时间实现幂等。

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

Trigger 会把以上字段映射为 RRULE 的 `BYMONTH`、`BYMONTHDAY`、`BYDAY`、`BYHOUR` 和 `BYMINUTE`，并固定加入 `BYSECOND=0`。因此所有示例都在第 0 秒执行，请求中没有秒字段。

`month`、`day`、`week`、`hours` 和 `minutes` 都是过滤条件。要表达全天每分钟或每小时执行，需要像下文示例一样覆盖全部小时；只传 `hours: [0]` 会把执行时间限制在 00 点。

当前请求没有 `interval` 字段，不能表达“从任意开始月份起滚动每隔三个月”。“每三个月”只能按固定月份集合表达，例如每年 1、4、7、10 月执行。

### 开始与结束时间

规则统一使用 `Asia/Shanghai` 时区。

| 项目 | 说明 |
| --- | --- |
| 字段 | `start_time`、`end_time` 均可省略或传空字符串 |
| 格式 | `yyyy-MM-dd HH:mm:ss` |
| 生效边界 | 开始和结束时间都包含在规则范围内 |
| 相等时间 | `start_time == end_time` 合法，可构造只有一个候选点的固定时间单次任务 |
| 错误顺序 | `end_time < start_time` 时请求失败 |
| 最大跨度 | `end_time` 不能晚于 `start_time` 加 3 年 |

省略字段时按以下方式补齐：

| `start_time` | `end_time` | 编译 RRULE 时使用的范围 |
| --- | --- | --- |
| 空 | 空 | 当前上海时间所在年份的 `01-01 00:00:00` 至 `12-31 23:59:59` |
| 有值 | 空 | 指定开始时间至该开始时间所在年份的 `12-31 23:59:59` |
| 空 | 有值 | 当前上海年份的 `01-01 00:00:00` 至指定结束时间；若结束时间早于默认开始时间则失败 |
| 有值 | 有值 | 使用调用方指定范围，并执行顺序与 3 年跨度校验 |

默认值会写入实际用于调度的 RRULE Set，但管理视图中的 `start_time`、`end_time` 保留调用方原始输入；调用方传空字符串时，查询结果仍可能为空，而 `rrule_str` 已包含补齐后的边界。生产配置建议显式填写两个字段，避免任务范围随创建年份变化。下文所有完整创建示例均显式填写范围。

### 创建配置字段

| 字段 | 语义 |
| --- | --- |
| `task_code` | 调用方提供的全局唯一稳定业务编码，最长 64 字符；与 Trigger 生成的 `job_id` 不是同一个标识 |
| `task_name` | 任务名称，必填，最长 128 字符 |
| `type` | 业务任务类型，必填，最长 64 字符 |
| `group_id` | 可选业务分组 ID，最长 64 字符 |
| `description` | 可选描述，最长 200 字符 |
| `dept_code` | 机构编码，必填，最长 64 字符 |
| `exclude_dates` | 排除日期数组，格式 `yyyy-MM-dd`；Trigger 会排除该日规则中全部小时和分钟组合 |
| `priority` | 非负整数；到期任务选择时数字越大优先级越高 |
| `payload` | 传给业务 Handler 的参数字符串；为空表示无参数，非空时必须是合法 JSON 字符串 |
| `extra` | 调用方扩展数据；为空表示无扩展，非空时必须是合法 JSON 字符串 |
| `lock_timeout` | 单次调度 lease 超时，单位毫秒；`0` 使用调度器默认值，实际有效锁时长不会低于 30 秒 |
| `max_delay` | 最大延迟容忍，单位秒；`0` 使用调度器默认值，超过该延迟时跳过本次并计算下一计划点 |
| `skip_time_filter` | 只影响首次 `next_run`；`true` 时最多选择一个已经发生的计划点用于尽快补触发，不追赶全部历史周期；该计划点仍受 `max_delay` 检查 |

`payload` 和 `extra` 的 proto 类型是字符串，因此 JSON 请求中需要把业务 JSON 转义为字符串，例如 `"payload": "{\"orderId\":\"demo-001\"}"`，不能直接传一个 JSON 对象。

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
  "extra": "{\"scenario\":\"every-minute\"}",
  "lock_timeout": 60000,
  "max_delay": 120,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每 10 分钟第 0 秒执行一次

当前 API 没有 `interval`，因此使用固定分钟集合 `0,10,20,30,40,50`。

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
  "extra": "{\"scenario\":\"every-10-minutes\"}",
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
  "extra": "{\"scenario\":\"hourly\"}",
  "lock_timeout": 60000,
  "max_delay": 300,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

### 每年固定季度月份 1 日 09:00 执行

该示例固定在每年 1、4、7、10 月执行，常用于表达当前 API 下的“每三个月”。它不是通用的滚动三个月间隔，因为请求没有 `interval` 字段。

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
  "extra": "{\"scenario\":\"fixed-quarter-months\"}",
  "lock_timeout": 120000,
  "max_delay": 1800,
  "skip_time_filter": false,
  "dept_code": "DEMO"
}
```

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
  "extra": "{\"scenario\":\"monthly-day-1\"}",
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
  "extra": "{\"scenario\":\"last-calendar-day\"}",
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
  "extra": "{\"scenario\":\"monday-wednesday-friday\"}",
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
  "extra": "{\"scenario\":\"monday-to-friday\"}",
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

如果配置 `hours: [9, 17]`、`minutes: [0, 30]`，实际会得到 09:00、09:30、17:00、17:30 四个时间点。要精确表达 09:00 和 17:30 两个不同分钟的时间对，需要创建两个 CronJob，不能用一个规则合并。

### 更多受支持的规则变体

以下内容是 `rule` 字段的紧凑写法；用于创建时仍需放入完整请求，并提供非空 `task_code`、`task_name`、`type`、`dept_code`、显式生效区间及其他创建字段。

| 场景 | `rule` 片段 | 说明 |
| --- | --- | --- |
| 每年 12 月 31 日 18:00 | `{"freq":0,"month":[12],"day":[31],"week":[],"hours":[18],"minutes":[0]}` | 只在请求生效区间内匹配 |
| 每季度末最后一个日历日 23:00 | `{"freq":0,"month":[3,6,9,12],"day":[-1],"week":[],"hours":[23],"minutes":[0]}` | 固定选择季末月份，不是从任意锚点开始的 `interval=3`；也不是最后一个工作日 |
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
  "extra": "{\"scenario\":\"bounded-daily\"}",
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
  "extra": "{\"scenario\":\"one-time\"}",
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
| 从任意锚点滚动每 N 天、N 小时或 N 分钟等 | 请求没有 `interval`；枚举固定月份、小时或分钟只表示固定日历位置，不等价于通用滚动间隔 |
| 精确执行 N 次后停止 | 请求没有 `count`；只能用包含边界的 `end_time` 限定区间，不能保证生成数量恰为 N |
| 直接填写序号星期，或从候选集合取第 N 个 | `week` 只接收普通星期值，没有序号或 `BYSETPOS` 字段。部分固定序号星期可改用 `day` 与 `week` 的交集表达，例如每月第一个周一使用 `day: [1,2,3,4,5,6,7]` 和 `week: [1]`；通用候选排序与选位仍不支持 |
| 法定工作日、动态节假日、最后一个工作日 | CronJob 规则不接入动态节假日日历；`exclude_dates` 只是调用方提交的静态日期，不会识别调休工作日 |
| 节假日或周末自动前移、后移 | 排除日期只会删除原候选点，不会生成替代日期；替代执行日需由外部计算和管理 |
| 在一个规则中精确表达 09:00 和 17:30 | `hours` 与 `minutes` 会形成笛卡尔积，应拆成两个 CronJob |
| 将每月 29、30、31 日自动收敛到月末 | 不存在的日期会跳过；仅当业务语义就是最后一个日历日时使用 `day: [-1]` |
| 单条规则覆盖超过 3 年或永久不设结束时间 | 生效区间最多 3 年；省略 `end_time` 只会默认到开始时间所在年份年末，不会生成无界规则 |

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
  "extra": "{\"scenario\":\"create-and-catch-up\"}",
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
  "next_run": "2026-08-11 09:00:00"
}
```

`next_run` 是本次规则编译得到的首次计划点。任务创建成功与 Handler 业务处理成功是两个不同阶段。

## 预览未来计划时间

`PreviewCronJobSchedule` 接收 `job_id` 和可选 `count`。`count` 为 `0` 时默认返回最多 10 条，显式值最大为 100；规则耗尽时 `execution_times` 为空。预览严格从请求处理时刻之后开始，禁用任务也可以查询。

响应中的 `rrule_str` 是当前持久化并用于调度的完整 RRULE Set，`schedule_description` 和 `execution_times` 均基于该字符串生成。预览会自然应用其中的 `DTSTART`、`UNTIL`、BY* 条件和 `EXDATE`，不会从管理视图的 `rule`、`start_time`、`end_time` 或 `exclude_dates` 重新编译，也不会修改 `next_run`、启停状态、运行历史或 lease。

请求示例：

```json
{
  "jobId": "0198-demo-cron-job-id",
  "count": 5
}
```

响应示例：

```json
{
  "jobId": "0198-demo-cron-job-id",
  "taskCode": "demo-every-minute",
  "executionTimes": ["2026-08-11 10:01:00", "2026-08-11 10:02:00"],
  "scheduleDescription": "按分钟生成候选，并在每分钟的 00 秒执行",
  "rruleStr": "DTSTART;TZID=Asia/Shanghai:20260101T000000\nRRULE:FREQ=MINUTELY;UNTIL=20261231T155959Z;BYSECOND=0"
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
- 确认 `task_name`、`type`、`dept_code` 非空，`hours` 和 `minutes` 均至少包含一个合法元素。
- 显式填写 `start_time` 与 `end_time`，并检查上海时区、边界顺序和 3 年跨度。
- 确认 `payload`、`extra` 是空字符串或合法 JSON 字符串，而不是直接嵌套的 JSON 对象。
- 需要创建后补一次时设置 `skip_time_filter: true`；需要人工执行已有任务时调用 `RunCronJob`。
- 不把创建响应、人工执行响应或 gRPC 调用成功当作业务完成，下游按计划点实现幂等。
