# Production terminal UI framework

## Goal

Turn `cli/uix/` and `cli/dtui/` into a production-grade terminal UI project:

- `uix` is the reusable Bubble Tea shell/framework layer for chat-like command, module, prompt, overlay, timeline, and shared component behavior.
- `dtui` is the production Docker terminal UI application hosted on `uix`.
- Every shipped module must be reachable, visibly documented in the UI, resilient to missing Docker daemon/config, and covered by focused validation.

The user explicitly rejected a minimal closed loop. This task must deliver a complete, production-usable terminal project, split into independently verifiable child tasks.

## Confirmed Facts

- Product safety decision: destructive or overwrite-style operations are available by default, but must require a second confirmation before execution. Confirmation must show the target object, expected impact, and recovery/backup status where applicable.
- `cli/uix/` already owns shell routing, prompt modes (`/`, `@`, `#`, `!`), timeline, modal/file overlays, module registration, command registration, and runner abstraction.
- `cli/uix/components/` already contains table, list, help, progress, textarea, spinner, modal, dropdown, log viewer, state/panel, status bar, and chart wrappers.
- `.trellis/spec/backend/uix-framework.md` is the canonical contract for `cli/uix/` and `cli/dtui/`; it requires `uix` to stay generic and Docker/business clients to initialize lazily.
- `cli/dtui/main.go` currently registers `/test`, `/images`, `/compose`, and `/deploy`.
- `cli/dtui/plugins/containers` and `cli/dtui/plugins/config` exist but are not registered in `main.go`; this makes container instructions and config UI unreachable from the running app.
- `cli/dtui/plugins/settings` is an older duplicate-style config plugin and is not wired by the current host.
- `cli/dtui/README.md` is stale: it describes `dtui` as a lightweight test host and says Docker modules are not wired, while `main.go` already wires images/compose/deploy.
- Docker client creation is lazy in container/image/deploy modules, so startup can remain independent of Docker daemon state.
- Docker internals already provide container list/start/stop/restart/remove, logs, stats, inspect, image list/remove/tag/save/history/prune, compose CLI wrappers, deploy copy, config persistence, history helpers, and backup cleanup helpers.
- Existing container UI exposes list/start/stop/restart/remove/logs, but not inspect/details or stats even though internals exist.
- Existing image UI exposes list/remove/prune, but not save/tag/history even though internals exist.
- `!` shell execution is intentionally disabled by spec and must not become an implicit command runner in this scope.

## Child Task Map

- `06-12-uix-production-framework-foundation`: make the `uix` framework production-ready and reusable without Docker-specific coupling.
- `06-12-dtui-production-host-navigation`: make the `dtui` host, module wiring, navigation, and command/help visibility production-ready.
- `06-12-dtui-docker-resource-modules`: finish container and image resource modules to production depth.
- `06-12-dtui-config-compose-deploy-workflows`: finish config, compose, and deploy workflows with safety and recovery.
- `06-12-dtui-docs-packaging-quality`: make docs, packaging, test commands, and quality gates accurate and repeatable.

## Requirements

- Execution must be sequential and low-concurrency because the current model/session has concurrency limits; do not fan out multiple implementation/check subagents in parallel.
- `uix` must remain a generic framework layer: no Docker, provider, shell-execution, deployment, or app-specific logic in `cli/uix/`.
- `uix` must provide stable module, command, prompt, overlay, layout, status/help, and shared component behavior suitable for production TUI apps.
- `dtui` must start reliably with `go run ./cli/dtui` even when Docker is not installed or daemon is stopped.
- `dtui` must expose all production modules through `/` command palette and aliases; no production module may remain unreachable.
- `dtui` must make the current module's commands visible in the terminal UI, including container instructions.
- Docker modules must use lazy clients, clear loading/empty/error/success states, non-blocking Bubble Tea commands, confirmations for destructive operations, and readable action output.
- Delete/cleanup, overwrite/deploy, compose down, backup cleanup, and rollback operations must require second confirmation; read-only operations such as list, inspect, logs, stats, and history must not add unnecessary confirmation friction.
- Configuration, compose, and deployment flows must be operable from the TUI without editing JSON as the only path.
- Deployment must include safety controls for backup/rollback/history before it is considered production-ready.
- Documentation must match shipped behavior, describe module keys, describe safety behavior, and provide build/test/packaging commands.
- Validation must include focused tests for `uix`, module wiring, Docker-unavailable behavior, config persistence, and packaging/build commands.

## Acceptance Criteria

- [ ] Child tasks are implemented and checked one at a time in the planned order; no broad parallel agent fan-out is used.
- [ ] `go run ./cli/dtui` starts the TUI without requiring Docker daemon availability.
- [ ] Every destructive or overwrite-style operation requires second confirmation with target object and impact text before execution.
- [ ] `/` palette lists every shipped module with useful descriptions and aliases: at minimum `test`, `containers`, `images`, `compose`, `deploy`, and `config` or equivalent production config module.
- [ ] Entering `/containers` shows visible key instructions for container operations in the status/footer or help surface.
- [ ] Every module has clear loading, empty, error, and success states and handles terminal resize safely.
- [ ] Container module supports production workflows for list, detail/inspect, logs, stats, start, stop, restart, and remove with confirmation where needed.
- [ ] Image module supports production workflows for list, detail/history, tag, save/export, remove, and prune with confirmation where needed.
- [ ] Config/compose/deploy modules support full TUI workflows for managing compose directories, deploy targets, deploy packages, compose up/down/log output, deploy backup, deploy execution, deploy history, and rollback or recovery guidance.
- [ ] `uix` shell tests cover prompt modes, command/module routing, disabled `!`, overlays, status/help rendering, and safe sizing.
- [ ] `dtui` tests cover host module registration, no-Docker startup, config CRUD, and module behavior that does not require a real daemon.
- [ ] README and packaging scripts accurately describe the current app, modules, keys, safety model, validation commands, and build outputs.
- [ ] Required validation passes: `go test ./cli/uix/... ./cli/dtui/...`, `go build ./cli/dtui`, `go vet ./cli/uix/... ./cli/dtui/...`, and `git diff --check`.

## Out Of Scope

- Enabling `!` arbitrary shell command execution.
- Adding Kubernetes, remote cluster management, SSH deployment, registry authentication flows, or cloud provider integrations.
- Replacing Docker SDK/CLI internals with a new dependency stack unless required by a concrete bug.
- Building a general plugin marketplace or dynamic plugin loader.
- Adding LLM/provider integration to `uix` beyond the existing `Runner` interface.

## Resolved Decisions

- Destructive and overwrite-style operations are default-visible/default-available in production UI, but execution requires a second confirmation. Deploy/overwrite flows must backup first when possible, record history, and present recovery or rollback guidance.

## Notes

- This parent task owns the cross-child requirements, dependency order, and final integration review. Implementation should happen in child tasks unless a parent-level integration patch is explicitly needed.
