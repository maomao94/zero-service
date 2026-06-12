# dtui production host and navigation Implementation Plan

## Checklist

1. Load the parent task artifacts and confirm the `uix` child status/check result.
2. Add host construction helper if needed to make module registration testable without running Bubble Tea.
3. Wire containers and canonical config module into `main.go`.
4. Update startup messages to describe production `dtui` accurately.
5. Ensure no module constructor touches Docker daemon.
6. Add tests that inspect registered commands/modules without launching alt-screen UI.
7. Coordinate with docs child to update README after final module list is stable.

## Concurrency

Implement after the `uix` child. Do not run concurrently with Docker resource or workflow child tasks.

## Validation

```bash
go test ./cli/dtui/...
go build ./cli/dtui
git diff --check
```

## Risk / Rollback

- Risk: registering a module whose constructor has side effects could break no-Docker startup. Roll back that registration or fix lazy initialization.
- Risk: duplicate config modules confuse users. Keep only one wired config module.
