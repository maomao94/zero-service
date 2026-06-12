# Build chat-like uix shell

## Goal

Build `cli/uix` into a general-purpose, chat-like Go TUI glue shell inspired by OpenCode interactions. Treat `cli/dtui` as a non-essential host during this slice: after `uix` is solid, `dtui` should provide only a small test module that exercises the shell features.

The shell should support both future CLI LLM-agent workflows and business-module workflows. Users interact primarily through a persistent prompt and command entry, while `/` can switch into module screens. Docker business modules are out of scope for this slice.

## Confirmed Facts

- `cli/uix` already uses Bubble Tea, Bubbles, and Lipgloss with a framework app, bottom command bar, status bar, dropdown, file picker, modal, plugin registry, and plugin interface.
- Existing `cli/uix` is page/plugin oriented, not conversation oriented; it lacks a first-class message timeline, runner abstraction, command/action lifecycle, and module stack.
- Existing `cli/dtui` plugins each own too much UI glue: table styling, pending actions, modal confirmation handling, status/error presentation, and view layout.
- Compatibility with the existing `uix.Plugin` contract is not required.
- `dtui` should be ignored as a business module for this slice; it is only a test host for validating `uix` shell/module/component behavior.
- OpenCode interaction references that matter for this task: slash commands (`/`), file/reference autocomplete (`@`), optional shell command prefix (`!`), persistent prompt, keybind-driven navigation, and command palette style workflows.

## Requirements

- Make `uix` a chat-like glue shell rather than a Docker-specific or page-only framework.
- Keep a persistent bottom prompt as the primary interaction surface.
- Support a conversation/timeline main area for user, assistant, system, tool, and module-event messages.
- Support `/` command selection for shell actions and module entry.
- Support `@` reference/file autocomplete as a first-class input mode or explicit extension point.
- Reserve `#` resource selection and `!` shell-command semantics as explicit shell concepts; they may be minimal in the first implementation if the architecture is ready.
- Support module screens that fully occupy the main area while preserving the shared prompt, status bar, overlays, and back/escape behavior.
- Provide a small runner abstraction for future LLM-agent integration; first implementation may use a local mock/echo runner to validate streaming and message lifecycle.
- Provide a clear visual shell style for the main screen: timeline/module body, inline overlay area, status bar, and prompt must feel consistent and intentional.
- Provide reusable components or component contracts for command palette, file search/picker, modal dialogs, log/output view, status/error/loading feedback, and module panels.
- Provide a log/output design suitable for future tool calls and agent activity, including scrollable output and clear role/status presentation.
- Provide a `dtui` test module that exercises the shell features without depending on Docker daemon state.
- Improve UI quality through shared shell components and styles instead of each module hand-rolling layout and interaction glue.
- Update docs/specs that describe `uix` and `dtui` behavior if the implementation changes their contracts.

## Out of Scope

- Preserving backwards compatibility with the current `uix.Plugin` API.
- Adding real model provider configuration, API-key management, model selection, retries, or provider-specific streaming protocols.
- Implementing full OpenCode feature parity such as sessions, undo/redo file snapshots, sharing, provider auth, MCP, or agent orchestration.
- Rewriting or migrating Docker-specific `dtui` business screens.
- Requiring Docker daemon state for the test host.
- Enabling unrestricted `!` shell execution by default.

## Constraints

- Follow the project `uix` and Bubble Tea Trellis specs before implementation.
- Preserve text-selection friendliness by avoiding mouse capture unless there is a concrete reason.
- Keep the first implementation small enough to verify with `go test` and `go build` in this repository.
- Prefer simple interfaces and direct migration over compatibility adapters.

## Acceptance Criteria

- [ ] `cli/uix` exposes a new chat-like shell contract with message timeline, prompt, command registry, module registry/stack, overlay/modal handling, and runner extension point.
- [ ] The shell supports ordinary prompt submission, slash command selection, and module entry from `/` commands.
- [ ] At least one mock runner path demonstrates assistant/tool-style message updates without requiring external API credentials.
- [ ] `cli/uix` defines a restricted main UI style for timeline, module body, overlays, status bar, prompt, and empty states.
- [ ] `cli/uix` supports command palette, file search/picker, modal dialog, log/output viewer, and status/error/loading building blocks.
- [ ] `cli/dtui` builds as a test host with at least one test module that exercises shell commands, modal, file selection, logs/output, and module back behavior without Docker.
- [ ] Existing broken modal/action flows encountered during migration are either removed by the new action lifecycle or fixed in the migrated modules.
- [ ] The TUI remains keyboard-first: prompt submit, autocomplete navigation, module navigation, modal confirm/cancel, and ESC/back behavior are documented and work consistently.
- [ ] Relevant tests/build commands are run or explicitly documented if a command requires local Docker daemon state.
- [ ] `cli/dtui/README.md` or the canonical `uix` spec is updated to match the new architecture.

## Notes

- This is a complex task. It uses `design.md` and `implement.md` before implementation.
- Scope was revised after first implementation pass: prioritize `uix` quality and use `dtui` only as a test host before any Docker module rewrite.
