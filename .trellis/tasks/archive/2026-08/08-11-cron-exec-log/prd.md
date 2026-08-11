# 新增 CronExecLog 模型记录每次 crontask 触发执行记录

## Goal

为 cron task 模式新增执行日志模型 `CronExecLog`，在每次 crontask 调度触发并调用 gRPC 接口后，将执行结果持久化到数据库，方便后续追踪和排查。

## Requirements

- 新增 `CronExecLog` gorm 模型，关联 cron_job，记录执行状态、耗时、错误信息
- 在 crontask 每次调度执行（`executeTask` 和 `RunNow`）完成后自动写入执行日志
- 自动注册 auto-migrate，开发/测试环境自动建表
- 不影响现有 handler 执行逻辑，使用装饰器模式包装

## Acceptance Criteria

- [ ] 新增 `CronExecLog` 模型，包含 job_id、task_code、task_name、scheduled_time、start_time、end_time、cost_ms、status（1-成功 0-失败）、error_message 字段
- [ ] 新增 `NewLoggingEventHandler` 包装函数，在 handler 执行前后写入 CronExecLog
- [ ] `ServiceContext` 中 auto-migrate 注册 CronExecLog，handler 改用 LoggingEventHandler
- [ ] 编译通过，dev 模式启动后自动建表
