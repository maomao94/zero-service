# dtui Docker resource modules

## Goal

Finish the Docker resource modules so `dtui` can be used as a production terminal UI for containers and images, not only a list/action demo.

## Confirmed Facts

- `plugins/containers` already supports list, start/stop toggle, stop, restart, force delete confirmation, refresh, and fetched logs.
- `internal/docker` already supports container inspect and stats streaming, but `plugins/containers` does not expose detail/inspect or stats views.
- `plugins/images` already supports list, remove confirmation, prune, and refresh.
- `internal/docker` already supports image save/export, tag, and history, but `plugins/images` does not expose these flows.
- Both modules lazy-create Docker clients and render loading/empty/error/success states.

## Requirements

- This child starts after host navigation makes the Docker resource modules reachable; it must not run concurrently with config/compose/deploy workflow implementation.
- Container module must expose production workflows for list, inspect/detail, logs, stats, start, stop, restart, remove, refresh, and clear error recovery.
- Image module must expose production workflows for list, detail/history, tag, save/export, remove, prune, and refresh.
- Destructive actions such as container remove, image remove, and image prune must require second confirmation and show clear target identity and impact.
- Long-running/streaming operations must be cancellable or closeable through module subviews.
- Docker daemon errors must be user-readable and recoverable through retry without restarting `dtui`.
- Views must preserve native terminal selection and avoid mouse-only interactions.

## Acceptance Criteria

- [ ] `/containers` lists containers and shows visible key instructions.
- [ ] Container detail/inspect view shows state, image, platform, mounts, networks, ports, env/cmd summary, and restart policy.
- [ ] Container logs view supports scrolling and follow mode.
- [ ] Container stats view shows current CPU, memory, network, block IO, PIDs, and recent history.
- [ ] Container start/stop/restart/remove actions refresh the list and show output/status.
- [ ] `/images` lists images and shows visible key instructions.
- [ ] Image detail/history view shows layer history in a readable terminal-safe view.
- [ ] Image tag and save/export workflows validate input/path and report output.
- [ ] Image remove/prune actions require confirmation and refresh the list.
- [ ] Focused tests cover module state transitions without requiring a real Docker daemon where feasible.

## Out Of Scope

- Compose and deploy workflows.
- Registry login/push/pull beyond local image management unless already trivial and explicitly planned.
- Remote Docker context management.

## Resolved Decisions

- Destructive Docker resource actions are default-visible/default-available, but execution requires second confirmation. Automated tests must not perform real destructive Docker operations against the user's daemon.

## Notes

- This child should start after host navigation makes the modules reachable.
