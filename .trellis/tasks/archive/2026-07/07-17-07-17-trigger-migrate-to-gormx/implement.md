# Implementation Plan

## Checklist

1. Inspect current trigger persistence usage.
   - Find all `PlanModel`, `PlanBatchModel`, `PlanExecItemModel`, and `PlanExecLogModel` calls.
   - Find joins/raw SQL/goqu queries against trigger plan tables.
   - Find callback payload fields and log format strings involving plan/batch/item primary keys.

2. Align schema definitions.
   - Update `model/sql/genSql.sql` trigger plan tables to string/UUID primary and relationship key columns.
   - Update `model/sql/postgres.sql` with the same logical schema.
   - Preserve relevant unique constraints and indexes.

3. Align trigger-local GORM models.
   - Ensure `app/trigger/model/gormmodel/plan.go` matches SQL column names and types.
   - Keep `LegacyStringBaseModel` for all four trigger plan models.
   - Keep or remove `VersionMixin` according to actual `version` columns and update behavior.

4. Wire `gormx` into `ServiceContext`.
   - Use existing DB config and `common/gormx` helpers.
   - Remove trigger runtime dependency on generated SQLx plan models where replaced.

5. Migrate persistence logic.
   - Replace create-plan transaction with GORM transaction and preallocated UUIDs.
   - Replace find/list/update/delete/soft-delete logic with GORM queries and updates.
   - Preserve optimistic lock checks and error behavior.
   - Preserve `is_deleted = 0` filtering for custom joins and dashboards.

6. Migrate proto/callback/string ID fields.
   - Change trigger proto fields that carry primary keys to string.
   - Regenerate protobuf/go-zero files when proto changes.
   - Update logic mapping and validation code for string IDs.

7. Fix formatting and logs.
   - Replace `%d` for plan/batch/item UUID fields with `%s`.
   - Keep numeric formatting for status/count/duration fields.

8. Update specs if new executable contracts are learned.
   - Prefer `.trellis/spec/backend/gormx-guidelines.md` and `database-guidelines.md`.

## Validation

- `gofmt` on touched Go files.
- `go test ./app/trigger/...`
- `go test ./common/gormx`
- `go test ./model`
- `go build ./app/trigger/...`

## Review Gates

- Confirm trigger runtime no longer calls root generated SQLx plan model methods for plan persistence.
- Confirm MySQL and PostgreSQL SQL definitions agree on trigger plan table key types and indexes.
- Confirm callback/proto fields and log format strings are string-safe.
- Confirm no `LastInsertId()` remains in trigger plan creation paths.

## Risk Points

- Generated protobuf files can be large; inspect generated diff for unintended churn.
- Existing deployed bigint data needs a production migration outside this code task.
- GORM soft-delete filters apply only on model queries; custom raw/goqu joins must keep explicit `is_deleted = 0` filters.
- Optimistic-lock behavior may differ between hand-written SQLx and GORM `VersionMixin`; verify update paths carefully.
