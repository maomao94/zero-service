# Production terminal UI framework Design

## Architecture

`uix` remains the framework boundary. It owns global shell behavior: prompt modes, slash command/module palette, timeline, shared overlays, status/help, sizing, and reusable components. It must not import `cli/dtui` or Docker packages.

`dtui` is the application boundary. It wires concrete modules, loads local config, owns Docker-specific workflows, and reports business operation output through `uix` module views/messages.

Child modules are independent production surfaces. Each module must implement `uix.Module`, use `SetSize` safe defaults, return asynchronous work through `tea.Cmd`, and keep global `/`, `@`, `#`, `!` routing with the shell.

## Module Boundaries

- `uix framework`: contracts, routing, layout, components, tests, and example app.
- `dtui host`: startup messages, module registration, command palette visibility, navigation, no-Docker startup behavior.
- `Docker resource modules`: containers/images views and actions using `internal/docker` lazy client APIs.
- `Config/compose/deploy workflows`: local JSON config management, compose CLI actions, deploy backup/extract/copy/history/rollback surfaces.
- `Docs/packaging/quality`: README, build script behavior, testdata, validation commands, and release/build hygiene.

## Data Flow

1. User types in the `uix` prompt.
2. `/` resolves a registered command/module; `#` opens file picker; `@` shows references; `!` appends disabled-shell warning.
3. Active module receives normal keys and module-scoped shell messages such as `FileSelectedMsg` and `ConfirmMsg`.
4. Module starts async work via `tea.Cmd` and reports results through module-specific messages.
5. Module renders local state; shell renders status/help, prompt, overlays, and active module body.

## Compatibility

- Keep exported `uix` contracts compatible unless a child task explicitly plans a breaking change and updates all users.
- Preserve no-Docker startup. Docker daemon is only touched after a Docker module action or module load that explicitly lists Docker resources.
- Preserve disabled `!` behavior until a separate shell-execution design exists.
- Keep current config file path shape (`~/.dtui/config.json`) unless a migration plan is added.

## Safety

Destructive and overwrite-style operations include container remove, image remove/prune, compose down, deploy copy/overwrite, backup cleanup, and rollback. They are available in the production UI by default, but execution requires a second confirmation that includes the target object, expected impact, and recovery/backup status where applicable.

Read-only operations such as list, inspect, logs, stats, image history, config display, and deploy history should not require confirmation.

Deploy/overwrite operations must backup first when feasible, record success/failure history, and show recovery or rollback guidance after execution.

## Rollback

- Child tasks can be reverted independently because each owns a narrow surface.
- For deploy workflows, production readiness requires backup/history before copy operations are treated as safe.
- For framework changes, rollback is validated by `go test ./cli/uix/...` and the `/test` module.
