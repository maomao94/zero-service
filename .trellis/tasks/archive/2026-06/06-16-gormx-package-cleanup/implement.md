# Implementation Plan

## Checklist

1. Load backend specs relevant to common DB helpers and quality checks.
2. Inspect all `common/gormx` source and tests for bugs, leaks, stale helpers, and weak tests.
3. Keep previous `batch.go` split, and adjust only if review finds better boundaries.
4. Fix concrete issues with minimal code changes and targeted tests.
5. Split misplaced tests by feature area.
6. Add concise `common/gormx/README.md` quick-start guide.
7. Run validation commands.
8. Inspect final diff and confirm unrelated `app/djicloud` changes were not modified by this task.

## Validation Commands

- `go test ./common/gormx`
- `go test ./app/djicloud/internal/hooks`
- `git diff --check -- common/gormx .trellis/tasks/06-16-gormx-package-cleanup`

## Risk Points

- `common/gormx` is a shared package; avoid broad behavior changes.
- `OpenWithRawDB` and trace registration affect connection lifecycle expectations.
- Tenant helper defaults are contract-sensitive; test before changing.
- Existing worktree has unrelated `app/djicloud` changes; do not revert or edit those files unless directly required.
