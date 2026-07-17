# common/gormx Legacy BaseModel 对齐 is_deleted 软删

## Goal

Align `common/gormx` Legacy GORM models with old-system-compatible soft-delete fields while keeping existing non-Legacy callback behavior stable.

Legacy GORM models should use `is_deleted` as the delete-state field (`0` = active, `1` = deleted) and keep `delete_time` only as delete audit time. Common field lifecycle behavior should move toward Legacy BaseModel orchestration instead of callback field scanning.

## Scope

- In scope: `common/gormx` Legacy GORM model mixins, Legacy BaseModel lifecycle hooks, gormx callback compatibility behavior, conservative restore behavior, tests, README/spec updates for this package.
- Out of scope: go-zero generated models under `model/*_gen.go`, trigger SQL/goqu query migration, database migration SQL for non-GORM tables, broad business model migration outside current GORM Legacy usage.
- Future work: trigger and generated DAO `del_state` migration will be handled during later GORM migration work.

## Confirmed Facts

- Legacy delete state is `is_deleted`, where `0` means not deleted and `1` means deleted.
- `delete_time` is audit data and must not be used by business code as the delete-state source.
- Current `common/gormx` Legacy soft delete uses `delete_time + del_state`.
- Current callbacks populate audit and tenant fields for non-Legacy mixin models; existing tests rely on this behavior.
- `SkipHooksCreate` and `SkipHooksUpdate` currently skip GORM model hooks but still run gormx callbacks.
- `Restore` must stay conservative because model field shapes vary.

## Requirements

- Replace the Legacy GORM soft-delete state field with old-system-compatible `is_deleted` while preserving `delete_time` audit time.
- Keep using `gorm.io/plugin/soft_delete` for soft-delete update conversion, default query filtering, and delete-time filling.
- Make `LegacyBaseModel` and `LegacyStringBaseModel` the lifecycle orchestration entry for Legacy GORM models.
- Keep mixins as capability providers; BaseModel should orchestrate lifecycle behavior.
- Keep callback registration as the package-wide entry point.
- Keep callback registration as an extension placeholder, but callbacks currently do not inject audit, tenant, or delete fields.
- Common field lifecycle is model-owned; non-Legacy models only get auto fill when they implement their own hooks.
- Prevent Legacy BaseModel models from receiving duplicate audit/tenant writes by keeping callbacks no-op.
- `Restore` should support known gormx soft-delete fields conservatively: clear `delete_time`, set `is_deleted` to zero, and keep limited `del_state` transitional support where tests/models still require it.
- Unknown or complex restore cases should be documented as business-owned `Unscoped()` + explicit `Updates(...)` operations.
- Do not migrate go-zero generated models or trigger business SQL in this task.

## Acceptance Criteria

- [ ] `LegacySoftDeleteMixin` maps delete state to `is_deleted` and `delete_time` remains audit-only.
- [ ] `IsDeleted()` for Legacy soft delete uses `is_deleted` state, not `delete_time.Valid`.
- [ ] `LegacyBaseModel` / `LegacyStringBaseModel` provide GORM lifecycle hooks for create/update/delete orchestration.
- [ ] Legacy string IDs still auto-generate UUIDs on create and keep preset IDs.
- [ ] Legacy create fills audit and tenant fields when context contains user/tenant data.
- [ ] Legacy update fills update audit fields only.
- [ ] Legacy delete fills delete audit fields while soft-delete plugin controls `is_deleted/delete_time`.
- [ ] Non-Legacy callback-based audit/tenant tests no longer expect automatic fill without model hooks.
- [ ] `Restore` restores Legacy soft-delete records with `is_deleted = 0` and `delete_time = NULL`.
- [ ] `go test ./common/gormx` passes.
- [ ] gormx README/spec describe Legacy `is_deleted` soft-delete semantics and callback compatibility boundary.

## Notes

- This is a complex task and requires `design.md` and `implement.md` before `task.py start`.
