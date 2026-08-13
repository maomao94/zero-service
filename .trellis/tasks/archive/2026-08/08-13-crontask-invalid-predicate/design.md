# 技术设计：crontask 无效区间过滤重构

## 背景问题

现设计：`InvalidTimeFilter func(task *TaskConfig, next time.Time) time.Time`——业务侧（ispagent `skipInvalidTime`）在循环里反复调用 `crontask.NextAfter(rruleStr, next)` 逐点推进。每次调用都重新执行 `parseQuerySet`（全量字符串解析 + `ShiftSetForQuery` 平移）+ `set.After` 重建迭代器，跳过窗口覆盖 N 个候选就是 N 次解析；且业务侧被迫了解 rrule 推进机制与耗尽语义；窗口内解析错误被 `return time.Time{}` 吞成零值，与「规则耗尽」混淆。

## 目标结构

```
业务侧（ispagent）                    公共层（common/crontask）
----------------                     ------------------------
invalidTimePredicate()  ──谓词──▶    NextRunsFiltered(value, after, count, invalid)
  t → bool (true=无效)                ├─ parseQuerySet 一次（解析+平移）
NewInvalidTimePredicate()             ├─ set.Iterator() 单趟推进
  task → 谓词（无状态，逐次反序列化） ├─ 跳过 dt ≤ after
                                      ├─ 跳过 invalid(dt) 候选
                                      └─ 耗尽返回已收集 runs
```

## 接口设计

### `common/crontask/query.go`

```go
// NextRunsFiltered 返回严格晚于 after 的至多 count 个有效计划时间。
// invalid 为 nil 或返回 false 表示该候选有效；true 表示跳过。
// 解析与平移只做一次，单迭代器顺序收集，不重复解析规则字符串。
func NextRunsFiltered(value string, after time.Time, count int, invalid func(time.Time) bool) ([]time.Time, error) {
    // count <= 0 → 空切片
    // parseQuerySet 一次
    // next := set.Iterator()；循环：
    //   dt, ok := next()；!ok → break
    //   !dt.After(after) → continue
    //   invalid != nil && invalid(dt) → continue
    //   append；after = dt
}

func NextRuns(value string, after time.Time, count int) ([]time.Time, error) {
    return NextRunsFiltered(value, after, count, nil)
}

func NextAfter(value string, after time.Time) (time.Time, error) {
    runs, err := NextRunsFiltered(value, after, 1, nil)
    if err != nil || len(runs) == 0 { return time.Time{}, err }
    return runs[0], nil
}
```

要点：
- 谓词跳过时**不**推进 `after` 游标：迭代器本身单调递增，后续候选必然 > 已跳过候选；只推进已接受结果，保证严格递增。
- `after = dt` 仅在 append 时更新（与现 `NextRuns` 一致），谓词跳过的候选不影响游标。

### `common/crontask/options.go`

```go
// InvalidTimePredicate 判断给定候选时间是否处于不可用区间，true 表示无效应跳过。
// 谓词只做排除；推进跳过由调度器在单趟迭代中完成。
type InvalidTimePredicate func(task *TaskConfig, t time.Time) bool

func WithInvalidTimePredicate(f InvalidTimePredicate) SchedulerOption
```

`SchedulerOptions.InvalidTimeFilter` 字段同步改名 `InvalidTimePredicate`。

### `common/crontask/crontask.go`

```go
// nextRuns 将调度器谓词绑定到 task，委托 NextRunsFiltered。
func (s *Scheduler) nextRuns(task *TaskConfig, after time.Time, count int) ([]time.Time, error) {
    var invalid func(time.Time) bool
    if s.invalidTimePredicate != nil {
        invalid = func(t time.Time) bool { return s.invalidTimePredicate(task, t) }
    }
    return NextRunsFiltered(task.RRuleStr, after, count, invalid)
}

// computeNextRun 从包级函数改为 Scheduler 方法。
func (s *Scheduler) computeNextRun(cfg *TaskConfig) (time.Time, error) {
    if cfg.RRuleStr == "" { return time.Time{}, nil }
    now := carbon.Now().StdTime()
    base := now
    if cfg.ScheduledTime.After(now) { base = cfg.ScheduledTime }
    runs, err := s.nextRuns(cfg, base, 1)
    if err != nil || len(runs) == 0 { return time.Time{}, err }
    return runs[0], nil
}
```

