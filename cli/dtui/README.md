# dtui

`dtui` is currently a lightweight test host for `cli/uix`, the project chat-like TUI shell. This slice intentionally does not start Docker clients or require a Docker daemon.

## Quick Start

```bash
go run ./cli/dtui
go build ./cli/dtui
```

## Shell Interaction

| Input | Behavior |
| --- | --- |
| text + `enter` | Adds a user message and runs the local mock runner |
| `/` | Opens the command/module palette |
| `/test` | Opens the non-Docker shell exercise module |
| `@` | Shows reference extension placeholders |
| `#` | Opens the shared file picker/resource selector |
| `!` | Reserved for future safe shell execution; disabled in this build |
| `esc` | Closes overlays first, then exits an active root module |
| `ctrl+c` or `/exit` | Exits the TUI |

## Test Module

`/test` exercises shared `uix` behavior without external services:

| Key | Behavior |
| --- | --- |
| `m` | Opens shell-owned modal; `enter` confirms the action button, `esc` cancels |
| `#` | Opens the shared file picker from the prompt and routes selection to the module |
| `l` | Opens scrollable log/output view; `esc` returns to the module root |
| `a` | Appends a log line and success state |
| `e` | Shows an error state |
| `r` | Resets status, selected file, and logs |

## Architecture

```text
cli/dtui/
  main.go                 # wires the uix shell and /test module only
  plugins/test/           # non-Docker module for shell feature validation
  internal/docker/        # legacy Docker helpers kept for future module migration
  plugins/{containers,...}# legacy Docker modules not wired by the executable in this slice

cli/uix/
  app.go                  # Shell routing, prompt modes, overlays, commands, modules
  plugin.go               # Module, Command, HelpBinding contracts
  registry.go             # module and command registries
  timeline.go             # message timeline rendering
  runner.go               # assistant Runner interface and mock runner
  components/             # prompt, dropdown, modal, file picker, log viewer, state/panel, status bar
  theme/                  # shared colors, borders, truncation helpers
```

## Validation

```bash
go test ./cli/uix/... ./cli/dtui/...
go build ./cli/dtui
```

Docker-specific packages may still exist, but `cli/dtui` startup does not depend on Docker daemon state.
