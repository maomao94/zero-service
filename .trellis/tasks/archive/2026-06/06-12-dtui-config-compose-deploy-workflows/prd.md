# dtui config compose deploy workflows

## Goal

Make `dtui` production-usable for configuration, Docker Compose operations, and frontend/container deployment workflows with TUI-first management, operation output, backup/history, and recovery safety.

## Confirmed Facts

- `internal/config` persists `ComposeDirs`, `DeployTargets`, and `DeployPackages` in JSON under the default `~/.dtui/config.json` path.
- `plugins/config` can list compose dirs, deploy targets, and deploy packages, but add forms currently cover compose dirs and deploy targets only.
- `plugins/settings` duplicates older config behavior and does not handle deploy packages.
- `plugins/compose` lists configured compose entries and runs `docker compose up -d` and `down` through `exec.CommandContext`.
- `plugins/deploy` lists deploy targets, uses `#` to select a source path, unzips zip files into backup dir `/_extract`, and copies files into the target container.
- Deploy history and backup cleanup helpers exist in `internal/config`, but the deploy module does not yet record history or perform backup/rollback before copy.

## Requirements

- This child starts after host navigation chooses and wires the canonical config module; it must not run concurrently with Docker resource module implementation.
- Provide one canonical config module for compose dirs, deploy targets, and deploy packages.
- Config module must support add, delete, edit, and validation for all config sections exposed by the app.
- Compose module must show configured projects, validate compose file paths, run up/down with confirmation, capture output, and expose logs/output in a readable subview.
- Deploy module must support target selection, source package/folder selection, zip extraction, backup before copy, deploy history recording, and rollback/recovery guidance.
- Deploy operations must not overwrite container content without backup and second confirmation.
- Compose down, deploy overwrite, backup cleanup, and rollback operations must require second confirmation with target and impact text.
- Config changes must be reflected by modules without requiring users to restart whenever feasible.

## Acceptance Criteria

- [ ] `/config` can add/delete/edit compose dirs, deploy targets, and deploy packages from the TUI.
- [ ] Config validation prevents empty names, empty required paths, invalid indexes, and ambiguous duplicate names where it affects UX.
- [ ] `/compose` shows configured projects and clear instructions for up/down/log/output/refresh.
- [ ] Compose up/down confirmation shows the exact command target and captured output is visible after execution.
- [ ] `/deploy` supports choosing a configured target and a folder/zip/package source.
- [ ] Deploy performs a backup or clearly fails before overwrite when backup is not possible.
- [ ] Deploy records success/failure history with action, target, detail, timestamp, and error.
- [ ] Deploy exposes recent history and rollback or recovery path in the TUI.
- [ ] Tests cover config CRUD and deploy path classification/extraction safety.

## Out Of Scope

- Remote deployment over SSH.
- Registry/image build pipelines.
- Kubernetes/Swarm orchestration.

## Resolved Decisions

- Destructive/deploy operations are default-visible/default-available, but compose down, deploy overwrite, backup cleanup, and rollback require second confirmation. Deploy must backup before overwrite when feasible and record success/failure history.

## Notes

- This child depends on host navigation choosing the canonical config module.
