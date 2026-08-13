# crontask 调度规范

## 适用范围

修改 `common/crontask` 的 Scheduler、Task/Handler、Store、lease、`RunNow`，或实现 Trigger/ISP 存储适配器时读取。RRULE 解析、平移、批量查询和中文描述见 [rrulex RRULE 扩展规范](./rrulex-guidelines.md)。

## 核心状态与时间

- 非空 `TaskConfig.RRuleStr` 必须是至少包含 `DTSTART` 与 `RRULE` 的完整 RRULE Set；空字符串才表示一次性任务。业务生成和持久化不得使用裸 `FREQ=...` RRULE。
- `TaskConfig.NextRun` 在未 claim 时表示下一计划点；Store claim 后对应数据库列临时保存 lease 截止时间，返回给 handler 的副本中置零，不能再借它表示本次计划点。
- `TaskConfig.ScheduledTime` 表示当前 claim/retry 对应的原计划点；首次 claim 写入，重试保持稳定，成功完成或重新启用后清空。
- `TaskConfig.LastRun` 表示最近一次 handler 成功完成的实际时间；`LastScheduledRun` 表示该次成功周期执行对应的原计划点。手动 `RunNow` 只更新 `LastRun`。
- `TaskConfig.StartTime` / `EndTime` 表示规则生效边界，属于通用任务配置；适配器必须在公共配置和业务模型之间双向转换，零值按各存储的 NULL/空值约定处理。
- `TaskConfig.Extra` 当前仍是业务适配器的运行时载体，调度器不解析。业务表可以把其中字段平铺到专属列，并且不必额外持久化通用 `extra` blob；从数据库加载时由适配器重建运行时 `Extra`。
- 完整 Set 可以包含 `RDATE` 与 `EXDATE`；调度器不从业务 `specified_times`、`excluded_times` 或 `exclude_dates` 重建规则。首次 `NextRun`、Enable、成功完成推进和 Preview 必须消费持久化的同一个 `RRuleStr`。
- 数据库适配器用 SQL `NULL` 表达上述时间的零值，不使用远期哨兵时间。
- RRULE 无候选或已耗尽返回零值；语法无效返回 error，不能当作“自然结束”。
- Store 扫描只 claim 启用、`next_run` 非空且到期的任务。一次性/终止任务完成后不再参与扫描。

依据：`common/crontask/crontask.go`、`common/crontask/config.go`、`common/crontask/store.go`、`common/crontask/*_test.go`。

## Lease 与完成 CAS

- `LockAndFetch` 返回包含 `LockedUntil` 的 `TaskClaim`；Store 必须以 lease 作为 worker 所有权令牌。
- `Complete` 的更新条件至少包含任务 ID 和预期 `locked_until`。过期 worker 不得覆盖新 worker 的 `NextRun`、错误或完成结果。
- 调度结果由一次条件更新提交，检查 error 与 `RowsAffected`；竞争失败不是普通成功。
- 完成条件不应额外依赖当前启停状态：执行中的任务被禁用后，合法持 lease 的 worker仍要完成自己的本次结果，但不能重新启用任务。
- 有效锁时长经过 `ResolveLockTimeout` 规范化，最低为 30 秒；适配器不能绕过该下限。
- 管理 `Update` 必须保留 `ScheduledTime`、`LastRun`、`LastScheduledRun`；任务在途时不得覆盖作为 lease token 的 `NextRun`。
- 管理 `Update` 可以在同一事务中拆成两条 SQL：第一条用显式白名单更新普通配置，业务适配器可通过 `scheduled_time IS NULL` 拒绝会改变 RRULE 的在途更新；第二条更新 `next_run`。Trigger CronJob 的配置 UPDATE 零行返回 `ErrUpdate`，避免旧 worker 按旧 RRULE 回写下一计划点。

依据：`common/crontask/store.go`、`common/crontask/config.go`、`common/crontask/memory_store.go`、`app/trigger/internal/cronjob/db_store.go`、`app/ispagent/internal/crontask/db_store.go`。

## Handler 结果

- 成功：更新 `LastRun` 并根据 RRULE 计算/提交下一次时间。
- 普通 error：保留可重试调度语义，由 Store/Scheduler 约定下次执行；不要先把成功字段写入。
- `ErrDeleteTask`：删除任务；数据库 Delete 必须幂等。
- panic：Scheduler 必须 recover 为错误并释放/完成 claim，不能让调度循环退出。
- Store 的完成写入只拥有执行结果字段；配置、启停和其他管理字段由管理 API 拥有。

## `RunNow`

