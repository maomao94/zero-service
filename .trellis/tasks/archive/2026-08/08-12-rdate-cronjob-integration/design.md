# CronJob Exact-Time Integration Design

## Scope

本子任务把公共精确时间字段接入 CronJob 管理和运行链路，不修改 Plan 或最终用户文档。

## Data Flow

Create/Update/Submit 把 `specified_times` / `excluded_times` 传入共享 CronJob 编译器。完整 `rrule_str` 是首次 next_run、完成推进和预览的权威来源。

`cron_job` 新增两份可空 JSON 文本列保存原始列表。转换层通过 `CronJobExtra` 运行时载体平铺/重建，`CronJobPb` 回显列表。

## Ownership

两个列表属于可变配置：Update/Submit 可以整体替换或清空；在途任务仍由 `scheduled_time IS NULL` 条件原子拒绝，不能部分更新模型列或 Set。

## Runtime

Scheduler、NextAfter 和 Preview 已读取完整 Set，因此不增加第二套 RDATE/EXDATE 解析。测试验证创建首次时间、完成后的推进和预览即可。

## Migration

开发/测试 AutoMigrate 增加列；生产 DDL 由发布流程执行。可空列兼容旧数据。
