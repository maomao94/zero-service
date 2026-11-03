# Trigger Plan/CronJob RRULE 场景示例

本文给出可直接用于构造 gRPC JSON 请求的规则示例，覆盖分钟级、日历、节假日、区间与一次性场景。规则字段与校验边界见 [规则与字段参考](./trigger-rrule-syntax.md)，服务能力总览见 [Trigger 服务](./trigger.md)。

> 约定：示例 JSON 一律使用 snake_case 字段名，与 proto 字段名一致；以下日期仅用于展示规则，实际接入时应替换 `task_code`、业务字段与生效区间。

创建之后如何补触发、预览计划时间、人工执行一次，见[运行与对接](./trigger-rrule-operations.md)。

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

`MINUTELY` 的 `interval` 与 `minutes` 不应叠加（原因见[规则字段](./trigger-rrule-syntax.md#规则字段)中 MINUTELY + interval 的说明），因此用固定分钟列表表达：

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

如果该唯一候选点早于创建时间且 `skip_time_filter` 为 `false`，规则已经耗尽，响应中的 `next_run` 为空。需要创建后补触发这个过去候选点时，可以设置 `skip_time_filter: true`，并将 `max_delay` 配置为足以覆盖候选点到创建时刻的延迟；否则调度器可能跳过 Handler 并直接耗尽规则。具体做法见[创建后立即补触发一次](./trigger-rrule-operations.md#创建后立即补触发一次)。
