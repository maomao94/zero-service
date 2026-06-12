# Implementation Plan

## Checklist

1. Load `uix-framework.md`, `bubble-tea-tui-guide.md`, and relevant quality specs before editing implementation code.
2. Replace or reshape `cli/uix` core types around a chat-like `Shell` contract: timeline, prompt, command registry, module registry, overlays, and runner extension point.
3. Strengthen reusable UI components/helpers for message rendering, prompt mode detection, command selection, file search/picker, modal, log/output view, status/help text, state views, and safe width/height handling.
4. Add a mock runner flow so normal prompt submission creates user and assistant-style messages without external dependencies.
5. Restrict and document the main UI style: timeline/module body, inline overlay, status bar, prompt, empty/loading/error states.
6. Replace `dtui` runtime wiring with a single `/test` module that validates shell commands, modal, file selection, logs/output, status, and ESC/back behavior without Docker.
7. Update README or canonical spec to document the new shell/module/component contracts and key interactions.
8. Run focused tests/builds and fix issues caused by this task only.

## Validation Commands

- `go test ./cli/uix/...`
- `go test ./cli/dtui/...`
- `go build ./cli/dtui`

If Docker daemon-dependent tests are discovered, run non-Docker tests first and report daemon requirements explicitly.

## Review Gates

- Before `task.py start`: confirm PRD/design/implementation plan with the user.
- After `uix` shell compiles: verify the public contract is small and not tied to Docker.
- After `dtui` migration: verify modules are reachable through slash commands and ESC/back behavior is consistent.

## Risky Files

- `cli/uix/app.go`: top-level Bubble Tea routing and layout.
- `cli/uix/plugin.go`: likely replaced with new module/command contracts.
- `cli/uix/components/*`: shared UI behavior and prompt/overlay interactions.
- `cli/dtui/main.go`: shell wiring.
- `cli/dtui/plugins/*/plugin.go`: direct migration to new module contract.

## Rollback Points

- If the shell contract becomes too large, cut back to timeline + prompt + slash module entry + modal only.
- If full `dtui` migration is too broad for one pass, finish `containers` and `images` as proof points and leave clear compile-safe TODOs only if the package still builds.
