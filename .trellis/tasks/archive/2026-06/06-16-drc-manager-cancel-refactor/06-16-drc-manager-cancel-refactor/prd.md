# Refactor DRC manager cancel lifecycle

## Goal

Refactor `app/djicloud/internal/drc.Manager` so heartbeat goroutine cancellation has a clear parent/child context lifecycle and cannot accidentally remove or cancel a newer DRC session worker.

## Requirements

- `Manager.Close` should rely on the manager parent context to stop `cleanLoop` and all heartbeat child contexts.
- Per-device heartbeat cancellation must remain available for `Disable`, re-enable, max-deadline expiry, and TTL orphan cleanup.
- A heartbeat goroutine must only remove its own registered worker on exit, not any newer worker registered for the same gateway.
- Existing public `Manager` APIs and DRC behavior should remain compatible.
- Comments should describe the revised lifecycle accurately.

## Acceptance Criteria

- [x] `Close` no longer iterates all per-device cancel functions after cancelling the manager context.
- [x] Old heartbeat goroutine cleanup cannot delete a newer heartbeat worker for the same gateway.
- [x] Orphan cleanup still cancels heartbeat workers whose cache entry has expired.
- [x] Focused DRC manager tests pass.

## Notes

- Lightweight refactor requested before production rollout; breaking internal implementation details is acceptable if public behavior stays stable.
