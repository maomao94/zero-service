# crontask 调度契约

> 适用于 `common/crontask/` 通用调度器，以及 `app/ispagent/internal/crontask/` 的规则转换和 GORM `TaskStore` 实现。可空时间的通用数据库映射另见 [database-guidelines.md](./database-guidelines.md#scenario-可空时间字段清空为-sql-null)。

## Scenario: 任务调度、抢占与完成

### 1. Scope / Trigger

修改以下任一内容时必须遵循本契约：

- `TaskConfig.NextRun`、`LastRun`、`RRuleStr` 或任务终态语义。
- `TaskStore` 的扫描、抢占、配置更新或执行完成接口。
- `Scheduler.executeTask`、`RunNow`、最大延迟和不可用时间过滤。
- `cron_task_config.next_run`、`last_run`、`status` 或用于并发控制的字段。
- MemoryStore 与 DBStore 之间的可观察行为。

这是一条跨层契约：规则生成、调度器、Store、GORM 模型和数据库查询必须对“无下次调度”和“当前执行持有的 claim”保持同一解释。

### 2. Signatures

当前公共结构和接口：

```go
type TaskConfig struct {
    ID       string
    TaskCode string
    RRuleStr string
    Status   TaskStatus
    NextRun  time.Time // 零值表示无下次调度
    LastRun  time.Time // 零值表示从未成功执行
    Payload  json.RawMessage
    Extra    json.RawMessage
    Version  int64
}

type TaskStore interface {
    LockAndFetch(ctx context.Context, now time.Time, lockDur time.Duration) (*TaskConfig, error)
    UpdateNextRun(ctx context.Context, id string, nextRun, lastRun time.Time) error
    GetByCode(ctx context.Context, taskCode string) (*TaskConfig, error)
    Insert(ctx context.Context, cfg *TaskConfig) error
    Update(ctx context.Context, cfg *TaskConfig) error
    UpdateStatus(ctx context.Context, id string, status TaskStatus) error
    Delete(ctx context.Context, id string) error
    ListEnabled(ctx context.Context) ([]*TaskConfig, error)
}
```

只按 `id` 完成任务，无法阻止旧执行覆盖并发下发的新配置。如果允许“任务执行”和“配置更新”并发，接口必须演进为携带不可变 claim token 的等价形式。推荐使用锁定截止时间作为 token，避免仅为了 claim 新增数据库版本列：

```go
type TaskClaim struct {
    Task        *TaskConfig
    LockedUntil time.Time
}

LockAndFetch(ctx context.Context, now time.Time, lockDur time.Duration) (*TaskClaim, error)
Complete(ctx context.Context, id string, expectedLockedUntil, nextRun, lastRun time.Time) error
UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error
```

数据库模型：

```go
NextRun sql.NullTime `gorm:"column:next_run;type:timestamp"`
LastRun sql.NullTime `gorm:"column:last_run;type:timestamp"`
```

### 3. Contracts

#### 时间与任务终态

- `NextRun.IsZero()`、`sql.NullTime.Valid == false` 和 SQL `NULL` 都表示“无下次调度”。禁止用当前时间加 100 年或最大时间代替。
- `LastRun.IsZero()` 表示从未成功执行。规则耗尽、最大延迟跳过和规则解析失败都不能冒充成功执行。
- 启用任务可以处于 `NextRun` 零值终态；此时保留 `StatusEnabled`，但扫描永远不能选中它。
- 一次性任务可使用空 `RRuleStr` 或带 `COUNT=1` 的有效规则。空规则必须在调用 rrule parser 前直接转换为零 `NextRun`，不能把 parser error 当成完成结果。
- 有效规则返回零时间表示 `COUNT` 耗尽或超过 `UNTIL`，应持久化 SQL `NULL`。
- 非空但无法解析的规则是配置错误，必须在 Insert/Update 边界拒绝；不能先执行 handler，再因计算下次时间失败而等待锁过期重跑。
- `InvalidTimeFilter` 收到零时间时必须原样返回零时间，不能从零值重新推导出未来计划。

#### 扫描与 claim

- 候选查询条件必须同时包含 `status = enabled`、`next_run IS NOT NULL` 和 `next_run <= now`。
- SELECT 之后的 claim UPDATE 必须重新校验 `id`、`status = enabled` 和 SELECT 快照中的原始 `next_run`；只写 `next_run <= now` 不能阻止禁用或配置更新窗口。
- claim UPDATE 成功后把 `next_run` 写为 `lockedUntil`，并向执行方返回原计划时间和 `lockedUntil` token。
- `RowsAffected == 0` 表示 claim 已丢失，返回 `ErrNotFound`；调用方不得执行 SELECT 得到的旧快照。
- 两个实例争抢同一行时必须只有一个 claim 成功。

#### 完成与配置并发

- 完成更新必须使用 `id + expectedLockedUntil` 做 CAS。数据库当前 `next_run` 不等于 claim token 时，说明配置已更新、任务已被重新 claim 或状态已改变，旧执行不得覆盖它。
- 配置 Update 必须保留运行态 `LastRun`；不能因为新下发的 `TaskConfig.LastRun` 为零而清空执行历史。
- 只有确实嵌入并持久化 `gormx.VersionMixin`，且 UPDATE 带版本谓词和版本递增时，才能称为 optimistic lock。时间扩展 claim 应称为 lease/CAS，不能在注释中声称由 optimisticlock 插件处理。
- DBStore 和 MemoryStore 必须遵循相同可观察语义；MemoryStore 返回的 `TaskConfig` 是独立快照，`Payload` 和 `Extra` 也必须复制底层字节。

#### `RunNow`

- `RunNow(ctx, taskCode)` 是独立的立即执行，不推进、不清空持久化 `NextRun`，也不调用周期完成逻辑。
- handler 收到的执行快照应使用当前秒作为本次执行时间，不得把未来的计划 `NextRun` 当成立即执行时间。
- `RunNow` handler 成功后通过独立的 `UpdateLastRun` 记录实际执行时间；不得借 `UpdateNextRun` 顺带写回读取到的旧计划。
- `RunNow` 不修改 `NextRun` 或 `Status`；handler 返回 error 或 panic 时也不修改 `LastRun`。
- 异步执行必须使用 `threading.GoSafe` 或等价 recover 边界；handler panic 只能记录日志，不能导致进程退出。
- `GetByCode` 的错误同步返回；异步 handler 错误记录日志，调度状态保持不变。

### 4. Validation & Error Matrix

| 条件 | 正确行为 |
| --- | --- |
| `NextRun` 为零 / SQL `NULL` | 不进入扫描 |
| `RRuleStr == ""` 的一次性任务成功执行 | `NextRun` 置零，后续不再执行 |
| 非空 RRule 无法解析 | Insert/Update 返回错误，不允许进入可执行状态 |
| RRule `COUNT` 耗尽或超过 `UNTIL` | `NextRun` 置零，`Status` 保持不变 |
| 不可用时间过滤耗尽规则 | 保持零 `NextRun` |
| handler 返回 error | 不写 `LastRun`；保留 lease，锁过期后重试 |
| 超过 `MaxDelay` 被跳过 | 推进到下一计划，但不写 `LastRun` |
| SELECT 后任务被禁用 | claim UPDATE 影响 0 行，不执行 |
| 两个实例同时 claim | 一个成功，另一个返回 `ErrNotFound` |
| 执行期间配置更新了 `next_run` | 旧 Complete CAS 失败，不覆盖新计划 |
| `RunNow` 查询不到任务 | 同步返回 `ErrNotFound` |
| `RunNow` handler 成功 | 只更新 `LastRun`，`NextRun` 和 `Status` 不变 |
| `RunNow` handler error/panic | 记录错误，进程继续运行，三个调度状态字段都不变 |

### 5. Good/Base/Bad Cases

- Good: 一次性任务执行成功后写 SQL `NULL`，保持 enabled；多次扫描仍只执行一次。
- Good: claim 携带 `lockedUntil`，完成时用同一 token 做 CAS；并发配置下发胜出后旧执行不能覆盖新计划。
- Good: `RunNow` 使用当前时间快照和 panic recover，成功后只记录 LastRun，持久化计划保持一致。
- Base: 单实例 MemoryStore 也实现相同终态、LastRun 和深拷贝语义，便于测试替换 DBStore。
- Bad: 空 RRule 直接传给 parser，handler 已成功但 completion 报错，lease 到期后重复执行。
- Bad: `UPDATE ... WHERE id = ?` 完成任务，导致旧规则计算出的 `next_run` 覆盖新配置。
- Bad: SELECT 校验 enabled，但 claim UPDATE 不校验 status，导致刚禁用的任务仍执行一次。
- Bad: `RunNow` 直接复用 `executeTask` 或使用原生 `go`。

### 6. Tests Required

#### `common/crontask`

- 空 RRule 一次性任务：启动 Scheduler，跨越至少两个 lock window，断言 handler 只调用一次、`NextRun.IsZero()`、状态保持 enabled。
- 有限 RRule：断言 `COUNT=1` 执行后为零；非法非空 RRule 在写入边界返回错误且 handler 未调用。
- MaxDelay：断言跳过时 NextRun 推进，但 LastRun 不变。
- `RunNow`：分别覆盖零、未来和周期 NextRun；断言 handler 收到当前执行时间，成功后只更新 LastRun，NextRun/Status 不变。
- `RunNow` error/panic：断言 LastRun 不更新、进程不退出，后续正常任务仍可执行。
- MemoryStore clone：修改 Insert 入参、Get/List 返回值中的 `Payload`/`Extra` 字节，断言 Store 内部值不变。
- MemoryStore Update：断言配置更新不会清空已有 LastRun。

#### `app/ispagent/internal/crontask`

- SQL NULL：新增、局部更新和全量更新后均断言 `next_run.Valid == false`，且扫描不选中。
- claim 状态竞争：SELECT 后禁用任务，再执行 claim UPDATE，断言 0 行且 handler 不执行。
- 双实例 claim：并发争抢同一到期行，断言仅一个成功。
- 配置并发更新：claim 后更新 RRule/NextRun，再完成旧 claim，断言 CAS 失败且新 NextRun 未被覆盖。
- claim 精确性：两个到期任务中只更新被选中行；相同优先级仍可随机选择。
- 验证命令至少包括：

```bash
go test -count=1 ./common/crontask ./app/ispagent/internal/crontask
go test -race -count=1 ./common/crontask ./app/ispagent/internal/crontask
go test -count=1 ./app/ispagent/...
go vet ./common/crontask ./app/ispagent/internal/crontask
go build ./...
```

### 7. Wrong vs Correct

#### Wrong: 空规则在 handler 成功后才暴露解析错误

```go
if err := handler(ctx, task); err != nil {
    return
}
rule, err := rrule.StrToRRule(task.RRuleStr) // 空串报错，lease 到期后重复执行
```

#### Correct: 配置边界校验，空规则显式表示一次性终态

```go
func computeNextRun(task *TaskConfig) (time.Time, error) {
    if task.RRuleStr == "" {
        return time.Time{}, nil
    }
    rule, err := rrule.StrToRRule(task.RRuleStr)
    if err != nil {
        return time.Time{}, err
    }
    base := time.Now()
    if task.NextRun.After(base) {
        base = task.NextRun
    }
    return rule.After(base, false), nil
}
```

#### Wrong: claim 和 completion 只依赖宽松条件

```go
db.Where("id = ?", id).
    Where("next_run <= ?", now).
    Update("next_run", lockedUntil)

db.Where("id = ?", id).
    Updates(map[string]any{"next_run": nextRun, "last_run": lastRun})
```

#### Correct: claim 和 completion 使用同一个不可变 token

```go
claim := db.Where("id = ?", id).
    Where("status = ?", StatusEnabled).
    Where("next_run = ?", observedNextRun).
    Update("next_run", lockedUntil)

complete := db.Where("id = ?", id).
    Where("status = ?", StatusEnabled).
    Where("next_run = ?", lockedUntil).
    Updates(map[string]any{"next_run": toNullTime(nextRun), "last_run": lastRun})
```

#### Wrong vs Correct: `RunNow`

```go
// Wrong: 推进周期且 panic 无保护。
go s.executeTask(task)

// Correct: 使用独立执行快照，不进入周期完成逻辑。
snapshot := cloneTaskConfig(task)
snapshot.NextRun = time.Now().Truncate(time.Second)
threading.GoSafe(func() {
    if err := s.handler(context.Background(), snapshot); err != nil {
        logx.Errorf("[crontask] run now failed: %v", err)
        return
    }
    if err := s.store.UpdateLastRun(context.Background(), task.ID, snapshot.NextRun); err != nil {
        logx.Errorf("[crontask] update manual last run failed: %v", err)
    }
})
```

## Common Mistakes

- 用 Carbon `IsValid()` 判断业务时间是否存在。Carbon 零值也可能 valid；存在性统一使用标准库 `time.Time.IsZero()`。
- 把“没有下次调度”和“规则计算失败”都表示成零时间。前者是正常终态，后者必须返回错误。
- 只为扫描竞争做 CAS，却忽略执行完成与配置下发之间的 lost update。
- DBStore 保留 LastRun，但 MemoryStore 全量覆盖 LastRun，导致测试替身和生产行为不一致。
- 结构体浅拷贝后认为 `json.RawMessage` 已隔离；它仍共享底层 `[]byte`。
