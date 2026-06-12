# dtui production host and navigation

## Goal

Make `cli/dtui` a production host for the Docker terminal UI: all shipped modules must be wired, discoverable, navigable, and visibly documented from inside the running TUI while startup remains independent of Docker daemon availability.

## Confirmed Facts

- `main.go` registers `test`, `images`, `compose`, and `deploy`.
- `containers` and `config` modules exist but are not registered, so `/containers` and `/config` are currently unreachable.
- `README.md` states `dtui` is a test host and says Docker modules are legacy/not wired, which conflicts with `main.go`.
- `home` screen exists but is not a `uix.Module` and is not used by `main.go`.
- Existing startup messages mention `/test`, `/images`, `/compose`, `/deploy`, prompt modes, and Docker requirements.

## Requirements

- This child starts only after `uix-production-framework-foundation` is checked or explicitly deferred; it must preserve parent low-concurrency execution rules.
- Register every production module intended for the app: at minimum containers, images, compose, deploy, config, and test/demo.
- Ensure `/` command palette exposes clear names, descriptions, and aliases for all modules.
- Ensure entering each module shows visible module-specific commands, with special attention to container instructions.
- Keep Docker daemon optional at startup; module registration must not create Docker clients.
- Make startup/home messaging reflect the final production app rather than a test host.
- Decide and remove or quarantine duplicate legacy module paths so users do not face inconsistent config surfaces.

## Acceptance Criteria

- [ ] `go run ./cli/dtui` starts without Docker daemon running.
- [ ] `/containers`, `/images`, `/compose`, `/deploy`, `/config`, and `/test` resolve from the command palette and aliases.
- [ ] Container module entry visibly shows key commands such as select, start/stop, restart, remove, logs, refresh, and any new detail/stats actions.
- [ ] Host startup messages accurately describe the production modules and Docker-daemon behavior.
- [ ] There is one canonical config module exposed to users.
- [ ] Tests cover module registration and no-Docker host construction.

## Out Of Scope

- Completing the deeper container/image/compose/deploy business workflows; those live in sibling child tasks.
- Adding dynamic plugin discovery.

## Open Questions

- None blocking this child; module registration follows the parent scope.

## Notes

- This child depends on `uix` help/status improvements if the status line cannot currently show enough commands.
- If the `uix` child defers help/status changes, this child must implement only host wiring and leave UI-help polish blocked rather than inventing a second help system in `dtui`.
