# Design: Chat-like uix Shell

## Architecture

`cli/uix` becomes a small shell framework with five core responsibilities:

- Layout: persistent main area, prompt, status bar, and overlays.
- Interaction routing: prompt modes, slash commands, references, resources, modal focus, and ESC/back stack.
- Conversation state: timeline messages for user input, assistant output, system notices, tool results, and module events.
- Module hosting: business modules can take over the main area while still using shared prompt, status, command registry, and overlays.
- Component design: command palette, file search/picker, modal dialogs, log/output viewer, status/error/loading states, and panel style should be reusable and visually consistent.

The old page-first `Plugin` contract can be replaced. Current Docker-specific `dtui` plugins are not the design target for this slice. `dtui` should temporarily be a test host for validating the new shell.

## Core Types

Proposed shape, exact names may change during implementation:

- `Shell`: top-level Bubble Tea model, replacing or reshaping `FrameworkApp`.
- `Message`: timeline item with role, content, optional status, timestamp, and optional metadata for tool/module output.
- `Timeline`: viewport-backed message list with scroll controls and append/update helpers.
- `Prompt`: multiline input component with mode detection for normal text, `/`, `@`, `#`, and reserved `!`.
- `Command`: slash-command entry with name, description, aliases, and handler.
- `Module`: embeddable screen with init/update/view/size/bindings/root-state methods.
- `Runner`: future assistant execution interface; first implementation can stream mock output.
- `Overlay`: modal/autocomplete/select UI stack owned by the shell.
- `LogView` / `OutputView`: scrollable output surface for tool calls, logs, and module events.
- `Panel` / `StateView`: reusable module surface with empty/loading/error/success states.

## Interaction Model

- Normal text submits a user message and invokes the configured `Runner`.
- `/` opens command selection. Commands can append messages, run shell actions, or enter modules.
- `@` opens reference/file autocomplete. First implementation should expose the extension point and, if small enough, local file selection.
- `#` remains resource selection. First implementation may keep file/resource picker minimal.
- `!` is reserved for shell commands and should be disabled or explicit until safety and permissions are designed.
- `ESC` closes the top overlay first, then exits the active module if the module is at root, then returns focus to the prompt.
- Modules render in the main area and keep the shared prompt/status/footer. They should not own global command routing.

## Data Flow

1. User types in the prompt.
2. Shell detects mode by prefix and updates overlay/autocomplete state.
3. On submit, shell either dispatches a command/module action or appends a user message and invokes `Runner`.
4. Runner emits message chunks/results back to the shell.
5. Module actions use shell-owned modal/action result messages where practical, avoiding per-module bespoke pending state.

## UI Style

- Main body uses one of two modes: timeline or active module.
- Inline overlays appear between body and status/prompt, not as full-screen command palettes.
- Prompt remains the primary control surface and must always show available interaction hints.
- Status bar shows current mode/module and concise shortcuts.
- Modal dialogs are shell-owned and default Enter to the action button, Esc to cancel.
- Log/output views should be scrollable, copy-friendly, and not require mouse capture.

## dtui Test Host

`cli/dtui` should register a test module with the new shell:

- `/test` opens a module that exercises shell/module behavior.
- The test module must not require Docker daemon state.
- It should test modal confirm/cancel, file selection callbacks, log/output scrolling, status changes, and ESC/back behavior.

Docker-specific modules will be rewritten later after `uix` is accepted.

## Compatibility

No compatibility adapter is required. Existing imports and plugin constructors may be changed. The goal is a cleaner `uix` contract validated by the `dtui` test module.

## Risk And Rollback

- Risk: touching `uix` and all `dtui` modules can produce broad compile failures. Mitigation: stop migrating Docker modules in this slice and keep only the `dtui` test host.
- Risk: Bubble Tea key routing regressions. Mitigation: keep prompt/overlay/module routing explicit and validate with focused tests where practical.
- Risk: overbuilding an agent framework. Mitigation: mock runner only, no real provider integration in this slice.

## Validation Strategy

- Unit tests for pure registry/message/helper behavior where easy.
- `go test ./cli/uix/...` after shell changes.
- `go test ./cli/dtui/...` after the test host is wired.
- `go build ./cli/dtui` as final compile validation.
