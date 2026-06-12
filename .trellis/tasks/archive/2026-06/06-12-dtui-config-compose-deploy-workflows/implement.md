# dtui config compose deploy workflows Implementation Plan

## Checklist

1. Load parent artifacts and apply the recorded second-confirmation safety decision.
2. Confirm host navigation has chosen the canonical config module.
3. Consolidate around one config module and identify any legacy `settings` code to leave unwired or remove later.
4. Extend config forms to support deploy packages and edit existing entries.
5. Add validation and tests for config CRUD helpers.
6. Improve compose output/log subview and path validation.
7. Add deploy backup-before-copy flow using existing Docker copy helpers or a safe new helper.
8. Record deploy history for both success and failure.
9. Expose deploy history and rollback/recovery guidance in the TUI.
10. Add focused tests around `PathType`, zip extraction safety, config CRUD, and deploy state transitions with fakes where feasible.

## Concurrency

Implement after host navigation and separately from Docker resource module work. This child touches shared config, Docker helper safety, and deploy behavior.

## Validation

```bash
go test ./cli/dtui/internal/config ./cli/dtui/internal/docker ./cli/dtui/plugins/config ./cli/dtui/plugins/compose ./cli/dtui/plugins/deploy
go test ./cli/dtui/...
go build ./cli/dtui
git diff --check
```

## Risk / Rollback

- Risk: deploy copy can overwrite real container files. Backup must be validated before copy.
- Risk: accidental overwrite/down/rollback. Require second confirmation before compose down, deploy overwrite, backup cleanup, and rollback.
- Risk: zip extraction path traversal. Ensure extraction rejects paths escaping destination before treating zip deploy as production-ready.
- Risk: config edits can corrupt user config. Use temp test config files and avoid real home config in tests.
