# uix production framework foundation

## Goal

Make `cli/uix/` a production-ready reusable terminal UI framework for `dtui` and future TUI apps. The framework must provide stable shell routing, prompt modes, overlays, module lifecycle, command registry, shared components, safe sizing, status/help visibility, and tests without importing Docker or app-specific code.

## Confirmed Facts

- `uix` already exposes `Module`, `Command`, `Runner`, `Shell`, timeline roles, shell messages, and reusable components.
- `uix` owns global `/`, `@`, `#`, and disabled `!` routing by spec.
- `StatusBar` currently renders a single right-side help string and can overflow/truncate poorly in narrow terminals.
- `HelpText` is set when entering a module, but active module state changes can change `Bindings()` without automatically refreshing shell help.
- `components.Help` exists but is not integrated as a shell-level help surface.
- Tests exist for prompt submission, disabled shell prefix, file picker activation, module command registration, modal escape, and component wrappers.

## Requirements

- This child must load the parent task artifacts before implementation and preserve cross-child requirements such as no Docker coupling, low concurrency, and final production completeness.
- Keep `uix` generic and independent from Docker/`dtui` business packages.
- Preserve current exported contracts unless a planned change updates all callers and tests.
- Make active module commands visible and resilient to long help text or narrow terminal widths.
- Ensure prompt/overlay/key routing remains deterministic with active modules and nested module modes.
- Ensure status, dropdown, modal, file picker, timeline, and components handle width/height <= 0 and narrow layouts safely.
- Ensure framework examples and tests demonstrate production usage, not only a mock shell.

## Acceptance Criteria

- [ ] `go test ./cli/uix/...` passes.
- [ ] `go vet ./cli/uix/...` passes.
- [ ] Active module help is visible without overflowing the terminal width.
- [ ] Module state changes that affect `Bindings()` have a clear way to refresh help/status.
- [ ] `/`, `@`, `#`, disabled `!`, `esc`, modal, and file picker behavior remains covered by tests.
- [ ] Components used by production modules have safe default sizing and focused inputs where required.
- [ ] `cli/uix/_example` remains buildable and demonstrates module registration cleanly.

## Out Of Scope

- Docker module behavior.
- Arbitrary shell execution for `!`.
- Dynamic plugin loading.
- LLM provider integration beyond `Runner`.

## Open Questions

- None blocking this child. `uix` must provide reusable modal/confirmation surfaces but must not encode Docker-specific safety policy.

## Notes

- This child must complete before production host/module work that depends on shared help/status/layout behavior.
- Start this child first despite the current breadcrumb possibly pointing at the docs child.
