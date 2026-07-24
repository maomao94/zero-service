# crontask 调度契约

> 适用于 `common/crontask/` 通用调度器、`app/ispagent/internal/crontask/` 和 `app/trigger/internal/cronjob/` 的 GORM `TaskStore` 实现。可空时间的通用数据库映射另见 [database-guidelines.md](./database-guidelines.md#scenario-可空时间字段清空为-sql-null)。

## Scenario: 任务调度、抢占与完成

### 1. Scope / Trigger

修改以下任一内容时必须遵循本契约：

- `TaskConfig.NextRun`、`LastRun`、`RRuleStr` 或任务终态语义。
- `TaskStore` 的扫描、抢占、配置更新或执行完成接口。
- `Scheduler.executeTask`、`RunNow`、最大延迟和不可用时间过滤。
- `cron_task_config`、`cron_job` 的 `next_run`、`last_run`、`scheduled_time`、`status` 或用于并发控制的字段。
- MemoryStore 与 DBStore 之间的可观察行为。

这是一条跨层契约：规则生成、调度器、Store、GORM 模型和数据库查询必须对“无下次调度”和“当前执行持有的 claim”保持同一解释。

### 2. Signatures

当前公共结构和接口：

```go
type TaskConfig struct {
    ID       string
    TaskCode string
    RRuleStr string
    LockTimeout time.Duration // 零值使用 Scheduler 默认锁超时
    Status   TaskStatus
    NextRun  time.Time // 零值表示无下次调度
    LastRun  time.Time // 零值表示从未成功执行
    Payload  json.RawMessage
    Extra    json.RawMessage
}

type TaskClaim struct {
    Task        *TaskConfig
    LockedUntil time.Time
}

type ListCondition struct {
    Statuses []TaskStatus // 空切片表示不过滤状态
}

const MinLockTimeout = 30 * time.Second

var ErrUpdate = errors.New("[crontask] task update affected no rows")

type TaskStore interface {
    LockAndFetch(ctx context.Context, now time.Time, lockDur time.Duration) (*TaskClaim, error)
    Complete(ctx context.Context, id string, expectedLockedUntil, nextRun, lastRun time.Time) error
    UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error
    GetByCode(ctx context.Context, taskCode string) (*TaskConfig, error)
    Insert(ctx context.Context, cfg *TaskConfig) error
    Update(ctx context.Context, cfg *TaskConfig) error
    Enable(ctx context.Context, id string) error
    Disable(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, condition ListCondition) ([]*TaskConfig, error)
}

// app/ispagent/internal/crontask 的 ISP 101-1 配置重建入口；existing 为 nil 表示新增任务。
func NewTaskConfig(existing *crontask.TaskConfig, fields *IspTaskFields) (*crontask.TaskConfig, error)
```

完成接口必须携带不可变 claim token。锁定截止时间直接复用持久化的 `next_run`，不再为 claim 增加或维护无效的 `Version` 字段。

数据库模型：

```go
NextRun sql.NullTime `gorm:"column:next_run;type:timestamp"`
LastRun sql.NullTime `gorm:"column:last_run;type:timestamp"`
LockTimeout int64 `gorm:"column:lock_timeout"` // 毫秒，0 使用 Scheduler 默认值；最终执行值仍受 MinLockTimeout 约束

// 仅 Trigger CronJob 使用：重试期间固化首次业务计划时间。
ScheduledTime sql.NullTime `gorm:"column:scheduled_time;type:timestamp"`
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
- `TaskConfig.LockTimeout > 0` 时，本次 claim 优先使用任务级锁超时；否则使用 Scheduler 的 `LockExpire` 默认值。`ResolveLockTimeout` 必须把最终值限制为不小于 `MinLockTimeout`（当前为 30 秒），避免 `lockedUntil` 截到整秒后租约立即过期。通用层使用 `time.Duration`，支持该字段的数据库和对外协议使用毫秒。ISP `101-1` 不包含该字段，配置更新必须保留已有值。
- SELECT 之后的 claim UPDATE 必须重新校验 `id`、`status = enabled` 和 SELECT 快照中的原始 `next_run`；只写 `next_run <= now` 不能阻止禁用或配置更新窗口。
- claim UPDATE 成功后把 `next_run` 写为 `lockedUntil`，并向执行方返回原计划时间和 `lockedUntil` token。
- `lockedUntil` 必须在写入前截到整秒，匹配 RRULE 和 `timestamp` 列的共同精度；禁止把未归一化的纳秒值作为后续 CAS token。
- `RowsAffected == 0` 表示 claim 已丢失，返回 `ErrNotFound`；调用方不得执行 SELECT 得到的旧快照。
- 两个实例争抢同一行时必须只有一个 claim 成功。
- 只复用 `next_run` 作为 lease 时，重试读到的是上一次 `lockedUntil`，不能保证业务 `scheduledTime` 稳定。Trigger `cron_job` 必须在首次 claim 时写入 `scheduled_time`，重试沿用该值，成功 Complete 或重新启用时清空。

#### 完成与配置并发

- 完成更新必须使用 `id + expectedLockedUntil` 做 CAS。数据库当前 `next_run` 不等于 claim token 时，说明配置已更新、任务已被重新 claim 或重新启用，旧执行不得覆盖它。
- Complete 不增加 `status = enabled` 条件。禁用只阻止未来扫描，不撤销已执行中的 claim；否则 handler 已成功但并发禁用会导致 Complete 失败并在后续重新启用时重复执行。
- 配置 Update 必须保留运行态 `LastRun`；不能因为新下发的 `TaskConfig.LastRun` 为零而清空执行历史。
- 只有确实嵌入并持久化 `gormx.VersionMixin`，且 UPDATE 带版本谓词和版本递增时，才能称为 optimistic lock。时间扩展 claim 应称为 lease/CAS，不能在注释中声称由 optimisticlock 插件处理。
- DBStore 和 MemoryStore 返回的 `TaskConfig` 都是独立结构体对象。`Payload` 和 `Extra` 按只读值处理，不复制底层字节；Handler、调用方和 Store 都不得原地修改其内容。

#### ISP 配置重建与字段所有权

- `HandleTaskDispatch` 查询到已有任务后，将完整的 `existing` 传给 `NewTaskConfig(existing, fields)`；新增任务传 `nil`。
- `NewTaskConfig` 负责从 `existing` 继承 `ID` 和 ISP `101-1` 不携带的 `LockTimeout`，同时根据报文字段重新计算 `RRuleStr`、`NextRun` 和其他 ISP 配置。
- Handler 不得先用 `existing.ID` 构造，再在返回后补写 `cfg.LockTimeout`。这种两阶段构造会把字段所有权泄漏给调用方，后续增加协议外持久化字段时容易再次丢失。
- 新增、更新和删除分支统一使用构造结果的 `cfg.ID`：空值执行 Insert，非空执行 Update，删除报文只在非空时调用 Delete。

#### 生命周期与列表

- `Enable(ctx, id)` 由 Store 根据已持久化的 `RRuleStr` 从当前时间计算未来 `NextRun`，调用方不得传入自行计算的时间。
- 已启用任务重复调用 `Enable` 直接成功，不能重算或覆盖当前 `next_run`，避免撤销在途 claim。禁用任务启用时同时写入 enabled 和新 `next_run`；Trigger 还必须清空 `scheduled_time`。
- `Disable(ctx, id)` 必须先按唯一 ID 查询任务：不存在或查询失败统一返回 `ErrUpdate`，已 disabled 直接成功；其余状态执行更新，`result.Error != nil` 或 `RowsAffected == 0` 都统一返回 `ErrUpdate`。禁止更新后再用 `Count` 判断是否存在。
- ISP 和 Trigger 的 GORM `Delete(ctx, id)` 是幂等软删除：存在、已删除或不存在都返回 nil，只有实际数据库错误才返回 error。`Scheduler` 仍须兼容其他 `TaskStore` 在删除缺失任务时返回 `ErrNotFound`。
- 禁用只阻止未来扫描；在途 handler 成功后仍可通过原 lease token Complete。
- `List(ctx, ListCondition{})` 返回全部任务；`Statuses` 非空时使用 `status IN (...)` 过滤，支持同时查询多个状态。

#### `RunNow`

- `RunNow(ctx, taskCode)` 是独立的立即执行，不推进、不清空持久化 `NextRun`，也不调用周期完成逻辑。
- handler 收到的执行快照应使用当前秒作为本次执行时间，不得把未来的计划 `NextRun` 当成立即执行时间。
- `RunNow` handler 成功后通过独立的 `UpdateLastRun` 记录实际执行时间；不得借 `UpdateNextRun` 顺带写回读取到的旧计划。
- `RunNow` 不修改 `NextRun` 或 `Status`；handler 返回 error 或 panic 时也不修改 `LastRun`。
- 异步 handler 使用 `context.WithoutCancel(ctx)`：保留调用方注入的本次执行元数据，但不继承请求取消信号或截止时间。
- 异步执行必须使用 `threading.GoSafe` 或等价 recover 边界；handler panic 只能记录日志，不能导致进程退出。
- `GetByCode` 的错误同步返回；异步 handler 错误记录日志，调度状态保持不变。
- `RunNow` 直接使用 Store 返回的独立任务对象，不再额外 clone；成功时间在 handler 返回 nil 后生成。
- `RunNow` 收到 `ErrDeleteTask` 时删除当前任务，但仍不调用周期 Complete。
- `Scheduler.Stop` 必须停止新扫描并等待已经 claim 的周期 handler 退出；不能在在途回调仍运行时宣告服务已停止。

### 4. Validation & Error Matrix

| 条件 | 正确行为 |
| --- | --- |
| `NextRun` 为零 / SQL `NULL` | 不进入扫描 |
| `RRuleStr == ""` 的一次性任务成功执行 | `NextRun` 置零，后续不再执行 |
| 非空 RRule 无法解析 | Insert/Update 返回错误，不允许进入可执行状态 |
| RRule `COUNT` 耗尽或超过 `UNTIL` | `NextRun` 置零，`Status` 保持不变 |
| 不可用时间过滤耗尽规则 | 保持零 `NextRun` |
| handler 返回 error | 不写 `LastRun`；保留 lease，锁过期后重试 |
| handler 返回直接或包装的 `ErrDeleteTask` | 删除当前任务且不推进；删除失败保留 lease 等待重试 |
| 超过 `MaxDelay` 被跳过 | 推进到下一计划，但不写 `LastRun` |
| SELECT 后任务被禁用 | claim UPDATE 影响 0 行，不执行 |
| 两个实例同时 claim | 一个成功，另一个返回 `ErrNotFound` |
| 执行期间配置更新了 `next_run` | 旧 Complete CAS 失败，不覆盖新计划 |
| 执行期间任务被禁用 | 当前 claim 成功 Complete，状态保持 disabled，后续不扫描 |
| disabled 任务首次 Enable | Store 从当前时间重算未来 `NextRun`，不追赶禁用历史 |
| enabled 任务重复 Enable | 幂等成功，`NextRun` 和在途 claim 不变 |
| task/default lock timeout 小于 30 秒 | `ResolveLockTimeout` 返回 `MinLockTimeout` |
| 已 disabled 任务重复 Disable | 幂等成功，不执行 UPDATE |
| Disable 查询失败、任务不存在、UPDATE 报错或影响 0 行 | 统一返回 `ErrUpdate` |
| ISP/Trigger GORM Delete 的任务不存在或已删除 | 幂等成功 |
| ISP/Trigger GORM Delete 发生数据库错误 | 返回实际数据库错误 |
| ISP 新增任务，`existing == nil` | `cfg.ID` 为空、`cfg.LockTimeout` 为零，后续 Insert 使用 Scheduler 默认锁超时 |
| ISP 更新任务，`existing != nil` | 继承已有 `ID`、`LockTimeout`，报文字段生成新的规则与执行时间 |
| `RunNow` 查询不到任务 | 同步返回 `ErrNotFound` |
| `RunNow` handler 成功 | 只更新 `LastRun`，`NextRun` 和 `Status` 不变 |
| `RunNow` handler error/panic | 记录错误，进程继续运行，三个调度状态字段都不变 |

### 5. Good/Base/Bad Cases

- Good: 一次性任务执行成功后写 SQL `NULL`，保持 enabled；多次扫描仍只执行一次。
- Good: claim 携带 `lockedUntil`，完成时用同一 token 做 CAS；并发配置下发胜出后旧执行不能覆盖新计划。
- Good: Trigger 首次 claim 固化 `scheduled_time`，普通错误重试时 Eventstream 始终收到同一个计划时间。
- Good: `RunNow` 使用当前时间快照和 panic recover，成功后只记录 LastRun，持久化计划保持一致。
- Good: `NewTaskConfig(existing, fields)` 在一次构造中同时生成报文配置并继承协议外持久化字段。
- Good: 任务级值和 Scheduler 默认值都经过 `ResolveLockTimeout`，最终 lease 至少为 30 秒。
- Good: `Disable` 先查询并识别已禁用状态，再更新；查询、更新错误或 0 影响行统一映射为 `ErrUpdate`。
- Good: ISP 和 Trigger 的 GORM `Delete` 对重复删除和删除缺失任务保持幂等。
- Base: 单实例 MemoryStore 返回独立结构体快照，并约定 `Payload`、`Extra` 为只读数据。
- Base: `NewTaskConfig(nil, fields)` 构造新增任务，ID 和任务级锁超时保持零值。
- Bad: 空 RRule 直接传给 parser，handler 已成功但 completion 报错，lease 到期后重复执行。
- Bad: `UPDATE ... WHERE id = ?` 完成任务，导致旧规则计算出的 `next_run` 覆盖新配置。
- Bad: SELECT 校验 enabled，但 claim UPDATE 不校验 status，导致刚禁用的任务仍执行一次。
- Bad: Complete 增加 enabled 条件，导致执行中禁用的成功任务无法确认并在以后产生重复执行。
- Bad: 重复 Enable 仍然重算 `NextRun`，导致正在执行的 lease token 被无意覆盖。
- Bad: Handler 在 `NewTaskConfig` 返回后单独执行 `cfg.LockTimeout = existing.LockTimeout`。
- Bad: `Disable` 直接盲写并把 0 影响行当成功，或 UPDATE 后再执行 `Count` 判断唯一 ID 是否存在。
- Bad: `RunNow` 直接复用 `executeTask` 或使用原生 `go`。

### 6. Tests Required

#### `common/crontask`

- 空 RRule 一次性任务：启动 Scheduler，跨越至少两个 lock window，断言 handler 只调用一次、`NextRun.IsZero()`、状态保持 enabled。
- 有限 RRule：断言 `COUNT=1` 执行后为零；非法非空 RRule 在写入边界返回错误且 handler 未调用。
- MaxDelay：断言跳过时 NextRun 推进，但 LastRun 不变。
- `RunNow`：分别覆盖零、未来和周期 NextRun；断言 handler 收到当前执行时间，成功后只更新 LastRun，NextRun/Status 不变。
- `RunNow` error/panic：断言 LastRun 不更新、进程不退出，后续正常任务仍可执行。
- MemoryStore 快照：修改 Insert 入参、Get/List 返回对象的结构体字段，断言 Store 内部结构体字段不变；RawMessage 按只读契约使用。
- MemoryStore Update：断言配置更新不会清空已有 LastRun。
- 锁超时下限：覆盖任务级 1 秒、Scheduler 默认 1 秒都解析为 `MinLockTimeout`；MemoryStore 在带纳秒的当前时间下不能立即重复 claim。
- `ErrDeleteTask`：覆盖直接错误、包装错误、删除失败后 lease 过期重试。
- 生命周期：覆盖 Enable/Disable 幂等、重复 Enable 不覆盖当前 `NextRun`、缺失任务 Disable 返回 `ErrUpdate`、禁用期间 Complete 成功。

#### `app/ispagent/internal/crontask`

- SQL NULL：新增、局部更新和全量更新后均断言 `next_run.Valid == false`，且扫描不选中。
- claim 状态竞争：SELECT 后禁用任务，再执行 claim UPDATE，断言 0 行且 handler 不执行。
- 双实例 claim：并发争抢同一到期行，断言仅一个成功。
- 配置并发更新：claim 后更新 RRule/NextRun，再完成旧 claim，断言 CAS 失败且新 NextRun 未被覆盖。
- claim 精确性：两个到期任务中只更新被选中行；相同优先级仍可随机选择。
- 任务级锁超时：覆盖 0 值使用 Scheduler 默认值、正数覆盖默认值，以及 ISP/Trigger 毫秒列的持久化往返。
- ISP 配置重建：`existing == nil` 时断言 ID/LockTimeout 为零；`existing != nil` 时断言继承 ID/LockTimeout，并且 Handler 更新回归测试验证持久化值不被覆盖。
- ISP 与 Trigger GORM 生命周期：重复 Disable 成功、缺失任务 Disable 返回 `ErrUpdate`；重复 Delete 和删除缺失任务均成功。
- Trigger CronJob：覆盖 `scheduled_time` 首次固化、失败重试保持稳定、Complete/Enable 后清空。
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

#### Wrong: Handler 在构造完成后补写协议外字段

```go
cfg, err := ctask.NewTaskConfig(existing.ID, fields)
if err != nil {
    return err
}
cfg.LockTimeout = existing.LockTimeout
```

#### Correct: 构造函数拥有完整的配置重建规则

```go
cfg, err := ctask.NewTaskConfig(existing, fields)
if err != nil {
    return err
}
if cfg.ID == "" {
    return store.Insert(ctx, cfg)
}
return store.Update(ctx, cfg)
```

#### Wrong: Disable 盲写后再查询唯一 ID

```go
result := db.Model(&gormmodel.GormTaskConfig{}).
    Where("id = ?", id).
    Update("status", int(crontask.StatusDisabled))
if result.Error != nil {
    return result.Error
}
var count int64
db.Model(&gormmodel.GormTaskConfig{}).Where("id = ?", id).Count(&count)
```

#### Correct: 查询状态后更新并统一返回领域错误

```go
var record gormmodel.GormTaskConfig
if err := db.Where("id = ?", id).First(&record).Error; err != nil {
    return crontask.ErrUpdate
}
if crontask.TaskStatus(record.Status) == crontask.StatusDisabled {
    return nil
}
result := db.Model(&gormmodel.GormTaskConfig{}).
    Where("id = ?", id).
    Update("status", int(crontask.StatusDisabled))
if result.Error != nil || result.RowsAffected == 0 {
    return crontask.ErrUpdate
}
return nil
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
    Where("next_run = ?", lockedUntil).
    Updates(map[string]any{"next_run": toNullTime(nextRun), "last_run": lastRun})
```

#### Wrong vs Correct: `RunNow`

```go
// Wrong: 推进周期且 panic 无保护。
go s.executeTask(task)

// Correct: Store 已返回独立任务对象，直接设置本次执行时间，不进入周期完成逻辑。
task.NextRun = time.Now().Truncate(time.Second)
runCtx := context.WithoutCancel(ctx)
threading.GoSafe(func() {
    if err := s.handler(runCtx, task); err != nil {
        logx.Errorf("[crontask] run now failed: %v", err)
        return
    }
    if err := s.store.UpdateLastRun(runCtx, task.ID, time.Now()); err != nil {
        logx.Errorf("[crontask] update manual last run failed: %v", err)
    }
})
```

## Common Mistakes

- 用 Carbon `IsValid()` 判断业务时间是否存在。Carbon 零值也可能 valid；存在性统一使用标准库 `time.Time.IsZero()`。
- 把“没有下次调度”和“规则计算失败”都表示成零时间。前者是正常终态，后者必须返回错误。
- 只为扫描竞争做 CAS，却忽略执行完成与配置下发之间的 lost update。
- DBStore 保留 LastRun，但 MemoryStore 全量覆盖 LastRun，导致测试替身和生产行为不一致。
- 在 Handler 或调用方原地修改 `TaskConfig.Payload/Extra`。MemoryStore 只复制结构体，RawMessage 底层字节按只读所有权共享。
- 用 Complete 的状态条件实现禁用；禁用负责阻止新 claim，不应撤销已经成功执行中的 claim。
- `Disable` 只执行盲 UPDATE，或 UPDATE 后再用 `Count` 判断唯一 ID 是否存在；必须先查询状态，并直接检查更新结果的错误和影响行数。
- 用 `next_run` 同时表达 lease 和跨重试的业务计划时间；需要稳定回执时间的 Store 必须另存 nullable `scheduled_time`。