- `RunNow(ctx, taskCode)` 返回 `(traceID string, err error)`，是异步人工触发，使用正常 handler 但不 claim、不重算周期 `NextRun`、不修改 `Status`。
- traceID 通过 `trace.TraceIDFromContext(ctx)` 获取（go-zero 内置），同步返回给调用方以便追踪异步执行结果。
- 用 `context.WithoutCancel` 保留 trace/业务 metadata，让执行不随发起 RPC 返回而取消；执行自身仍要有可控超时和 panic 保护。
- 仅成功时更新 `LastRun`；失败或 panic 不伪造成功时间。
- 无周期计划的任务副本可用当前时间作为本次 handler 的执行时间，但不能持久化为新的周期计划。

## MaxDelay 任务级覆盖

- `SchedulerOptions.MaxDelay` 是调度器级兜底值，由 yaml 配置注入（默认 30m）。
- `TaskConfig.MaxDelay` 为任务级覆盖值；零值时走调度器默认值。
- `executeTask` 中取 `max(task.MaxDelay, s.maxDelay)` 语义：若 `task.MaxDelay > 0` 则用它替代 `s.maxDelay`；任务延迟超过该值跳过本次执行直接计算下次时间。
- 存储层使用**秒**（`int64`），API 层也使用秒；Go 内使用 `time.Duration`。
- 转换规则：
  - DB -> TaskConfig: `time.Duration(dbValue) * time.Second`
  - TaskConfig -> DB: `int64(d.Milliseconds() / 1000)`
  - TaskConfig -> API: `int64(d.Seconds())`

依据：`common/crontask/config.go`、`common/crontask/crontask.go`、`app/trigger/model/gormmodel/cron_job.go`、`app/trigger/internal/cronjob/convert.go`。

## 管理操作与适配器

- Enable/Disable 先按业务 task code 查找并保持幂等；不存在与更新失败使用明确错误，直接检查更新结果，不添加冗余 Count 查询。
- Delete 对不存在目标幂等成功，便于停用/清理重试。
- DB 与 Memory Store 必须实现同一空值、lease、完成和启停语义；不能让测试内存实现掩盖数据库竞争条件。
- Trigger 与 ispagent 的 `scheduled_time` 在重试间保持稳定，代表首次原计划时间而非每次 claim/lease 时间。
- Enable 当前用 `rrulex.ParseSet` + 官方 `Set.After(now, false)` 重新计算 `NextRun`。该路径不使用平移优化：高频且 DTSTART 很早的规则可能全历史扫描，MemoryStore 还会在写锁内完成计算；这是已知性能风险，不得描述成与 `rrulex.NextRuns` 相同复杂度。
- ISP Enable 当前不应用 `InvalidTimePredicate`，可能把无效窗口内的首个候选写入 `next_run`；新建/完成推进与重新启用的过滤语义尚不一致。修改该行为需独立任务和 Store 集成测试。

## 调度时间预览

- `Scheduler.PreviewNextRuns(task, after, count)` 只读任务配置，不访问 Store、不 claim、不执行 Handler，也不修改运行状态。
- Preview、首次 `CalcInitNextRun` 和成功完成后的 `computeNextRun` 都消费持久化的完整 `RRuleStr`；不得从业务平铺字段重建第二份规则用于其中某条路径。
- Scheduler 通过 `invalidPredicate(task)` 把 `InvalidTimePredicate func(task *TaskConfig, t time.Time) bool` 绑定成 rrulex 谓词，再调用 `rrulex.NextRuns(..., inc=false, ...)`。谓词只排除候选，不返回或重映射时间。
- ISP 的 `NewInvalidTimePredicate` 从运行时 `TaskConfig.Extra` 解析 `[InvalidStartTime, InvalidEndTime]` 闭区间；start、end 和窗口内候选无效，窗口前后有效，字段缺失/格式错误表示不启用过滤。
- 谓词必须是无副作用纯判断；rrulex 当前会在 inc 边界检查前调用它。永久规则上的永久拒绝谓词不会自然结束，调用方必须保证规则有界或谓词在有限未来恢复为 false。
- `count` 只统计过滤后接受的候选；规则自然耗尽返回已收集结果和 nil error，语法/结构错误向上传播，不能写成零值终态。
- RDATE/EXDATE、DTSTART/INTERVAL、DST 与安全平移的算法契约及测试要求见 `rrulex-guidelines.md`，不能在 Scheduler 层复制第二套实现。

依据：`common/crontask/crontask.go` 的 `PreviewNextRuns` / `computeNextRun`、`common/crontask/options.go`、`common/crontask/crontask_test.go`、`app/ispagent/internal/crontask/task_rule.go` 及其测试。

## 反模式

