# uix production framework foundation Implementation Plan

## Checklist

1. Load the parent task `prd.md`, `design.md`, and `implement.md` plus this child context.
2. Audit `StatusBar`, `HelpText`, and active module routing against `.trellis/spec/backend/uix-framework.md`.
3. Improve help/status rendering so long module instructions remain visible and terminal-width safe.
4. Add or extend tests for status/help visibility, active module help refresh path, and safe width behavior.
5. Verify all prompt modes and overlays still follow the canonical shell contract.
6. Keep example app buildable and update only if public usage changes.

## Concurrency

Implement this child directly or with at most one narrow subagent at a time. Do not run parallel child tasks.

## Validation

```bash
go test ./cli/uix/...
go vet ./cli/uix/...
go build ./cli/uix/_example
git diff --check
```

## Risk / Rollback

- Risk: changing status layout can alter every module screen. Roll back `components/statusbar.go` and tests if layout becomes unstable.
- Risk: dynamic help refresh can overcomplicate `Module`. Prefer existing `StatusMsg` before adding new public API.
