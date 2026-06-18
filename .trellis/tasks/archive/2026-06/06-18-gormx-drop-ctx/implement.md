# Implement: gormx ctx removal

## Checklist

### Step 1: Add `withTenantQueryFromDB` helper
- [ ] Add to `tenant_query.go`: `func withTenantQueryFromDB(db *gorm.DB) *gorm.DB`

### Step 2: Refactor `batch_tenant.go` (4 functions)
- [ ] `BatchInsertWithTenant`: drop `ctx`, use `db` directly (caller already `.WithContext(ctx)`)
- [ ] `BatchUpdateByIdsWithTenant`: drop `ctx`, remove `db.WithContext(ctx)` wrapper, use `withTenantQueryFromDB(tx)`
- [ ] `BatchDeleteByIdsWithTenant`: drop `ctx`, remove `db.WithContext(ctx)` wrapper, use `withTenantQueryFromDB`
- [ ] `BatchDeleteByConditionWithTenant`: drop `ctx`, remove `db.WithContext(ctx)` wrapper, use `withTenantQueryFromDB`

### Step 3: Refactor `delete.go` (1 function)
- [ ] `UnscopedDeleteWithTenant`: drop `ctx`, remove `db.WithContext(ctx)` wrapper, use `withTenantQueryFromDB`

### Step 4: Refactor `restore.go` (1 function)
- [ ] `RestoreWithTenant`: drop `ctx`, remove `db.WithContext(ctx)` wrapper, use `withTenantQueryFromDB`

### Step 5: Refactor `upsert.go` (3 functions)
- [ ] `Upsert`: drop `ctx`, use `db` directly (DB embeds *gorm.DB, ctx already on it)
- [ ] `UpdateOrCreate`: drop `ctx`, remove 3x `db.WithContext(ctx)` calls
- [ ] `CreateRecord`: drop `ctx`, remove `db.WithContext(ctx)` call

### Step 6: Update internal tests
- [ ] `batch_tenant_test.go`: update all call sites
- [ ] `delete_test.go`: update `UnscopedDeleteWithTenant` call sites
- [ ] `restore_tenant_test.go`: update `RestoreWithTenant` call sites
- [ ] `upsert_test.go`: update `Upsert`/`UpdateOrCreate`/`CreateRecord` call sites
- [ ] `context_test.go`: check if any test calls change

### Step 7: Update external callers
- [ ] `app/djicloud/internal/hooks/sys_status_up.go`: `UpdateOrCreate(ctx, tx, ...)` → `UpdateOrCreate(tx.WithContext(ctx), ...)` (3 calls) + `Restore(ctx, tx, ...)` → `Restore(tx.WithContext(ctx), ...)` note: Restore doesn't take ctx currently
- [ ] `app/djicloud/internal/hooks/event_notify_up.go`: `UpdateOrCreate` (2 calls) + `CreateRecord` (4 calls)
- [ ] `app/djicloud/internal/hooks/telemetry_up.go`: `UpdateOrCreate` (4 calls)
- [ ] `app/djicloud/internal/hooks/mqtt_drc_up.go`: `CreateRecord` (1 call)
- [ ] Note: `WithoutSQLTrace` and `WithFullSQL` signatures do NOT change

### Step 8: Verify
- [ ] `go build ./common/gormx/...`
- [ ] `go test ./common/gormx/...`
- [ ] `go build ./app/djicloud/...`

## Fallback / Safety

- If `db.Statement.Context` is nil inside a function, `withTenantQuery` will treat it as no tenant (same behavior as current when no tenant in ctx)
- All changes are mechanical (remove parameter, adjust body, update callers)
- Commit after tests pass; no intermediate state should be committable
