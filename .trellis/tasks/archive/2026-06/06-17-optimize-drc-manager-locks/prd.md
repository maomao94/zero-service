# Optimize DRC manager locks

## Goal

Reduce low-risk contention in the DRC Manager while preserving the current cache-based session lifecycle and state-machine behavior.

## Requirements

- Change `Manager.mu` from `sync.Mutex` to `sync.RWMutex` so read-only manager operations can use shared locking where appropriate.
- Update read-only manager status paths (`IsAlive`, `GetStatus`) to take `RLock` / `RUnlock` around cache access and state lookup.
- Change `State.seq` to `atomic.Int64` so sequence increments do not require `state.mu`.
- Preserve existing DRC lifecycle behavior for `Enable`, `Disable`, `OnDeviceHeartbeat`, `expireSession`, `cleanLoop`, and heartbeat workers.
- Keep `collection.Cache`, `state.mu`, and `workers sync.Map` in place for this task.
- Keep state-machine fields (`Enabled`, `SessionID`, `StartedAt`, `LastDeviceHeartbeat`, `MaxDeadline`) protected by `state.mu`.

## Acceptance Criteria

- [ ] `Manager.mu` is an `sync.RWMutex` and write paths still use exclusive `Lock`.
- [ ] `IsAlive` and `GetStatus` use manager read locks without changing their return semantics.
- [ ] `State.seq` uses `atomic.Int64`; `Enable` resets it, `GetNextSeq` atomically increments it, and `GetStatus` reads it atomically.
- [ ] Existing DRC manager tests pass.
- [ ] No broad refactor is introduced: cache TTL behavior, heartbeat worker lifecycle, and session expiration semantics remain unchanged.

## Out of Scope

- Removing `collection.Cache` or replacing it with a plain `map[string]*State`.
- Moving `OnDeviceHeartbeat` off the manager lock.
- Converting `LastDeviceHeartbeat` or state-machine fields to atomics.
- Changing external DRC APIs or hook behavior.

## Notes

- This is a lightweight implementation task; PRD-only planning is sufficient because the desired change is intentionally narrow and localized to `app/djicloud/internal/drc`.
