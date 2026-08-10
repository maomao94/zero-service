# crontask 调度规范

## 适用范围

修改 `common/crontask` 的 Scheduler、Task/Handler、Store、RRULE、`RunNow`，或实现 Trigger/ISP 存储适配器时读取。

## 核心状态与时间

- 非空 `TaskConfig.RRuleStr` 必须是至少包含 `DTSTART` 与 `RRULE` 的完整 RRULE Set；空字符串才表示一次性任务。业务生成和持久化不得使用裸 `FREQ=...` RRULE。
- `TaskConfig.NextRun` 在未 claim 时表示下一计划点；Store claim 后对应数据库列临时保存 lease 截止时间，返回给 handler 的副本中置零，不能再借它表示本次计划点。
- `TaskConfig.ScheduledTime` 表示当前 claim/retry 对应的原计划点；首次 claim 写入，重试保持稳定，成功完成或重新启用后清空。
- `TaskConfig.LastRun` 表示最近一次 handler 成功完成的实际时间；`LastScheduledRun` 表示该次成功周期执行对应的原计划点。手动 `RunNow` 只更新 `LastRun`。
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
- Trigger 与 ispagent 的 `scheduled_time` 在重试间保持稳定，代表首次原计划时间而非每次 claim/lease 时间。

## 反模式

- 完成时按 ID 整行更新，没有 `locked_until` CAS。
- `RunNow` 复用正常扫描完成路径，改变下一次计划或启停状态。
- 用 Redis 锁叠加补偿未证明的低并发 ID 风险，却不修数据库所有权条件。
- RRULE 错误被吞掉并写成零值，或 SQL `NULL` 被转换成远期时间。
- 同一 `RRuleStr` 列同时写入裸 RRULE 和完整 Set，迫使执行、描述和排障维护双解析分支。
- handler 从 `NextRun` 读取本次计划时间；claim 后应只读 `ScheduledTime`。

## 验证

```bash
go test ./common/crontask
go test ./app/trigger/internal/cronjob
go test ./app/ispagent/internal/crontask
go test -race ./common/crontask
```

测试至少覆盖过期 lease、并发完成、执行中 Disable、终止 RRULE、无效 RRULE、panic、`RunNow` 状态保持、成功/失败 `LastRun` 和 Delete 幂等。

## Scenario: 完整 RRULE Set 与执行时间状态

### 1. Scope / Trigger

- 生成、持久化、claim、重试、完成或展示周期任务规则与执行时间时适用。

### 2. Signatures

```go
type TaskConfig struct {
    RRuleStr        string
    NextRun         time.Time
    ScheduledTime   time.Time
    LastRun         time.Time
    LastScheduledRun time.Time
}
```

### 3. Contracts

- 写入：`RRuleStr == ""` 表示一次性任务；否则必须可由 `rrule.StrToRRuleSet` 解析，且 `GetDTStart()` 非零、`GetRRule()` 非 nil。
- claim：数据库 `next_run` 变为 `LockedUntil`，`scheduled_time` 保存首次原计划点；返回 Task 的 `NextRun` 为零、`ScheduledTime` 为原计划点。
- success：同一 CAS 写入未来 `next_run`、实际 `last_run`、原计划 `last_scheduled_run`，并清空 `scheduled_time`。
- retry：继续返回首次 `scheduled_time`，不能把上次 lease 截止时间当成计划点。

### 4. Validation & Error Matrix

- 裸 `FREQ=...` -> 校验错误。
- Set 缺少 DTSTART 或 RRULE -> 校验错误。
- Set 耗尽 -> `NextAfter` 返回零时间和 nil error。
- lease token 不匹配 -> `ErrNotFound`，不得写入成功时间。

### 5. Good/Base/Bad Cases

- Good: `DTSTART...\nRRULE...\nEXDATE...` 原样持久化并由同一字符串计算与描述。
- Base: 一次性任务使用空 `RRuleStr`，完成后 `NextRun` 为零。
- Bad: claim 后把 `Task.NextRun` 填成原计划时间；该字段与数据库 lease 语义发生分叉。

### 6. Tests Required

- Memory、Trigger DB、ISP DB 均断言首次 claim、lease 重试、成功完成和 Enable 清理。
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

## Scenario: RRULE 中文业务描述

### 1. Scope / Trigger

- 当调用方需要展示调度规则时，复用 `common/crontask` 的描述能力；不要在服务 Logic 中按 proto 或表字段复制第二套规则解释。

### 2. Signatures

```go
func DescribeRRule(value string) (string, error)
```

### 3. Contracts

