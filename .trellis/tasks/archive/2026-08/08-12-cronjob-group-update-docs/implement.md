# Implementation Plan

1. 修改 `trigger.proto`，为 Create/Update/Submit 响应增加 `group_id`，更新字段注释并运行 `app/trigger/gen.sh`。
2. 在 CronJob Logic/helper 中集中实现 group ID 创建默认值和稳定身份一致性校验；Create/Submit 新建生成 UUID，Update/Submit 更新保留原值并返回最终 group ID。
3. 收紧 DBStore 配置更新白名单，移除 `group_id/dept_code/type`，以 `scheduled_time IS NULL` 原子拒绝在途更新，并补充错误映射和 Store/Logic 测试。
4. 拆分 Plan 3 年与 CronJob 100 年编译范围，保持默认年份语义和单 RRULE 模型，补边界测试。
5. 给 `CronExecLog` 增加 message 字段，重构 Event Handler 私有调用结果以记录所有非 transport-error 回执 message，补 SQLite 日志落库测试。
6. 更新 `docs/trigger-rrule-api-guide.md` 和必要的 Trigger 总览，增加分组、更新边界、100 年范围、元旦/国庆和中秋处理示例。
7. 更新 Trellis Trigger/crontask 规范，记录稳定身份、跨度分离和回执 message 契约。
8. 运行格式化与验证：
   - `go test ./app/trigger/internal/cronjob ./app/trigger/internal/logic`
   - `go test ./common/crontask`
   - `go test -race ./app/trigger/internal/cronjob ./common/crontask`
   - 视生成影响运行 `go test ./app/trigger/...`
   - `git diff --check`
9. 审查生成文件和最终 diff，确认无多 RRULE 改造、无 Plan 100 年放宽、无非任务范围改动。

## Rollback Points

- Proto/Logic 响应变更可整体回滚，不复用新增字段号。
- Store 白名单与在途条件必须一起回滚，避免 Logic 与字段所有权不一致。
- `cron_exec_log.message` 数据库列可在代码回滚后保留。
