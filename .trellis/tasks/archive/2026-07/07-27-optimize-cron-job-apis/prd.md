# 优化 CronJob 管理接口

## Goal

在现有 `CreateCronJob`、启用、禁用和删除能力之上，补齐 CronJob 的立即执行与查询能力，使调用方可以完整管理和查看 RRULE 周期任务。

## Confirmed Facts

- CronJob 对外契约定义在 `app/trigger/trigger.proto`。
- 当前已经提供创建、启用、禁用和软删除 RPC。
- 公共调度器已经提供 `RunNow(ctx, taskCode)`，其既有契约是立即异步执行一次且不改变周期计划。
- CronJob 使用独立的数据库模型和 `DBStore`，软删除记录默认不应出现在正常查询结果中。
- `RunNow` 不检查启用状态，因此禁用或周期已耗尽的任务也可以被人工立即执行一次。
- 现有项目分页约定使用 `int64 pageNum/pageSize`，零值应用默认分页；Trigger 约束页码最大 1000000、页大小最大 500，`gormx` 负责安全计算 Offset。
- CronJob 模型已为 `taskCode`、`deptCode`、`type`、`groupId` 和调度状态建立查询所需的唯一约束或索引。

## Requirements

- 新增 CronJob 立即执行 RPC。
- 新增 CronJob 详情 RPC。
- 新增 CronJob 分页列表 RPC。
- 列表支持 `taskCode/taskName` 模糊查询，支持 `status` 多选，支持 `deptCode/type/groupId` 精确查询。
- 详情和列表返回完整 CronJob 管理字段，包括调用方提交的 `payload/extra`；内部 RRULE 文本和在途 `scheduledTime` 不作为对外字段。
- 新接口复用现有 CronJob 存储与调度能力，并保持现有创建、启用、禁用、删除接口兼容。
- 立即执行不得改变任务原有的启用状态和下次周期执行时间。
- 详情与列表必须正确表达不存在下次执行时间的终止调度状态。
- `common/crontask.TaskStore` 提供按 ID 查询的 `GetByID`，各 Store 对缺失任务统一返回 `crontask.ErrNotFound`。
- 详情从 `TaskConfig` 转换为 Proto；列表 Logic 直接查询 Trigger GORM 模型，并立即转换为 `TaskConfig` 后复用同一个 Proto 转换。

## Acceptance Criteria

- [x] 调用立即执行接口后，目标 CronJob 被异步触发一次，周期 `NextRun` 和状态保持不变。
- [x] 立即执行、详情查询在 JobId 不存在或已软删除时返回一致、可识别的 NotFound 错误。
- [x] 详情接口返回完整且稳定的 CronJob 对外字段。
- [x] 列表接口支持分页，返回记录列表和总数，且不包含软删除记录。
- [x] 列表结果具有稳定排序，翻页不会因默认数据库顺序产生随机结果。
- [x] 列表筛选条件可组合使用，并且总数与同一组筛选条件一致。
- [x] 禁用或周期已耗尽的 CronJob 仍可通过立即执行接口触发一次。
- [x] proto 生成代码、server、logic、store 和相关测试保持同步。
- [x] CronJob 相关单元测试、竞态检查和 Trigger 包构建通过。
- [x] 极大页码不会造成 Offset 溢出或错误返回第一页；Trigger 全部分页请求具有统一范围校验。

## Out Of Scope

- 修改现有 RRULE 计算语义。
- 修改 CronJob 事件回调协议。
- 恢复或查询已软删除 CronJob。
- 本任务之外的调度器并发模型重构。

## Decisions

- 用户要求开始开发，采用规划阶段给出的推荐列表筛选范围。
- 对外时间沿用 Trigger 既有接口的 `YYYY-MM-DD HH:mm:ss` 字符串格式；可空时间使用空字符串。
- `CronJobPb` 必须返回数据库创建/更新时间；`TaskConfig` 增加对应审计时间，由数据库 Store 查询时填充。
- Proto 分页字段保留 `int64` 兼容类型，不改为 `uint64`；通过范围校验和 `gormx.PageParams` 同时保证非负语义与 Offset 安全。
