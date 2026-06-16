# 整理 gormx 包职责和使用指南

## Goal

Improve `common/gormx` maintainability and reliability by reviewing the package for code bugs, cleanup opportunities, test organization gaps, and usage documentation needs.

The package should remain API-compatible for existing callers while becoming easier to navigate and safer to use.

## Requirements

- Review the full `common/gormx` package for practical correctness issues: resource handling, callback behavior, tenant filtering, soft delete/restore semantics, upsert behavior, pagination, logger and trace configuration.
- Fix concrete bugs or low-risk maintainability issues found during review without expanding into unrelated rewrites.
- Keep existing exported function names and behavior compatible unless a bug fix requires a clearly justified behavior adjustment.
- Organize tests so core areas are easy to locate: batch operations, tenant-aware batch operations, delete/restore helpers, hook helpers, callbacks, logger, pagination, upsert, tenant scope, and connection opening.
- Clean obsolete or misplaced code only when it has no independent behavior value and removal is covered by tests.
- Add `common/gormx/README.md` as a concise quick-start usage guide, not a full API reference.
- Do not modify unrelated `app/djicloud` changes already present in the worktree except when running validation tests.

## Acceptance Criteria

- [ ] `common/gormx` has been reviewed end-to-end for code bugs, resource leaks, confusing helpers, stale code, and test gaps.
- [ ] Any fixed issues have targeted tests or existing tests proving behavior.
- [ ] Test files are organized by feature area and no longer hide restore/delete/hook helper tests inside unrelated batch or legacy files.
- [ ] `common/gormx/README.md` documents quick usage for opening DB connections, model mixins, user/tenant context, transactions, batch helpers, soft delete/restore, upsert, pagination, logging/tracing, and tests.
- [ ] `go test ./common/gormx` passes.
- [ ] `go test ./app/djicloud/internal/hooks` passes to verify the current `gormx.Restore` caller.

## Notes

- User requested a concise quick-start guide.
- Worktree had pre-existing `app/djicloud` changes before this task; avoid altering or reverting them.
