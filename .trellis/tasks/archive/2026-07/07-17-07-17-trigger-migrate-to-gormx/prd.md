# trigger 迁移到 gormx 持久化

## Goal

Migrate `app/trigger` persistence from go-zero/sqlx generated models to service-local `gormx` GORM models.

The trigger plan tables and APIs should consistently use UUID/string primary keys and string relationship keys. MySQL and PostgreSQL SQL definitions, callback payloads, logic, and logs must be aligned with the string-ID contract.

## Scope

- In scope: `app/trigger` persistence code, trigger-local GORM models, service context DB wiring, trigger plan/callback proto fields and generated code when required, MySQL/PostgreSQL SQL definitions, query filters, transactions, log formatting, tests/build verification.
- In scope: removing trigger runtime dependency on go-zero/sqlx generated plan models for plan/batch/exec-item/exec-log persistence.
- In scope: preserving old-system-compatible `is_deleted` soft-delete semantics and `delete_time` audit-only semantics.
- Out of scope: unrelated services, broad root `model` regeneration not required by trigger runtime, deployed data migration execution, and non-trigger business behavior changes.
- Out of scope: changing queue/asynq semantics except where trigger IDs are carried in payloads or logs.

## Confirmed Decisions

- Trigger runtime persistence should use `gormx`, not go-zero/sqlx default database driver.
- Trigger-local GORM models live under `app/trigger/model/gormmodel`.
- Plan tables use `LegacyStringBaseModel` and UUID/string `id` values.
- Relationship columns that synchronize primary keys use strings: `plan_pk`, `batch_pk`, `item_pk`.
- Business unique IDs such as `plan_id`, `batch_id`, and `exec_id` remain strings.
- `is_deleted = 0` means active and `is_deleted = 1` means deleted.
- `delete_time` is delete audit time, not the delete-state source.
- MySQL and PostgreSQL SQL definitions must stay aligned for trigger plan tables.
- Logs and formatted strings must use string formatting for string IDs; `%d` must not be used for UUID/string IDs.

## Requirements

- Replace trigger plan persistence paths that use root go-zero/sqlx generated models with `gormx` operations against trigger-local GORM models.
- Keep service dependencies injected through `app/trigger/internal/svc.ServiceContext`; do not create ad-hoc database connections in logic.
- Use `gormx.DB` / `*gorm.DB` transaction boundaries for multi-table writes and updates.
- Ensure create-plan flow assigns UUID/string IDs before insert and propagates them through `plan_pk`, `batch_pk`, and `item_pk` relationships.
- Sync `model/sql/genSql.sql` and `model/sql/postgres.sql` trigger plan table definitions to string/UUID-compatible primary keys and relationship keys.
- Preserve or adapt indexes and unique constraints for the new string key columns.
- Update trigger proto/callback fields and generated code where request/response/payload fields still expose numeric plan/batch/item primary keys.
- Update all trigger logic, cron, planscope, and callback code to compare, query, and log string IDs correctly.
- Preserve optimistic lock behavior for tables that have `version`.
- Preserve `is_deleted = 0` filters in queries and GORM soft-delete behavior where `gormx.LegacyStringBaseModel` applies it.
- Use `tool.UUID()` for generated IDs and propagate UUID generation errors.
- Remove or stop using root go-zero/sqlx generated plan models from trigger runtime once equivalent gormx paths exist.
- Avoid unrelated business logic rewrites.

## Acceptance Criteria

- [ ] `app/trigger` runtime persistence for plan, plan_batch, plan_exec_item, and plan_exec_log uses `gormx` local GORM models rather than go-zero/sqlx generated model methods.
- [ ] `ServiceContext` exposes a gormx database dependency for trigger logic and no longer wires trigger plan SQLx generated model dependencies for runtime use.
- [ ] Trigger-local GORM models match MySQL and PostgreSQL schema column names, string key types, indexes, soft-delete fields, and version fields.
- [ ] `model/sql/genSql.sql` and `model/sql/postgres.sql` define trigger plan table `id`, `plan_pk`, `batch_pk`, and `item_pk` as string/UUID-compatible columns.
- [ ] Trigger proto/callback fields that carry plan/batch/item primary keys use string types where those keys are UUID/string.
- [ ] Generated trigger protobuf/go-zero files are regenerated if `trigger.proto` changes.
- [ ] Create-plan flow does not rely on `LastInsertId()` and uses generated UUID/string IDs across plan, batch, and exec item records.
- [ ] Query filters, joins, and associations use string IDs and retain `is_deleted = 0` semantics.
- [ ] Logs and `fmt.Sprintf` calls use `%s` for string IDs and have no `%d` misuse for plan/batch/item UUID fields.
- [ ] Optimistic-lock updates continue to check and increment `version` where applicable.
- [ ] `go test ./app/trigger/...` passes.
- [ ] `go test ./common/gormx` passes.
- [ ] `go test ./model` passes if root model package remains touched or compiled by dependencies.
- [ ] `go build ./app/trigger/...` passes.

## Notes

- This is a complex task and requires `design.md` and `implement.md` before implementation starts.
- Existing deployed databases with bigint IDs need a separate rollout/data-migration plan before production cutover.
