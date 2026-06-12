# dtui production host and navigation Design

## Boundary

This child owns `cli/dtui/main.go`, host construction, module registration, startup copy, and host-level tests/docs hooks. It does not complete each module's deeper business workflows.

## Host Shape

- Load config once at startup for modules that need config snapshots.
- Register production modules explicitly through `app.RegisterModule`.
- Keep module constructors side-effect-light; Docker clients must remain lazy.
- Keep `/test` as a diagnostic/demo module for validating `uix` behavior without Docker.

## Navigation

The slash palette is the source of truth for discoverability. Module names and aliases must match README and startup messages. Status/help must show the active module's local key map.

## Legacy Handling

`plugins/settings` overlaps with `plugins/config`. Prefer one canonical production module. If `settings` remains for history, do not wire it into `main.go` unless it is updated to the same contract and documented.
