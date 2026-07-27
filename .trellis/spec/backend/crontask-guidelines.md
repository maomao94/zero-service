# crontask 调度规范

## 适用范围

修改 `common/crontask` 的 Scheduler、Task/Handler、Store、RRULE、`RunNow`，或实现 Trigger/ISP 存储适配器时读取。

## 核心状态与时间

- `TaskConfig.NextRun` 的 `time.Time{}` 表示无下一次调度；有效时间表示计划执行点。数据库适配器用 SQL `NULL` 保留同一语义，不使用远期哨兵时间。
- `LastRun` 零值表示从未成功执行；只有 handler 成功才更新。
- RRULE 无候选或已耗尽返回零值；语法无效返回 error，不能当作“自然结束”。
- Store 扫描只 claim 启用、`next_run` 非空且到期的任务。一次性/终止任务完成后不再参与扫描。

依据：`common/crontask/crontask.go`、`common/crontask/config.go`、`common/crontask/store.go`、`common/crontask/*_test.go`。

## Lease 与完成 CAS

- `LockAndFetch` 返回包含 `LockedUntil` 的 `TaskClaim`；Store 必须以 lease 作为 worker 所有权令牌。
- `Complete` 的更新条件至少包含任务 ID 和预期 `locked_until`。过期 worker 不得覆盖新 worker 的 `NextRun`、错误或完成结果。
- 调度结果由一次条件更新提交，检查 error 与 `RowsAffected`；竞争失败不是普通成功。
- 完成条件不应额外依赖当前启停状态：执行中的任务被禁用后，合法持 lease 的 worker仍要完成自己的本次结果，但不能重新启用任务。
- 有效锁时长经过 `ResolveLockTimeout` 规范化，最低为 30 秒；适配器不能绕过该下限。

依据：`common/crontask/store.go`、`common/crontask/config.go`、`common/crontask/memory_store.go`、`app/trigger/internal/cronjob/db_store.go`、`app/ispagent/internal/crontask/db_store.go`。

## Handler 结果

- 成功：更新 `LastRun` 并根据 RRULE 计算/提交下一次时间。
- 普通 error：保留可重试调度语义，由 Store/Scheduler 约定下次执行；不要先把成功字段写入。
- `ErrDeleteTask`：删除任务；数据库 Delete 必须幂等。
- panic：Scheduler 必须 recover 为错误并释放/完成 claim，不能让调度循环退出。
- Store 的完成写入只拥有执行结果字段；配置、启停和其他管理字段由管理 API 拥有。

## `RunNow`

- `RunNow(ctx, taskCode)` 是异步人工触发，使用正常 handler 但不 claim、不重算周期 `NextRun`、不修改 `Status`。
- 用 `context.WithoutCancel` 保留 trace/业务 metadata，让执行不随发起 RPC 返回而取消；执行自身仍要有可控超时和 panic 保护。
- 仅成功时更新 `LastRun`；失败或 panic 不伪造成功时间。
- 无周期计划的任务副本可用当前时间作为本次 handler 的执行时间，但不能持久化为新的周期计划。

## 管理操作与适配器

- Enable/Disable 先按业务 task code 查找并保持幂等；不存在与更新失败使用明确错误，直接检查更新结果，不添加冗余 Count 查询。
- Delete 对不存在目标幂等成功，便于停用/清理重试。
- DB 与 Memory Store 必须实现同一空值、lease、完成和启停语义；不能让测试内存实现掩盖数据库竞争条件。
- Trigger CronJob 的 `scheduled_time` 在重试间保持稳定，代表原计划时间而非每次 claim 时间。

## 反模式

- 完成时按 ID 整行更新，没有 `locked_until` CAS。
- `RunNow` 复用正常扫描完成路径，改变下一次计划或启停状态。
- 用 Redis 锁叠加补偿未证明的低并发 ID 风险，却不修数据库所有权条件。
- RRULE 错误被吞掉并写成零值，或 SQL `NULL` 被转换成远期时间。

## 验证

```bash
go test ./common/crontask
go test ./app/trigger/internal/cronjob
go test ./app/ispagent/internal/crontask
go test -race ./common/crontask
```

测试至少覆盖过期 lease、并发完成、执行中 Disable、终止 RRULE、无效 RRULE、panic、`RunNow` 状态保持、成功/失败 `LastRun` 和 Delete 幂等。

## Scenario: RRULE 中文业务描述

### 1. Scope / Trigger

- 当调用方需要展示调度规则时，复用 `common/crontask` 的描述能力；不要在服务 Logic 中按 proto 或表字段复制第二套规则解释。

### 2. Signatures

```go
func DescribeRRule(value string) (string, error)
```

### 3. Contracts

- `value` 可为单条 RRULE，也可为包含 `DTSTART`、`RRULE`、`RDATE`、`EXDATE` 的 RRULE Set。
- 描述按简体中文稳定输出；`DTSTART` 存在时，时间边界和日期列表统一转换到它的时区。
- 同维度值是并集，不同 BY* 维度是交集；小时、分钟、秒按笛卡尔积解释。
- 描述器只消费已生成的 RFC 5545 string，不依赖 Trigger proto 或业务 model。

### 4. Validation & Error Matrix

- 空字符串 -> `"", nil`。
- RRULE 语法无效或 Set 缺少 RRULE -> 解析 error。
- `BYYEARDAY`、`BYWEEKNO`、`BYEASTER` 或无法准确表达的组合 -> 可被 `errors.Is(err, ErrUnsupportedDescription)` 识别。
- 合法且可描述的规则 -> 非空中文描述。

### 5. Good/Base/Bad Cases

- Good: `DTSTART;TZID=Asia/Shanghai:20260727T000000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0` -> 包含“每天 09:30 执行”。
- Base: 空规则用于一次性任务 -> 空描述。
- Bad: 捕获描述错误后返回“自定义周期” -> 会掩盖规则和展示能力不一致。

### 6. Tests Required

- 表驱动覆盖 YEARLY/MONTHLY/WEEKLY/DAILY/HOURLY/MINUTELY、INTERVAL、负数月日和序号星期。
- 断言多小时与多分钟展开为笛卡尔积。
- 断言 UTC `UNTIL` 按 `DTSTART` 时区展示。
- RRULE Set 有 `RDATE`/`EXDATE` 时，`COUNT` 文案只能描述周期规则生成次数，不能声称最终总执行次数。

### 7. Wrong vs Correct

#### Wrong

```go
description := translateEnglish(rule.ToText())
```

#### Correct

```go
description, err := crontask.DescribeRRule(ruleSet.String())
```