- 非空 `value` 必须是至少包含 `DTSTART` 和 `RRULE` 的完整 RRULE Set，也可包含 `RDATE`、`EXDATE`；不接受裸 RRULE。
- 描述按简体中文稳定输出；`DTSTART` 存在时，时间边界和日期列表统一转换到它的时区。
- 同维度值是并集，不同 BY* 维度是交集；小时、分钟、秒按笛卡尔积解释。
- `INTERVAL` 表示从 `DTSTART` 相位推进的频率步长，`BYHOUR`、`BYMINUTE`、`BYSECOND` 再按 RFC 5545 的频率层级过滤或展开候选；`INTERVAL > 1` 且存在离散 BY* 条件时描述为“按 N 单位间隔”并保留条件，不得简写成均匀的“每 N 单位”。
- 只有 `INTERVAL = 1`，且低频规则的高位过滤与低位默认值可准确组成完整日内固定时刻集合时，才可等价描述为“每天 HH:mm…”。
- `WEEKLY` 在 `INTERVAL > 1` 或使用 `BYSETPOS` 时必须展示 `WKST`；前者由周起始决定间隔相位，后者由周起始决定每周期候选分组。
- 普通 `BYDAY` 与序号 `BYDAY` 混用时，`rrule-go` 会按内部普通/序号星期集合的交集筛选，不是同维度并集；描述器应返回 `ErrUnsupportedDescription`，不能将两组值用顿号连接。
- 周期条件只说明候选如何形成，不保证一定存在 occurrence；主句必须使用条件式执行表述。`UNTIL < DTSTART` 应保留两个边界并明确边界倒置，不能称为“有效期”，也不应通过遍历规则自行求解一般日历可达性。
- `RDATE` 是加入 RRULE 候选并集的额外候选，`EXDATE` 从该合并结果中排除；不能把每个 `RDATE` 直接描述为最终执行。
- `BYSETPOS` 按 `rrule-go` 当前频率的实际候选序列定位。若会扩展该频率 `timeset` 的显式时钟维度包含重复值，描述器必须返回 `ErrUnsupportedDescription`，不能去重显示后继续解释位置；只作为高位过滤的重复时钟值不占额外位置，不应误拒绝。
- `SECONDLY` 的每个周期仍有当前秒这个单一候选，因此 `BYSETPOS=1/-1` 可描述；不存在的位置只能条件式描述，测试不得迭代无 `UNTIL` 的永久空位置规则。
- 锁定的 `rrule-go v1.8.2` 对月作用域小于 `-5` 的序号星期可能发生负索引 panic。`DescribeRRule` 对 `MONTHLY` 及带 `BYMONTH` 的 `YEARLY` 规则安全拒绝该组合，但不得把限制扩散到 `parseRRuleSet`、`ValidateRRule` 或 `NextAfter`。
- 可视化以 `rrule.Options` 的归一化生效配置为准，不区分字段是用户显式声明还是由 `rrule-go` 根据 `DTSTART` 补齐；由于默认时、分、秒不会全部回写到 `Options`，时间描述和 `BYSETPOS` 候选校验必须结合 `DTSTART` 补齐。
- Set 的语法和组件处理完全采用 `parseRRuleSet` / `rrule-go` 的解析结果；描述器不得再次扫描原始 content lines、维护组件白名单或实现第二套 Set 形状校验。
- 描述器只消费已生成的 RFC 5545 string，不依赖 Trigger proto 或业务 model。

### 4. Validation & Error Matrix

- 空字符串 -> `"", nil`。
- RRULE 语法无效，或 Set 缺少 DTSTART/RRULE -> 解析 error。
- `BYYEARDAY`、`BYWEEKNO`、`BYEASTER` 或无法准确表达的组合 -> 可被 `errors.Is(err, ErrUnsupportedDescription)` 识别。
- `BYSETPOS` 候选 `timeset` 的扩展维度含重复值，或月作用域使用小于 `-5` 的序号星期 -> `ErrUnsupportedDescription`。
- 永久无候选或 `UNTIL < DTSTART` -> 仍返回描述且不承诺 occurrence；后者明确显示倒置边界。
- 合法且可描述的规则 -> 非空中文描述。

### 5. Good/Base/Bad Cases

- Good: `DTSTART;TZID=Asia/Shanghai:20260727T000000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0` -> 包含“每天 09:30 执行”。
- Base: 空规则用于一次性任务 -> 空描述。
- Bad: 把 `RDATE` 写成“额外执行”，或把永久空候选写成必然执行 -> 会把候选集合误报为最终 occurrence。

### 6. Tests Required

- 表驱动覆盖 YEARLY/MONTHLY/WEEKLY/DAILY/HOURLY/MINUTELY、INTERVAL、负数月日和序号星期。
- 断言多小时与多分钟展开为笛卡尔积。
- 断言 `INTERVAL > 1` 与稀疏 `BYHOUR` 保留 DTSTART 相位和过滤条件，不输出“每 N 小时”或“每天固定时刻”。
- 断言 `WEEKLY + BYSETPOS` 的 `WKST` 文案与实际 occurrence 分组一致，并拒绝普通/序号 `BYDAY` 混用的误导描述。
- 断言 YEARLY/MONTHLY/WEEKLY/DAILY 的默认日期或时刻与 `rrule-go` 实际 occurrence 一致，包括仅显式声明 `BYSETPOS` 的规则。
- 断言 UTC `UNTIL` 按 `DTSTART` 时区展示。
- RRULE Set 有 `RDATE`/`EXDATE` 时，`COUNT` 文案只能描述周期规则生成次数，不能声称最终总执行次数。
- 断言永久空日历交集使用有界 `UNTIL` 验证，倒置边界为空，并避免迭代无边界的永久空 `SECONDLY + BYSETPOS`。
- 用 occurrence 差分断言重复的 `timeset` 扩展值会改变 `BYSETPOS` 位置，并断言仅作高位过滤的重复值不会被误拒绝。

### 7. Wrong vs Correct

#### Wrong

```go
description := translateEnglish(rule.ToText())
```

#### Correct

```go
description, err := crontask.DescribeRRule(ruleSet.String())
```

对于 `FREQ=HOURLY;INTERVAL=2;BYHOUR=1,5,7`，正确描述为“按 2 小时间隔，小时=01/05/07…”，不能描述为“每 2 小时”或直接展开成每天固定时刻。
