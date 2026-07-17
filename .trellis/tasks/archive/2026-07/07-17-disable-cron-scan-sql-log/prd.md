# 关闭 cron 扫表 SQL 日志

## Goal

Stop routine cron polling from emitting noisy GORM SQL logs for the `plan_exec_item` scan query.

## Requirements

- Suppress the recurring SQL log for the cron scan that selects pending `plan_exec_item` rows.
- Only suppress the scan `SELECT`; lock updates and follow-up plan/exec-item operations must keep normal SQL trace behavior.
- Keep existing cron scan behavior unchanged.
- Avoid globally disabling useful SQL logs unless the codebase has no narrower logging control.
- Keep the change small and localized.

## Acceptance Criteria

- [x] The cron scan query no longer emits the shown GORM SQL log during normal polling.
- [x] The cron scan still queries eligible `plan_exec_item` rows with the same filters and limit.
- [x] Targeted build/test or static verification passes for the touched package.

## Notes

- User-provided example shows repeated zero-row `SELECT ... FROM plan_exec_item ... ORDER BY RANDOM() LIMIT 1` logs from `trigger.rpc`.
