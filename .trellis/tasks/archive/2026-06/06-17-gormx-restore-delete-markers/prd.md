# Fix gormx restore delete markers

## Goal

Make `gormx.Restore` and `gormx.RestoreWithTenant` restore soft-deleted records for non-standard legacy schemas that may expose only one delete marker column.

## Requirements

- Preserve current restore behavior for standard GORM soft delete models using `deleted_at`.
- Preserve current legacy restore behavior when both `delete_time` and `del_state` exist.
- Support legacy models that expose only `delete_time` by clearing that column.
- Support legacy models that expose only `del_state` by setting it to `0`.
- Support Java-style models that expose `is_deleted` by setting it to `0`/false-equivalent through GORM update values.
- Do not update delete marker columns that are absent from the parsed model schema.
- Keep tenant-scoped restore filtering behavior unchanged.

## Acceptance Criteria

- [ ] `Restore` restores models with only `delete_time`.
- [ ] `Restore` restores models with only `del_state`.
- [ ] `Restore` restores models with only `is_deleted`.
- [ ] `Restore` continues to restore standard GORM `deleted_at` models.
- [ ] `RestoreWithTenant` keeps tenant filtering while using the same delete-marker selection logic.
- [ ] `go test ./common/gormx` passes.

## Out of Scope

- Adding callback registration options.
- Supporting arbitrary delete marker column names beyond `delete_time`, `del_state`, `is_deleted`, and standard `deleted_at`.

## Notes

- This is a lightweight task and remains PRD-only.
