# Design: gormx ctx removal

## Boundary

Scope: `common/gormx/` package + callers in `app/djicloud/`.

## Pattern

Before:
```go
func BatchUpdateByIdsWithTenant(ctx context.Context, db *gorm.DB, model any, updates []Ups) error {
    return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        ...
        q := withTenantQuery(ctx, tx.Model(model).Where("id = ?", id))
        ...
    })
}
// call: gormx.BatchUpdateByIdsWithTenant(ctx, db, &Device{}, updates)
```

After:
```go
func BatchUpdateByIdsWithTenant(db *gorm.DB, model any, updates []Ups) error {
    return db.Transaction(func(tx *gorm.DB) error {
        ...
        q := withTenantQueryFromDB(tx.Model(model).Where("id = ?", id))
        ...
    })
}
// call: gormx.BatchUpdateByIdsWithTenant(db.WithContext(ctx), &Device{}, updates)
```

## Key decisions

### 1. Context source: `db.Statement.Context`

GORM populates `db.Statement.Context` when `db.WithContext(ctx)` is called. This is the canonical way to retrieve context from a `*gorm.DB` instance. All 9 functions use `ctx` exclusively for:
- Calling `db.WithContext(ctx)` → caller now does this before passing `db`
- Calling `withTenantQuery(ctx, db)` → replaced by `withTenantQueryFromDB(db)` which reads `db.Statement.Context`
- Calling `GetTenantID(ctx)` → already available through `withTenantQueryFromDB`

### 2. Internal helper: `withTenantQueryFromDB`

Add to `tenant_query.go`:
```go
func withTenantQueryFromDB(db *gorm.DB) *gorm.DB {
    return withTenantQuery(db.Statement.Context, db)
}
```

### 3. Transaction callback context preservation

When calling `db.Transaction(func(tx *gorm.DB) error { ... })`, GORM passes the parent's `Statement.Context` into `tx.Statement.Context`. So `withTenantQueryFromDB(tx)` works correctly inside transaction callbacks.

### 4. Functions NOT changed

| Category | Functions | Reason |
|---|---|---|
| `*DB` chained methods | `WithContext`, `WithTenant`, `WithTenantStrict`, `WithDeleted`, `WithTenantDeleted` | Need ctx to SET it, not read it |
| Context value producers | `WithUserContext`, `WithTenantContext`, `WithUserAndTenantContext`, `WithStringUserAndTenantContext`, `WithFullSQL`, `WithoutSQLTrace` | Core function is context mutation |
| Context value readers | `GetUserContext`, `GetUserID`, `GetUserIDAs`, `GetUserIDText`, `GetUserName`, `GetTenantID` | Core function is context extraction |
| Scope factories | `TenantScope`, `TenantScopeStrict`, `TenantScopeWithDelete`, `TenantEq`, `TenantNotEq`, `TenantIn` | Closures that return `func(*gorm.DB) *gorm.DB` |
| Logger interface | `ParamsFilter`, `Info`, `Warn`, `Error`, `Trace` | Must match `logger.Interface` from gorm |
| Package-level no-tenant helpers | `SoftDelete`, `UnscopedDelete`, `BatchInsert`, `BatchUpdateByIds`, `BatchDeleteByIds`, `BatchDeleteByCondition`, `SkipHooksUpdate`, `SkipHooksCreate` | No ctx param to begin with |

## File organization

Current files to modify:
- `batch_tenant.go` — 4 functions (drop ctx, use `withTenantQueryFromDB`)
- `delete.go` — 1 function (drop ctx)
- `restore.go` — 1 function (drop ctx)
- `upsert.go` — 3 functions (drop ctx, remove `WithContext` calls since caller already attached ctx)
- `tenant_query.go` — add `withTenantQueryFromDB`

External callers:
- `app/djicloud/internal/hooks/sys_status_up.go` — UpdateOrCreate x3, Restore x1
- `app/djicloud/internal/hooks/event_notify_up.go` — UpdateOrCreate x2, CreateRecord x4
- `app/djicloud/internal/hooks/telemetry_up.go` — WithoutSQLTrace x1, UpdateOrCreate x4
- `app/djicloud/internal/hooks/mqtt_drc_up.go` — CreateRecord x1
- `app/djicloud/internal/svc/device_online_refresher.go` — WithoutSQLTrace, WithFullSQL (no change needed)

## Tradeoffs

| Pro | Con |
|---|---|
| Cleaner signatures: `(db, model, ...)` vs `(ctx, db, model, ...)` | Callers must call `.WithContext(ctx)` — but they already do this in 99% of cases |
| More GORM-idiomatic | If called without `WithContext`, tenant filtering silently misses (ctx == nil → no tenant filter) |
| No breaking contracts — all functions extract ctx from the same `db.Statement.Context` consistently | `withTenantQueryFromDB` adds indirection |

## Rollback

All changes are signature-only refactoring. To roll back: revert the commit. No data migration, no config changes.
