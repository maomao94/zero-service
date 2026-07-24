# 设计：用空值表示无下次调度

## 数据契约

- `common/crontask.TaskConfig.NextRun` 保持 `time.Time`，零值表示无下一次调度，非零值表示实际调度时间。
- `TaskConfig.LastRun` 同样使用 `time.Time`，零值表示从未执行。
- `TaskStore.UpdateNextRun` 接收 `time.Time`，各存储实现保留相同零值语义。
- ISP GORM 模型使用 `sql.NullTime`，在转换层完成 `time.Time` 零值与 SQL `NULL` 的双向映射。

## 调度数据流

1. `LockAndFetch` 只返回启用且 `next_run` 非空、已经到期的任务。
2. 调度器执行任务后由 RRULE 计算下一次时间。
3. RRULE 返回零值或无效时间过滤耗尽候选时，调度器直接用零值调用 `UpdateNextRun`。
4. DBStore 将零值转换为 SQL `NULL`；MemoryStore 保存零值。
5. 后续扫描自然跳过该任务，但 `Status` 仍保留 ISP 配置的启停语义。

## 边界处理

- `computeNextRun` 遇到当前 `NextRun.IsZero()` 时以当前时间为基准。
- 正常扫描只会把非零 `NextRun` 的任务交给 handler；`RunNow` 手动触发无计划任务时，在任务副本上补当前时间，统一保证 handler 接收到实际执行时间。
- `NewTaskConfig` 返回 `(*TaskConfig, error)`：构造失败返回错误；有效规则没有候选时间则返回零值 `NextRun`。
- 转换成 Carbon 前使用标准库 `time.Time.IsZero()` 判断是否存在计划；Carbon `IsValid()` 只表示时间对象可用，不能代替空值判断。
- 配置列表只在 `sql.NullTime.Valid` 时格式化时间，否则保持 protobuf 字符串默认空值。

## 兼容性

- `next_run` 的 GORM tag 仍为无精度 `timestamp`，只改变 Go 扫描类型和可空语义。
- 既有 proto/API 不变；空值继续通过空字符串对外表示。
- DB 查询显式使用 `next_run IS NOT NULL AND next_run <= ?`，把“无下次调度不参与扫描”的契约写进查询。

## 风险与回滚

- `TaskConfig.NextRun` 和 `TaskStore` 是公共接口，所有仓库内实现与调用点必须一次性同步并通过编译检查。
- `time.Time` 按值复制，MemoryStore 不需要为调度时间维护指针深拷贝。
- 回滚代码不会自动恢复历史空值为远期时间；如需回滚，先评估线上空 `next_run` 数据和旧版本扫描行为。
