# Implementation Plan

## Checklist

1. Update `common/gormx/model_legacy.go`.
   - Map Legacy soft delete to `is_deleted` with `DeletedAtField:DeleteTime`.
   - Change `IsDeleted()` to use the delete flag only.
   - Move string ID generation into a capability method.
   - Add Legacy BaseModel lifecycle hooks and marker methods.

2. Update `common/gormx/model_audit.go` and `common/gormx/model.go`.
   - Add audit capability methods for uint and string audit mixins.
   - Add tenant capability method.
   - Avoid adding GORM hook methods to mixins.

3. Update `common/gormx/callbacks.go`.
   - Keep registration unchanged.
   - Make callbacks no-op extension placeholders.
   - Do not inject audit/tenant/delete fields globally.

4. Update `common/gormx/restore.go` only if needed.
   - Ensure `is_deleted` and `delete_time` restore path is covered.
   - Keep transitional `del_state` support conservative.

5. Update tests.
   - Adjust Legacy soft-delete tests from `del_state` to `is_deleted`.
   - Add or update Legacy create/update/delete audit and tenant tests.
   - Update non-Legacy callback tests to assert callbacks do not auto-fill fields without model hooks.
   - Preserve UUID generation tests.

6. Update docs/spec.
   - `common/gormx/README.md`: Legacy soft-delete fields and Restore boundary.
   - `.trellis/spec/backend/gormx-guidelines.md`: Legacy BaseModel lifecycle, `is_deleted`, callback no-op boundary.

## Validation

- `go test ./common/gormx`

If this reveals caller package failures from public API changes, inspect whether the failure is in scope before broadening the task.

## Risk Points

- GORM embedded hook dispatch for `LegacyBaseModel` and `LegacyStringBaseModel` must be verified by tests.
- Callback skip detection must work for embedded BaseModel values and pointers.
- `soft_delete.DeletedAt` with `DeletedAtField:DeleteTime` must fill `delete_time` while filtering by `is_deleted`.
- Avoid changing go-zero generated models or trigger SQL during this task.

## Review Gate Before Start

- PRD scope excludes generated DAO and trigger SQL.
- Design keeps callbacks registered as no-op extension placeholders.
- Tests include both Legacy BaseModel behavior and non-Legacy callback behavior.
