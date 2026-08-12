# Implementation Plan

1. 修改 `trigger.proto`，为 Plan 计算/创建和 CronJob Create/Update/Submit 请求增加各最多 1000 项的 `specified_times` / `excluded_times`，为 CronJob 管理视图增加回显字段，并运行 `app/trigger/gen.sh`。
2. 扩展共享 Schedule 编译入口：解析上海时区精确时间、执行闭区间校验、分别写入 `Set.RDate` / `Set.ExDate`，并让 `exclude_dates` 排除当天所有指定时间。
3. 串联 Plan `CalcPlanTaskDate` / `CreatePlanTask`，确认日期预览、RRULE 原文和 Batch/ExecItem 展开包含 RDATE，且保持 3 年与 5000 项限制。
4. 串联 CronJob Create/Update/Submit helper 和 Logic，新增模型 rdates 列及转换、清空、响应回显，保持在途更新保护与 100 年限制。
5. 验证公共 `crontask` 的 NextAfter、预览和描述直接消费包含 RDATE 的持久化 Set；仅在缺少覆盖时补定向测试，不复制解析逻辑。
6. 更新 Trigger 总览和统一 RRULE 场景指南，说明 Plan/CronJob 两种模式、指定时间、精确时间排除、范围控制、去重和整日排除行为。
7. 更新 Trigger/crontask/contract-generation/GORM Spec 中新增的跨层契约。
8. 运行 `app/trigger/gen.sh`、目标包测试、Trigger 全量测试、相关 race test、JSON/链接检查和 `git diff --check`，审查生成 diff 与迁移边界。

## Rollback Points

- Proto、编译参数、Logic、模型和文档作为一个兼容单元回滚。
- CronJob 新增可空数据库列可以保留，不要求破坏性回滚。
- 不修改 `rrule-go` 或公共 Set 解析器，避免依赖 fork。
