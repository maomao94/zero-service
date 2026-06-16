# Design: gormx Package Cleanup

## Boundary

Scope is limited to `common/gormx` code, tests, and a package-local `README.md`. Existing callers should keep using the same package-level API names.

## Package Organization

Group helpers by operational concern while preserving package `gormx`:

- `batch.go`: generic batch helpers.
- `batch_tenant.go`: tenant-aware batch helpers.
- `delete.go`: soft and hard delete helpers.
- `restore.go`: soft delete restore helpers and legacy-delete-field detection.
- `hook_helpers.go`: explicit hook-skipping helpers.
- `tenant_query.go`: internal tenant-aware query helper.
- Existing focused files (`callbacks.go`, `tenant_scope.go`, `logger.go`, `trace.go`, `upsert.go`, `pagination.go`, model files) remain as-is unless review finds a concrete issue.

## Compatibility

Do not rename exported functions or change parameter lists. File moves are safe because Go package exports are package-scoped.

Behavior changes are allowed only for concrete bugs and must be covered by tests.

## Review Targets

- Resource handling: `Open`, `OpenWithRawDB`, trace plugin registration, logger parameter filtering.
- Data semantics: tenant filtering default behavior, soft delete/restore for both standard `deleted_at` and legacy `delete_time/del_state`, callback version/audit behavior.
- Query safety: pagination order column validation, upsert fallback behavior, batch update ID handling.
- Test clarity: tests should live near feature names, not incidental historical file names.

## README Shape

`common/gormx/README.md` is a concise quick-start guide with short examples and caveats. It should not duplicate every function signature.

## Rollback

Because function names remain unchanged, rollback is file-level: revert `common/gormx` changes and remove `common/gormx/README.md` if tests reveal an unexpected behavior change.
