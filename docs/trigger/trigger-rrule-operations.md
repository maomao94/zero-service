# Trigger CronJob 运行与对接

创建之后的运行操作：补触发、计划预览、人工执行与上线前检查。规则写法见[场景示例](./trigger-rrule-api-guide.md)，字段与校验边界见[规则与字段参考](./trigger-rrule-syntax.md)，服务能力总览见 [Trigger 服务](./trigger.md)。

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
