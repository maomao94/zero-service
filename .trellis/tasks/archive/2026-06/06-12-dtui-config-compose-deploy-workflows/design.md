# dtui config compose deploy workflows Design

## Boundary

This child owns `plugins/config`, `plugins/compose`, `plugins/deploy`, `internal/config`, and the deploy/compose helper functions needed by those modules.

## Config Flow

The config module is the canonical TUI editor for all config sections. It should manage compose dirs, deploy targets, and deploy packages consistently. External editor support can remain as an escape hatch, not the primary path.

## Compose Flow

Compose remains CLI-backed because Docker SDK does not expose compose. Commands must stay bounded by context timeouts and must capture stdout/stderr for display. Confirmation surfaces exact target path/service before execution.

## Deploy Flow

Deployment flow:

1. Choose target.
2. Choose folder/zip/package source.
3. Validate source and target config.
4. Backup current target content to configured backup dir.
5. Extract zip if needed to a safe temp path.
6. Copy source into container.
7. Record history success/failure.
8. Show output and recovery/rollback option.

## Safety

Backup is part of production readiness. If backup fails, deploy should stop before overwrite and show the failure unless a future task explicitly designs an override mode.

Compose down, deploy overwrite, backup cleanup, and rollback require second confirmation. Confirmation must show the target, expected impact, and backup/recovery status. Read-only config display, compose output view, deploy history, and log views should not require confirmation.
