# gormx: Remove ctx param from tool functions, source ctx from db.Statement.Context

## Goal

Remove the redundant `ctx context.Context` parameter from 9 gormx tool functions where the context can be derived from `*gorm.DB.Statement.Context` (which is already populated by `db.WithContext(ctx)` at the call site).

## Motivation

Current signatures like `func BatchUpdateByIdsWithTenant(ctx context.Context, db *gorm.DB, ...)` are unidiomatic for a GORM tool package: both `ctx` and `db` carry context, making the call sites unnecessarily verbose. Since `*gorm.DB` already stores the context via `db.WithContext(ctx)`, the separate `ctx` parameter is redundant.

## Requirements

1. **Drop `ctx` from 9 tool functions**: `UnscopedDeleteWithTenant`, `RestoreWithTenant`, `BatchInsertWithTenant`, `BatchUpdateByIdsWithTenant`, `BatchDeleteByIdsWithTenant`, `BatchDeleteByConditionWithTenant`, `Upsert`, `UpdateOrCreate`, `CreateRecord`
2. **Derive ctx from `db.Statement.Context`** inside each function instead of taking it as a parameter
3. **Keep `ctx` on functions that fundamentally operate on context values** (context getters/setters, logger interface implementations, `*DB` chained methods like `WithTenant`)
4. **Update all callers** in `app/djicloud/` (16 call sites across 4 files) to pass `db.WithContext(ctx)` instead of `ctx, db` as separate args
5. **Update all internal tests** in `common/gormx/`
6. **File organization**: consider extracting tenant-batch operations into a separate file or grouping related helpers

## Acceptance Criteria

- [ ] All 9 functions no longer accept `ctx context.Context` as a parameter
- [ ] Internal ctx retrieval uses `db.Statement.Context` (for `*gorm.DB` args) or `db.Statement.Context` (for `*DB` args)
- [ ] `go build ./...` passes
- [ ] `go test ./common/gormx/...` passes
- [ ] All callers in `app/djicloud/` updated and compile
- [ ] Context-value functions (`WithUserContext`, `GetTenantID`, `WithFullSQL`, etc.) unchanged
- [ ] `*DB` method signatures (`WithTenant`, `WithContext`, etc.) unchanged
- [ ] Logger interface implementations unchanged

## Notes

- `*gorm.DB.Statement.Context` is set by GORM's `WithContext(ctx)`; callers must ensure they call `db.WithContext(ctx)` before passing `db` to the tool function
- Functions taking `*DB` (Upsert series) can also use `db.Statement.Context` since `*DB` embeds `*gorm.DB`
