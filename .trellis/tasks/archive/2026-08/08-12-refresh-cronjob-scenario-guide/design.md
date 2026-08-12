# Trigger RRULE RDATE Design

## Boundaries

Plan 与 CronJob 继续共用 `PlanRulePb` 和 RRULE Set 编译函数。新增能力只扩展候选集合为 `RRULE ∪ RDATE - EXDATE`，不改变 Plan 状态机、CronJob lease、Handler 或回执语义。

## API Contract

相关 Plan/CronJob 请求新增 `repeated string specified_times` 和 `repeated string excluded_times`，每项格式为 `yyyy-MM-dd HH:mm:ss`，按 `Asia/Shanghai` 解析。前者编译为 RDATE，后者编译为 EXDATE。CronJob 管理视图返回两份原始配置；Plan 以完整 `rrule_str` 保存和审计最终 Set。

每个列表最多 1000 项；空列表保持现有行为。现有 `exclude_dates` 继续表示整日排除。

`specified_times` / `excluded_times` 使用精确时间而不是纯日期，因此同一天可以配置多个不同时间，也不依赖 `rule.hours × rule.minutes` 展开。`excluded_times` 不是起止时间范围。

## Compile Flow

1. 按现有模式规范化 `start_time/end_time`，Plan 最大 3 年，CronJob 最大 100 年。
2. 创建 RRULE 并写入 Set。
3. 解析每个 `specified_times`，截断到秒，校验位于 `[start_time, end_time]`，再调用 `Set.RDate`。
4. 解析每个 `excluded_times`，执行相同范围校验，再调用 `Set.ExDate`。
5. 解析 `exclude_dates`。除现有规则时刻外，对落在排除日期上的指定时间精确写入 `Set.ExDate`，保证整日排除语义。
6. 使用 Set 计算首次时间、Plan 展开、CronJob 推进、预览和描述。

`rrule-go.Set.Iterator` 自带排序和相同时间去重；EXDATE 对 RRULE/RDATE 合并结果按精确时间排除。服务端范围校验不能省略，因为库不会用 RRULE 的 DTSTART/UNTIL 裁剪 RDATE。

## Persistence

CronJob 模型新增可空 JSON 文本列保存原始 `specified_times` 和 `excluded_times`，转换层在 `CronJobExtra` 与模型列之间平铺/重建，Create/Update/Submit 可替换或清空列表。最终 `rrule_str` 同时持久化 RDATE/EXDATE，是调度和预览的权威来源。

Plan 不增加运行时调度列；`plan.rrule_str` 保存包含 RDATE 的完整 Set，创建时的 `set.All()` 产生对应 Batch/ExecItem。若管理视图需要回显原始输入，则从请求持久化字段或完整 Set 读取，不重新编译规则。

## Compatibility

新增 repeated 字段使用未占用字段号。旧客户端不传时行为不变；旧服务忽略新字段属于滚动升级限制。生成文件只能通过 `app/trigger/gen.sh` 更新。

## Error Behavior

- 任一精确时间格式错误 -> 参数错误。
- 任一精确时间早于开始或晚于结束 -> 参数错误。
- 重复指定时间，或与 RRULE 同时命中 -> 接受，最终 Set 去重。
- 指定时间同时出现在 `excluded_times`，或落在 `exclude_dates` -> 接受配置，但最终不产生该候选。
- 清空 CronJob 两个列表 -> 更新后 Set 不再包含对应旧 RDATE/EXDATE；仍遵循在途更新拒绝规则。

## Documentation

功能完成后把现有指南升级为 Plan/CronJob 两种 RRULE 模式的统一场景指南，新增多个指定时间、精确时间排除、与周期混合、去重、整日排除和范围限制示例。

## Risks And Rollback

- 长 RDATE 列表会放大 Set 字符串和 Plan 展开数量；Plan 继续受现有 5000 执行项上限保护，API 应设置合理列表上限。
- 生产数据库需要为 CronJob 原始 rdates 列执行迁移；回滚代码时可保留可空列。
- 回滚 Proto/Logic/模型时必须整体撤销 RDATE 输入与回显，不能留下只保存但不参与 Set 的字段。
