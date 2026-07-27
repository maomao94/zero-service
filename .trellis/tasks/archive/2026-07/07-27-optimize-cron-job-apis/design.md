# CronJob 管理接口设计

## Architecture And Boundaries

数据流：

```text
trigger.proto
  -> app/trigger/gen.sh 生成 pb、校验、gRPC server 和 Swagger
  -> internal/logic
  -> internal/cronjob.DBStore / common/crontask.Scheduler
  -> cron_job / Eventstream CronJob handler
```

- `common/crontask.TaskStore` 增加 `GetByID(ctx, id) (*TaskConfig, error)`，MemoryStore、ISP DBStore 和 Trigger DBStore 保持相同领域错误语义。
- `RunCronJob` 由 Logic 按 `jobId` 查询任务，再调用 `CronJobScheduler.RunNow(taskCode)`。
- `GetCronJob` 消费 `TaskConfig`；`ListCronJobs` 在 Logic 查询 GORM 模型后立即转换为 `TaskConfig`，对外转换统一集中在 `internal/cronjob`。
- GORM 默认 scope 继续排除 `is_deleted=1` 的记录。

## RPC Contracts

- `RunCronJob(RunCronJobReq) returns (RunCronJobRes)`：请求只包含必填 `jobId`，接受禁用或已耗尽任务。成功表示异步执行已受理，不表示 Eventstream 已执行成功。
- `GetCronJob(GetCronJobReq) returns (GetCronJobRes)`：请求只包含必填 `jobId`，响应包含一个 `CronJobPb`。
- `ListCronJobs(ListCronJobsReq) returns (ListCronJobsRes)`：使用 `pageNum/pageSize`，支持 `taskCode/taskName` 模糊、`status` 多选以及 `deptCode/type/groupId` 精确筛选，响应返回 `cronJobs` 和 `total`。
- `CronJobPb` 返回创建/更新时间、Job/Task 标识、名称、优先级、锁超时、Payload、调用方 Extra、状态、NextRun/LastRun、机构/类型/分组/描述、开始/结束时间、结构化规则、排除日期和 ext1..ext5。
- `TaskConfig` 增加 `CreateTime/UpdateTime`，数据库 Store 在模型转换时填充；MemoryStore 保留调用方传入值。
- `rruleStr` 与 `scheduledTime` 是内部调度实现字段，不对外暴露。
- 时间统一格式化到秒；SQL NULL 的 `nextRun/lastRun/startTime/endTime` 返回空字符串。

## Query And Conversion

- 公共 `TaskStore` 仅补充通用 `GetByID`，不扩大 Scheduler 的查询职责。
- 列表 Logic 直接构建管理筛选条件并调用 `gormx.QueryPage`，排序为 `create_time DESC, id DESC`，确保稳定翻页。
- Trigger 全部分页请求校验 `pageNum=0..1000000`、`pageSize=0..500`；数据库分页保持 `int64` 到 `gormx.PageParams`，不在 Logic 提前转 `int`。
- `gormx.QueryPage` 先 Count 并短路越界页；无 Count 的 `QueryPageData` 通过安全 Offset 检查避免乘法溢出。
- 模糊条件使用参数化 `LIKE`；精确条件使用参数化 `=`；状态使用参数化 `IN`。
- Trigger Store 先把平铺数据库字段重建为 `TaskConfig.Extra`；Proto 转换再从 `TaskConfig` 恢复 `PlanRulePb`、排除日期和调用方 `bizExtra`。任一持久化 JSON 损坏时返回错误，不静默跳过列表项，以保持 `total` 与响应列表一致。

## Error Semantics

- 请求校验沿用生成的 `Validate()`。
- 立即执行和详情中的不存在/已删除任务映射为 `RECORD_NOT_EXIST`。
- Store、转换或分页失败映射为数据库/业务查询错误，并保留 cause。
- `RunCronJob` 仅同步报告查找/受理失败；异步 Eventstream 结果仍按 Scheduler 现有日志和重试语义处理。

## Compatibility And Rollback

- 只追加 RPC、消息和字段，不改已有字段编号或现有 RPC，保持 wire compatibility。
- 不新增数据库列和迁移。
- 回滚可移除新增 RPC/Logic/Store 查询方法，不影响现有周期扫描与创建、启停、删除能力。

## Tests

- DBStore：详情软删除隔离、组合筛选、分页总数、稳定排序、NULL 时间保留。
- Logic：立即执行确实触发 handler，禁用/耗尽任务可执行，周期状态和 NextRun 不变；详情/列表字段映射；不存在错误。
- Proto 生成后执行 Trigger 测试、race、vet、build 和 diff 检查。