- 删除完成路径（现 :193-195）的后置 `invalidTimeFilter(task, nextRun)` 调用；`runAndComplete` 直接 `nextRun, err := s.computeNextRun(task)`。
- `PreviewNextRuns` 保留 nil scheduler/task/count 校验后：`return s.nextRuns(task, after, count)`。删除原迭代器循环、过滤块（:281-289）与非前进守卫。

### `app/ispagent/internal/crontask/task_rule.go`

```go
func NewInvalidTimePredicate() crontask.InvalidTimePredicate {
    return func(task *crontask.TaskConfig, t time.Time) bool {
        fields := DeserializeExtra(string(task.Extra))
        if fields == nil { return false }
        is := parseTime(fields.InvalidStartTime)
        ie := parseTime(fields.InvalidEndTime)
        if is.IsZero() || ie.IsZero() { return false }
        return !t.Before(is) && !t.After(ie)
    }
}
```

- 谓词无状态：每次调用反序列化 Extra（性能可接受，不引入缓存），天然并发安全。

- `invalidTimePredicate()` 从 `IspTaskFields` 生成同样判定逻辑的纯谓词（窗口缺失返回 nil）。
- `CalcInitNextRun`：`NextRunsFiltered(rruleStr, now, 1, pred)` 取首个。
- 删除 `skipInvalidTime`。
- `servicecontext.go:64`：`WithInvalidTimePredicate(ctask.NewInvalidTimePredicate())`。

## 边界与语义

| 场景 | 行为 |
| --- | --- |
| 无窗口/字段非法 | 谓词恒 false，行为同无过滤 |
| 候选落在窗口内 | 跳过，继续迭代；窗口覆盖整段剩余规则 → 耗尽 → 零值/空结果 |
| 解析错误 | `parseQuerySet` 失败向上返回 error，只发生一次 |
| 谓词被并发调用 | 谓词无状态、无共享可变数据，天然并发安全 |
| `count` 语义 | 只统计最终接受的有效点（不变） |
| 非连续/多窗口 | 谓词对每个候选独立判定，天然支持 |
| 重映射能力 | 删除（谓词只能排除）——唯一使用方只做排除 |

## 测试设计

### common/crontask 重写/新增
- `TestSchedulerPreviewNextRunsSkipsInvalidCandidates`：`FREQ=HOURLY;COUNT=8`，谓词排除 09-12 时 → preview(count=2) = [13, 14]；谓词调用次数 = 候选数（6）。
- 删除 `TestSchedulerPreviewNextRunsRejectsNonAdvancingFilterResult`。
- `TestSchedulerPreviewNextRunsExhaustedInsideInvalidWindow`：谓词永久 true → 返回空结果 + nil error。
- `computeNextRun` 三个现有用例改为 `NewScheduler(nil, nil).computeNextRun(...)`。
- `TestNextRunsFiltered`（query_test.go，若已有 NextRuns 用例则扩展）：nil 谓词等价 NextRuns；谓词跳过中间候选后游标严格递增。

### ispagent 重写/新增
- `TestInvalidTimeSkip` → 改用 `NextRunsFiltered(rruleStr, base, 1, pred)` 断言跳到下周。
- `TestNoInvalidTimeNoSkip` → 同上断言原候选返回。
- `TestInvalidTimeFilterReturnsZeroWhenRuleExhausted` → `NewInvalidTimePredicate()(task, runAt)` 为 true；`NextRunsFiltered` 返回空。
- `TestNewInvalidTimePredicateConcurrent`：谓词无状态，并发调用结果一致（-race 兜底）。

## 验证命令

```bash
go build ./...
go test ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/internal/cronjob
go test -race ./common/crontask ./app/ispagent/internal/crontask
```

## 规格同步

更新 `.trellis/spec/backend/crontask-guidelines.md`：
- 「调度时间预览」节：`InvalidTimeFilter` 描述改为谓词语义（`InvalidTimePredicate`，公共层单趟推进，谓词逐候选介入，只做排除）。
- 反模式条目「`InvalidTimeFilter` 返回后再次调用 `Set.After` 推进」改为「谓词内调用 `NextAfter`/`Set.After` 或返回时间（谓词只做排除）」「业务侧自行循环调用 `NextAfter` 推进（解析与平推逻辑必须留在公共层）」。
- 验证节补 `go test -race ./app/ispagent/internal/crontask`。
