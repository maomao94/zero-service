# 实施计划：crontask 无效区间过滤重构

## 执行顺序

### 1. common/crontask/query.go
- [ ] 新增 `NextRunsFiltered(value, after, count, invalid)`：单解析 + 单迭代器，谓词跳过候选。
- [ ] `NextRuns` 改为委托 `NextRunsFiltered(..., nil)`。
- [ ] `NextAfter` 改为委托 `NextRunsFiltered(..., 1, nil)` 取首个。

### 2. common/crontask/options.go
- [ ] `InvalidTimeFilter` → `InvalidTimePredicate func(task *TaskConfig, t time.Time) bool`，更新注释。
- [ ] `WithInvalidTimeFilter` → `WithInvalidTimePredicate`。
- [ ] `SchedulerOptions.InvalidTimeFilter` 字段改名 `InvalidTimePredicate`。

### 3. common/crontask/crontask.go
- [ ] `Scheduler.invalidTimeFilter` 字段改名 `invalidTimePredicate`（:40, :61）。
- [ ] 新增 `(s *Scheduler) nextRuns(task, after, count)`。
- [ ] `computeNextRun` 改方法 `(s *Scheduler) computeNextRun`，走 `s.nextRuns(cfg, base, 1)`。
- [ ] 完成路径删除 :193-195 后置过滤调用。
- [ ] `PreviewNextRuns` 委托 `s.nextRuns`，删除内联迭代与过滤块、非前进守卫。

### 4. app/ispagent/internal/crontask/task_rule.go
- [ ] 删除 `skipInvalidTime`。
- [ ] 新增 `invalidTimePredicate()`（纯谓词，窗口缺失返回 nil）。
- [ ] `CalcInitNextRun` 改用 `crontask.NextRunsFiltered(rruleStr, now, 1, pred)`。
- [ ] `NewInvalidTimeFilter` → `NewInvalidTimePredicate`：无状态谓词，每次调用反序列化 Extra，不引入缓存。

### 5. app/ispagent/internal/svc/servicecontext.go
- [ ] `WithInvalidTimeFilter` → `WithInvalidTimePredicate`，`NewInvalidTimeFilter` → `NewInvalidTimePredicate`。

### 6. 测试重写
- [ ] common/crontask/crontask_test.go：preview 谓词用例重写（候选数断言）、删非前进守卫用例、新增窗口内耗尽用例；computeNextRun 用例改方法调用。
- [ ] common/crontask/query_test.go（若存在）：补 NextRunsFiltered 谓词跳过用例。
- [ ] app/ispagent task_rule_test.go：三个无效区间用例改用新 API；补谓词缓存行为用例。

### 8. rrulex 抽取（工具方法整理）
- [ ] 新建 `common/rrulex/`：`rrulex.go`（ParseSet/Validate）、`query.go`（ShiftSetForQuery/NextRuns + shiftDtStartByPeriod；不导出 NextAfter/QuerySet）、`describe.go`（Describe + ErrUnsupportedDescription，从 common/crontask 原样迁移改名）。
- [ ] 删除 `common/crontask/query.go`、`describe.go`、`describe_test.go`；`errors.go` 移除 `ErrUnsupportedDescription`；`crontask.go` 移除 `parseRRuleSet`/`ValidateRRule`，`computeNextRun`/`nextRuns`/`PreviewNextRuns` 委托 `rrulex.NextRuns`。
- [ ] 单次 after 场景改原生官方风格：ispagent `db_store.go` Enable、trigger `cronjob/db_store.go` Enable、`common/crontask/memory_store.go` Enable —— `rrulex.ParseSet` + `set.After(time.Now(), false)`（保留 RRuleStr 非空守卫）。
- [ ] ispagent `task_rule.go`（NextRunsFiltered→rrulex.NextRuns）、`logic/helper.go`（Describe）；trigger `cronjob/schedule.go`（ShiftSetForQuery）、`calcplantaskdatelogic.go`、`getplanlogic.go`、`previewcronjobschedulelogic.go`、`listplanslogic.go`（Describe）。
- [ ] 测试迁移：`common/crontask/crontask_test.go` 中 rrule 级测试迁到 `common/rrulex/rrulex_test.go`（NextAfter/ValidateRRule 用例改写为 ParseSet/Validate + 官方 set.After 与 NextRuns 的平移差分断言；TestNextRunsFiltered 改 TestNextRuns）；`describe_test.go` 迁到 `common/rrulex/` 改名 Describe；crontask_test.go 保留调度器行为测试。

### 9. 规格同步
- [ ] `.trellis/spec/backend/crontask-guidelines.md`：预览节谓词语义 + rrulex 归属 + 反模式更新 + 验证命令补 race。

## 验证（每步后按需，最终全跑）

```bash
go build ./...
go test ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/internal/cronjob
go test -race ./common/crontask ./app/ispagent/internal/crontask
go vet ./common/crontask ./app/ispagent/internal/crontask
```

## 回滚点

- 第 1-3 步为公共层核心，完成后先 `go test ./common/crontask` 绿再动业务侧。
- 第 4-5 步为 ispagent 同步，完成后 `go test ./app/ispagent/internal/crontask`。
- 任何一步红 → 定位修复或回退该步 diff。
