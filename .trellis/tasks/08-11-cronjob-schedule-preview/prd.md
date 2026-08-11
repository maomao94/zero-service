# Add CronJob schedule preview API

## Goal

为 Trigger CronJob 提供按 `job_id` 预览未来若干计划执行时间的只读 RPC，使调用方能够核对当前持久化规则、排除日期和调度器非法时间过滤后的实际候选时间。

## Requirements

- 新增 `PreviewCronJobSchedule` RPC，按 Trigger `job_id` 查询已存在且未删除的 CronJob。
- 请求支持 `count`；`0` 默认返回 10 条，最大允许 100 条。
- 响应返回 `job_id`、`task_code`、持久化 `rrule_str`、规则描述和按升序排列的 `execution_times`，不返回规则总区间或展开全部日期。
- 使用持久化的完整 RRULE Set 计算，不能根据管理视图中的 `rule/start_time/end_time/exclude_dates` 重新编译。
- 未来时间必须严格晚于接口计算起点；逐次迭代直到达到 `count` 或规则耗尽，禁止使用 `set.All()`。
- RRULE Set 中的 `DTSTART`、`UNTIL`、BY* 条件和 `EXDATE` 必须自然生效。
- `common/crontask` 提供可复用的有界预览能力，并支持应用 Scheduler 当前配置的 `InvalidTimeFilter`，使预览与实际下一次时间计算保持一致。
- 接口仅只读：不得修改 `next_run`、启停状态、运行历史或 lease，不得执行 Handler 或写执行日志。
- 禁用任务仍可预览；任务不存在返回记录不存在；非法持久化规则返回明确错误；规则耗尽返回空列表。

## Acceptance Criteria

- [ ] `count=0` 返回至多 10 条，显式 `count` 返回至多指定数量，`count>100` 校验失败。
- [ ] 预览直接消费持久化 RRULE Set，并通过测试证明 `EXDATE` 被排除。
- [ ] 公共预览能力通过测试证明会应用 `InvalidTimeFilter`，且只迭代到数量上限。
- [ ] 禁用任务可预览，规则耗尽返回空列表，任务不存在和非法规则按契约返回错误。
- [ ] Proto 生成物、Trigger Server、Logic 和相关文档与契约一致。
- [ ] `go test ./common/crontask ./app/trigger/internal/cronjob ./app/trigger/internal/logic` 与 `git diff --check` 通过。

## Notes

- `InvalidTimeFilter` 是 RRULE Set 之外的调度策略；`EXDATE` 则属于 RRULE Set 自身，两者均需在预览结果中生效。
