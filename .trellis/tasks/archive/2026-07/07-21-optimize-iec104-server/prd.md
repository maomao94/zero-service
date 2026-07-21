# Optimize IEC104 server

## Goal

Unify the IEC104 server wrapper around `common/iec104/server/core.go`, keeping the remote server features while preserving go-zero logging integration used by `iecagent`.

## Requirements

- `common/iec104/server/core.go` provides the primary server API.
- Server construction accepts an explicit config object for host, port, IEC104 protocol settings, ASDU params, and logging.
- Default config remains usable without callers filling every field.
- Logging uses the existing go-zero `common/iec104.LogProvider` path by default.
- Existing IEC104 server callers are migrated to the unified config-based `NewServer` constructor.
- Connection lifecycle callbacks and current connection listing from the remote `core.go` are preserved.

## Acceptance Criteria

- [x] `core.go` exposes a server config and initializes defaults safely.
- [x] Server construction is represented by the unified `NewServer` implementation.
- [x] go-zero logging is installed by default without caller-provided log provider boilerplate.
- [x] `app/iecagent` starts the IEC104 server through the new config API.
- [x] Related package tests or builds pass.

## Notes

- User requested direct changes in `core.go` and a server config surface.
- Keep the change narrow; avoid broad IEC104 protocol refactors.
