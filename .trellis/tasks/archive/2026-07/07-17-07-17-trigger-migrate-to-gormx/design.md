# Design

## Boundary

This task migrates `app/trigger` runtime persistence to `gormx` for trigger plan tables only:

- `plan`
- `plan_batch`
- `plan_exec_item`
- `plan_exec_log`

The root go-zero/sqlx generated model package may remain in the repository for other callers or historical compatibility, but trigger runtime should stop depending on it for these tables once equivalent gormx operations exist.

## Data Model Contract

Trigger plan tables use string/UUID primary keys:

- `id varchar(64)` primary key.
- `plan_pk varchar(64)` references `plan.id` where present.
- `batch_pk varchar(64)` references `plan_batch.id` where present.
- `item_pk varchar(64)` references `plan_exec_item.id` where present.

Business IDs remain separate string columns:

- `plan_id`
- `batch_id`
- `exec_id`
- `item_id`

`LegacyStringBaseModel` provides `id`, `create_time`, `update_time`, `delete_time`, and `is_deleted`. `VersionMixin` is embedded on plan tables that keep optimistic locking through a `version` column.

## SQL Contract

Both `model/sql/genSql.sql` and `model/sql/postgres.sql` must define the same logical shape for trigger plan tables. Differences may remain for dialect syntax only.

Indexes and constraints should be preserved by intent:

- Unique `plan_id`, `batch_id`, `exec_id` constraints remain.
- Existing lookup indexes on `plan_pk`, `batch_pk`, `item_pk`, `plan_id`, `batch_id`, `item_id`, `status`, time fields, and core scan fields remain where still used.
- If index names mention `_pk`, they may stay because column names stay `_pk`; only column type changes.

## Persistence Design

`ServiceContext` should initialize and expose a `gormx.DB` or compatible `*gorm.DB` dependency using existing database configuration and `common/gormx` helpers.

Logic should use `db.WithContext(ctx)` consistently. Multi-table create/update flows should use GORM transactions. The create-plan flow must allocate UUIDs before inserts so relationship fields can be set without `LastInsertId()`.

`gormx` default soft-delete should own `is_deleted/delete_time` behavior through `LegacyStringBaseModel`. Query code must either use normal GORM model queries, which apply soft-delete filters, or explicitly filter `is_deleted = 0` when using raw/custom SQL.

## Proto And Callback Contract

All trigger API/callback fields that carry plan/batch/item primary keys must use `string` if they refer to `id`, `plan_pk`, `batch_pk`, or `item_pk`.

Fields that are business IDs stay string. Counters and statuses stay numeric.

After `trigger.proto` changes, generated protobuf, validation, server/client stubs, and logic signatures must be regenerated through the service generation path.

## Logging Contract

Any log or formatted message that prints plan/batch/item primary keys must use `%s` or structured fields with string values. `%d` is valid only for numeric status/count/duration fields.

## Compatibility And Rollout

This task changes runtime expectations from bigint IDs to UUID/string IDs. Existing deployed data requires separate migration before cutover:

- add/convert string ID columns or rebuild tables,
- backfill relationship fields,
- recreate indexes/constraints,
- update clients compiled against old int64 proto fields.

Repository SQL definitions should represent the target schema. Production migration execution is out of scope for this coding task.

## Rollback Shape

Rollback is restoring trigger service wiring and logic to sqlx generated model calls plus restoring SQL/proto string-ID changes. If SQL definitions change, deployment rollback must also consider database schema/data rollback separately.
