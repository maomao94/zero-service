# crontask 无效区间过滤重构：推进逻辑收编进 common/crontask，业务侧只提供谓词

## Goal

重构 `common/crontask` 的无效时间过滤机制：过滤语义从「业务侧返回下一个有效时间」改为「业务侧只提供判定谓词，推进跳过逻辑由公共层在单趟迭代中完成」。同步 ispagent（唯一使用方）改动。

## Requirements

1. `common/crontask` 的 `InvalidTimeFilter`（`func(task, next) time.Time`）改为谓词 `InvalidTimePredicate`（`func(task *TaskConfig, t time.Time) bool`，true = 该时刻无效需跳过）。
2. 推进跳过逻辑全部收编进公共层：新增核心函数 `NextRunsFiltered(value, after, count, invalid)`，解析 + `ShiftSetForQuery` 平移只做一次，用单个迭代器顺序收集，跳过 ≤ after 和被谓词判无效的候选；耗尽返回已收集结果。
3. `NextRuns`、`NextAfter` 改为委托 `NextRunsFiltered`（nil 谓词），语义不变。
4. `Scheduler.computeNextRun` 改为方法并在内部应用谓词推进；删除 crontask.go 完成路径的后置过滤调用（现 :188-195）。
5. `Scheduler.PreviewNextRuns` 委托统一的 `nextRuns` 助手，谓词逐候选介入；删除「filter 返回不前进即报错」守卫。
6. ispagent：删除 `skipInvalidTime`，新增 `invalidTimePredicate`（无效区间判定谓词）；`CalcInitNextRun` 一次调用 `NextRunsFiltered`；`NewInvalidTimeFilter` 改为 `NewInvalidTimePredicate`，谓词无状态（每次调用反序列化 Extra），不引入缓存。
7. 无效区间判定语义不变：`is <= t <= ie`（两端闭区间）；窗口字段缺失/解析失败 = 无过滤。
8. 同步更新 `.trellis/spec/backend/crontask-guidelines.md` 中关于 `InvalidTimeFilter` 的描述与反模式条目。
9. 工具方法整理：把基于 rrule-go 的重复封装与官方包不支持的能力抽到新公共包 `common/rrulex`，`common/crontask` 只保留调度器职责（Scheduler/Store/config/options/lease），rrule 查询与描述全部委托 `rrulex`：
   - `rrulex.ParseSet`（完整 Set 解析+校验，返回官方 `*rrule.Set`）、`rrulex.Validate`
   - `rrulex.ShiftSetForQuery`（官方包缺失的平移优化，保留导出）
   - `rrulex.NextRuns(value, after, count, invalid)` 唯一批量迭代封装（含谓词过滤，count=1 即「下一个有效点」）；**不再提供 NextAfter 封装**——单次取 after 直接用官方包 `ParseSet` + `set.After(after, false)` 的原生风格
   - `rrulex.Describe`（原 DescribeRRule）+ `ErrUnsupportedDescription`
10. 调用方全部切换：ispagent（db_store/task_rule/helper）、trigger（db_store/schedule/四个 logic）、common/crontask 内部（memory_store/computeNextRun/PreviewNextRuns）不再直接调用 crontask 包的 rrule 工具；单次 after 场景（db_store Enable、memory_store Enable）改为 `rrulex.ParseSet` + 官方 `set.After`。

## Acceptance Criteria

- [ ] `go build ./...` 通过。
- [ ] `go test ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/internal/cronjob` 通过（含 -race）。
- [ ] 调度完成路径：候选落在无效区间时，单次解析+迭代推进到区间后第一个有效点；区间内规则耗尽返回零值。
- [ ] `PreviewNextRuns`：谓词跳过候选后继续迭代，`count` 只统计最终接受的有效点；窗口内耗尽提前返回已收集结果。
- [ ] `CalcInitNextRun` 对跳过整周窗口的任务返回下周首个有效点；无窗口时返回原候选。
- [ ] 解析错误只发生一次并向上传播，不再被吞成零值。
- [ ] ispagent `NewInvalidTimePredicate` 无状态、并发安全，按 Extra 逐次判定。
- [ ] 相关测试改写/新增完成，无对旧 `InvalidTimeFilter` 签名或 `skipInvalidTime` 的引用。
- [ ] `common/crontask` 不再含 rrule 解析/平移/迭代/描述代码；`common/rrulex` 包形成并通过全部迁移后的测试。
- [ ] 全仓无 `crontask.(NextAfter|NextRuns|NextRunsFiltered|ValidateRRule|DescribeRRule|ShiftSetForQuery)` 业务引用；rrule 级测试迁移到 `common/rrulex`。
- [ ] crontask-guidelines.md 与实现一致。

## Notes

- 纯重构，不保留旧 API 兼容；旧能力「过滤器返回任意时间（重映射）」被有意收窄为「只能排除」——唯一使用方 ispagent 只做排除，无影响。
- 谓词只做排除，不得返回时间；公共层保证单调推进。
