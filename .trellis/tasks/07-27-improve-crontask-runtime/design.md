# CronTask 基础能力设计

## Boundary

公共能力位于 `common/crontask/`，不导入任何服务 proto、GORM model 或传输错误码。Trigger 仅在 `CalcPlanTaskDate` 边界接入该公共能力；具体数据库和业务回调接入后续单独处理。

## Execution Time Contract

`TaskConfig` 保留 `NextRun` 和 `LastRun`，新增 `ScheduledTime` 与 `LastScheduledRun`：

- `NextRun`：未 claim 时的下一次计划执行时间；Store claim 后暂存 lease 截止时间。
- `ScheduledTime`：当前 claim/retry 对应的原计划时间，handler 和下次时间计算使用该字段。
- `LastRun`：最近一次 handler 成功返回后的实际完成时间。
- `LastScheduledRun`：最近一次成功周期执行对应的原计划时间。

使用 `Completion` 结构替代 `Complete` 的多个时间位置参数：

```go
type Completion struct {
    NextRun         time.Time
    LastRun         time.Time
    LastScheduledRun time.Time
}
```

`TaskStore.Complete(ctx, id, expectedLockedUntil, completion)` 必须在 lease CAS 中原子提交结果。周期成功填充三个时间；MaxDelay 跳过只填 `NextRun`；失败不调用 Complete；`RunNow` 继续通过 `UpdateLastRun` 只写实际成功时间。

MemoryStore 作为契约参考实现，同时保留管理 `Update` 时的两个历史时间字段。

## Audit Logging

日志边界位于 Scheduler，因为它拥有 claim、handler、RRULE 计算和 Store 完成的完整状态。使用 context fields 携带：

- `task_id`
- `task_code`
- `scheduled_run`
- `locked_until`

生命周期事件：

1. claim 成功。
2. handler 开始。
3. handler 成功或失败，记录耗时。
4. stale 跳过或 ErrDeleteTask 删除结果。
5. 下次执行时间计算成功或失败。
6. completion CAS 成功或失败。
7. RunNow 排队、开始、成功或失败。

不记录 Payload、Extra 或完整 RRULE。轮询无任务不输出基础包日志；SQL 静默由适配器负责。

## RRULE Description

新增 `describe.go`，导出：

```go
func DescribeRRule(value string) (string, error)
```

### Parsing

1. 空输入直接返回空字符串。
2. 非空输入只接受包含 DTSTART 和 RRULE 的完整 RRULE Set。
3. 提取 `GetDTStart`、`GetRRule().Options/OrigOptions`、`GetRDate` 和 `GetExDate`。
4. 无 DTSTART、无 RRULE 或语法错误返回解析错误。
5. 展示时区使用 DTSTART location。

### Normalization

- `INTERVAL` 零值归一为 1。
- 所有数值列表排序、去重，不修改解析对象。
- 对低于对应时间粒度且未显式指定 BYHOUR/BYMINUTE/BYSECOND 的规则，仅在存在显式 DTSTART 时使用 DTSTART 补足显示时间。
- MONTHLY 未指定 BYMONTHDAY/BYDAY 时从 DTSTART 推导日期；WEEKLY 未指定 BYDAY 时从 DTSTART 推导星期；YEARLY 缺少日期过滤时按 rrule-go/RFC 的 DTSTART 默认语义描述。
- 时间列表是 BYHOUR × BYMINUTE × BYSECOND 的笛卡尔积；小集合展开，大集合按维度压缩。

### Rendering

按“周期主句 + 日期交集条件 + 时间 + 边界 + 额外/排除日期”组织中文。数组连接顺序稳定，负数月日使用“最后一天/倒数第 N 天”，带序号 BYDAY 使用“第 N 个/倒数第 N 个周X”。

支持能准确表达的 BYSETPOS 典型组合；其他 BYSETPOS、BYYEARDAY、BYWEEKNO、BYEASTER 等高级组合返回 `ErrUnsupportedDescription`。描述错误不影响现有日期计算 API。

### Exclusions And Filters

- RRULE Set 的 `EXDATE` 按完整 datetime occurrence 精确匹配；业务日期 `yyyy-MM-dd` 必须按该日实际 `hours × minutes × seconds` 展开。
- `EXDATE;VALUE=DATE` 在 rrule-go 中会成为当天 00:00，不能排除当天其他执行时间。
- 少量、明确的 Trigger `excludeDates` 使用 EXDATE 持久化，保证规则可独立重放和排障。
- 大范围或动态非法区间（如 ispagent）不得展开海量 EXDATE，使用 `InvalidTimeFilter` 在 Set 计算候选时间后循环跳过。
- 若 Trigger 后续支持长日期区间，应增加日期/区间过滤机制，而不是无限扩大 RRULE Set。

## Compatibility

功能尚未上线，本次不保留旧 `TaskStore.Complete` 签名，不增加双写或 deprecated wrapper。由于源码范围限制，外部 Store 适配器将在接入任务中迁移；本任务只验证 `common/crontask`。

## CalcPlanTaskDate Integration

`CalcPlanTaskDateRes` 新增 `string scheduleDescription = 2`。Logic 继续使用现有 `rrule.Set` 展开 `planDates`，在完成 RRULE 和 EXDATE 组装后调用 `crontask.DescribeRRule(set.String())`。因此日期与描述共享 DTSTART、UNTIL、BY* 和 EXDATE，不从 `PlanRulePb` 重写第二套中文规则。

描述错误映射为 Trigger 参数错误，避免返回日期成功但描述失真的部分结果。字段尚未上线，不增加兼容字段或旧名称。

## Configuration Views

- `Plan.RRuleStr` 保存创建时用于展开 Batch 日期的完整 Set 快照，仅用于审计和展示；Plan 调度仍由既有 Batch/ExecItem 状态机负责。
- `PlanPb.rruleStr` 与 `scheduleDescription` 从该快照返回，不从批次反推规则。
- `CronJobPb.scheduleDescription` 从 `TaskConfig.RRuleStr` 动态生成。
- `ispagent.TaskConfigItem.schedule_description` 从 `GormTaskConfig.RRuleStr` 动态生成。
- 描述不落库缓存，防止规则更新后文案漂移。
- Plan/PlanBatch/PlanExecItem 保持独立的预展开调度模型，不接入 TaskConfig/TaskStore/Scheduler；保存 Set 只是创建时的规则快照。

## Rollback

描述器是独立纯函数，可单独移除。时间契约回滚需要同时恢复 `TaskConfig`、`TaskStore`、Scheduler 和 MemoryStore；不涉及数据库迁移。