- 完成时按 ID 整行更新，没有 `locked_until` CAS。
- 把管理配置和 `next_run` 放进同一条无条件 UPDATE，覆盖调度器持有的 lease。
- `RunNow` 复用正常扫描完成路径，改变下一次计划或启停状态。
- 用 Redis 锁叠加补偿未证明的低并发 ID 风险，却不修数据库所有权条件。
- RRULE 错误被吞掉并写成零值，或 SQL `NULL` 被转换成远期时间。
- 同一 `RRuleStr` 列同时写入裸 RRULE 和完整 Set，迫使执行、描述和排障维护双解析分支。
- handler 从 `NextRun` 读取本次计划时间；claim 后应只读 `ScheduledTime`。
- 预览用 `Set.All()` 展开全部 occurrence，或按原始候选次数消耗 `count`，导致长期规则放大内存或连续非法区间返回数量不足。
- 谓词内自行解析规则并调用 `Set.After` 推进或确认，或谓词返回重映射时间：谓词只做排除，推进与单调性由 `rrulex.NextRuns` 保证。
- 把永久拒绝谓词用于无 COUNT/UNTIL 的永久 RRULE，却假定查询会自然结束。
- 把 Enable 的原生 `Set.After` 描述成已经使用平移优化，或忽略 ISP Enable 未应用无效窗口的现状。
- 在 `common/crontask` 里重新实现 rrule 解析/平移/迭代/描述：这些能力归属 `common/rrulex`。

## 验证

```bash
go test ./common/crontask
go test ./app/trigger/internal/cronjob
go test ./app/ispagent/internal/crontask
go test -race ./common/crontask ./app/ispagent/internal/crontask
```

测试至少覆盖过期 lease、并发完成、执行中 Disable、终止 RRULE、无效 RRULE、panic、`RunNow` 状态保持、成功/失败 `LastRun`、Delete 幂等，以及预览严格 after、连续非法区间、耗尽和 ISP 闭区间边界。RRULE 算法差分测试见 rrulex 规范。

## Scenario: 完整 RRULE Set 与执行时间状态

### 1. Scope / Trigger

- 生成、持久化、claim、重试、完成或展示周期任务规则与执行时间时适用。

### 2. Signatures

```go
type TaskConfig struct {
    RRuleStr         string
    StartTime        time.Time
    EndTime          time.Time
    Extra            json.RawMessage
    NextRun          time.Time
    ScheduledTime    time.Time
    LastRun          time.Time
    LastScheduledRun time.Time
}
```

### 3. Contracts

- 写入：`RRuleStr == ""` 表示一次性任务；否则必须通过 `rrulex.Validate`，即官方解析成功且 `GetDTStart()` 非零、`GetRRule()` 非 nil。
- 范围：`StartTime` / `EndTime` 必须由拥有规则编译的业务适配器写入并在模型转换时保留；它们不能被塞回 `Extra` 作为第二份权威值。
- 适配：业务模型已将扩展字段平铺为列时，可以不提供 `extra` 列，但 `ToTaskConfig` 必须从这些列重建 Handler 所需的运行时 `Extra`。
- claim：数据库 `next_run` 变为 `LockedUntil`，`scheduled_time` 保存首次原计划点；返回 Task 的 `NextRun` 为零、`ScheduledTime` 为原计划点。
- success：同一 CAS 写入未来 `next_run`、实际 `last_run`、原计划 `last_scheduled_run`，并清空 `scheduled_time`。
- retry：继续返回首次 `scheduled_time`，不能把上次 lease 截止时间当成计划点。

### 4. Validation & Error Matrix

- 裸 `FREQ=...` -> 校验错误。
- Set 缺少 DTSTART 或 RRULE -> 校验错误。
- Set 耗尽 -> 官方 `Set.After` 返回零时间、`rrulex.NextRuns` 返回空结果，均为 nil error。
- lease token 不匹配 -> `ErrNotFound`，不得写入成功时间。

### 5. Good/Base/Bad Cases

- Good: `DTSTART...\nRRULE...\nEXDATE...` 原样持久化并由同一字符串计算与描述。
- Base: 一次性任务使用空 `RRuleStr`，完成后 `NextRun` 为零。
- Bad: claim 后把 `Task.NextRun` 填成原计划时间；该字段与数据库 lease 语义发生分叉。
- Bad: 同时把开始/结束边界写入 `TaskConfig` 和业务 `Extra`，或删除表的 `extra` 列后不再从平铺列重建运行时 `Extra`。

### 6. Tests Required

- Memory、Trigger DB、ISP DB 均断言首次 claim、lease 重试、成功完成和 Enable 清理。
- Trigger/ISP 转换测试断言 `StartTime` / `EndTime` 往返不丢失；平铺业务模型不依赖数据库 `extra` 列，加载后仍能重建 Handler 所需字段。
- 断言 handler、日志和业务执行 ID 使用 `ScheduledTime`。
- 断言裸 RRULE 被拒绝，Trigger/ISP 生成值均含 DTSTART 与 RRULE。

### 7. Wrong vs Correct

#### Wrong

```go
scheduled := task.NextRun
```

#### Correct

```go
scheduled := task.ScheduledTime
```
