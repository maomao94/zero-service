# 用空值表示无下次调度

## Goal

用明确的空值表达周期任务已经没有下一次执行时间，移除“当前时间加 100 年”的时间哨兵，避免数据库时间范围问题和远期误触发。

## Confirmed Facts

- `TaskConfig.Status` 与 ISP `isenable` 对齐，表示配置启停状态，不应被周期自然耗尽改写。
- DB 扫描条件为 `status = enabled AND next_run <= now`，数据库 `NULL` 会自然排除在到期任务之外。
- `next_run` 使用 `timestamp` 并需要兼容 MySQL、PostgreSQL/GaussDB 和 SQLite，Go/Carbon 最大时间不是可靠的跨数据库值。
- 无下一次执行会出现在 RRULE 的 `COUNT`/`UNTIL` 耗尽，以及无效时间过滤后找不到后续时间两条路径。

## Requirements

- `TaskConfig.NextRun` 使用 `time.Time` 零值表示“没有下一次调度”。
- DB 模型使用可空时间类型；写入空值时持久化为 SQL `NULL`，读取 `NULL` 时恢复为空值。
- MemoryStore 和 DBStore 都不得扫描或锁定 `next_run` 为空的任务。
- RRULE 已耗尽或无效时间过滤后无候选时间时，持久化空 `next_run`，不得写入远期时间哨兵。
- ISP `status/isenable` 语义保持不变；周期耗尽后任务仍保留原配置启停状态。
- 规则构造失败必须返回错误，不能与“规则有效但已无下一次”混为一谈。
- API 列表中的空 `next_run` 输出空字符串，既有 protobuf 契约不变。
- 手动触发和现有任务处理路径必须安全处理空 `next_run`，不得 panic。

## Acceptance Criteria

- [x] 通用调度器在 RRULE 耗尽后把 `next_run` 更新为空，且后续扫描不再执行该任务。
- [x] DBStore 能往返保存有效时间和 SQL `NULL`，SQL `NULL` 任务不会被 `LockAndFetch` 选中。
- [x] MemoryStore 不会选中零值 `next_run` 任务，时间字段按值复制。
- [x] ISP 初始计划已耗尽时保存空 `next_run`；规则构造错误会返回下发错误。
- [x] 任务配置列表对空 `next_run` 返回空字符串。
- [x] `common/crontask` 与 `app/ispagent/internal/crontask` 相关测试通过。
- [x] `go vet` 覆盖受影响包并通过。

## Out Of Scope

- 不自动清洗数据库中历史写入的“100 年后”哨兵记录；如线上已有此类数据，另行制定可审计的数据迁移。
- 不修改 ISP 协议字段、protobuf 字段或任务启停枚举。

## Notes

- 用户确认目标是让无下一次计划的任务在业务生命周期内不再执行，并批准采用空值方案。
