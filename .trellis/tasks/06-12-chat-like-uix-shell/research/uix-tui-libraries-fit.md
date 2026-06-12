# Research: cli/uix TUI library fit

- **Query**: Active task `.trellis/tasks/06-12-chat-like-uix-shell`: Research how Bubbles, Lip Gloss, Harmonica, BubbleZone, and ntcharts could fit into the current `cli/uix` architecture as fast out-of-box embeddable components, prioritizing a strong `cli/uix` shell before CLI modules.
- **Scope**: mixed
- **Date**: 2026-06-12

## Findings

### Files Found

| File Path | Description |
|---|---|
| `cli/uix/app.go` | Current `Shell` Bubble Tea model, layout, prompt/dropdown/filepicker/modal routing, module entry, runner lifecycle, and `Run()` program options. |
| `cli/uix/plugin.go` | Public `Module`, `HelpBinding`, and `Command` contracts. |
| `cli/uix/registry.go` | Module and command registries with name/alias resolution and search. |
| `cli/uix/timeline.go` | Viewport-backed chat timeline with role rendering and scroll helpers. |
| `cli/uix/runner.go` | `Runner` interface and local `MockRunner`. |
| `cli/uix/components/cmdbar.go` | Focused Bubbles `textinput` prompt with `/`, `@`, `#`, `!` mode detection. |
| `cli/uix/components/dropdown.go` | Custom inline command/reference palette with filtering and keyboard navigation. |
| `cli/uix/components/filepicker.go` | Wrapper around Bubbles `filepicker` with required `Init()` and safe sizing. |
| `cli/uix/components/logviewer.go` | Viewport-backed log/output component with follow mode and scroll controls. |
| `cli/uix/components/modal.go` | Shell-owned modal dialog with keyboard button selection. |
| `cli/uix/components/state.go` | Reusable `StateView` and `Panel` surfaces for modules. |
| `cli/uix/theme/styles.go` | Lip Gloss style helpers, shared borders, palette styles, and UTF-8-safe `Truncate`. |
| `cli/dtui/plugins/test/plugin.go` | Existing Docker-free test module exercising modal, file picker, log viewer, state views, and module back behavior. |
| `go.mod` | Dependency manifest; currently contains Bubble Tea, Bubbles, and Lip Gloss only among the named libraries. |
| `.trellis/spec/backend/uix-framework.md` | Canonical `uix` shell/module/component contract and validation expectations. |
| `.trellis/spec/guides/bubble-tea-tui-guide.md` | Bubble Tea safety rules: textinput focus, filepicker init, mouse capture, routing, resize handling. |

### Code Patterns

#### Existing `cli/uix` component/API surface relevant to each library

**Bubbles**

- Current `uix` already embeds Bubbles components rather than exposing raw Bubbles types as public API: `CmdBar` wraps `textinput`, `FilePicker` wraps `filepicker`, and `Timeline`/`LogViewer` wrap `viewport`.
- `components.NewCmdBar` creates `textinput.New()`, clears the internal prompt, sets placeholder and width, and calls `ti.Focus()` immediately at `cli/uix/components/cmdbar.go:52`; this matches the project guide rule that unfocused textinput ignores keys at `.trellis/spec/guides/bubble-tea-tui-guide.md:7`.
- `components.NewFilePicker` configures Bubbles filepicker defaults and `FilePicker.Init()` delegates to `fp.fp.Init()` at `cli/uix/components/filepicker.go:56`; shell `#` mode returns that init command at `cli/uix/app.go:473`.
- `Timeline` uses `viewport.New(width, height)` at `cli/uix/timeline.go:37`, updates content on append, and exposes scroll helpers at `cli/uix/timeline.go:88`.
- `LogViewer` uses `viewport.New(width, height)` at `cli/uix/components/logviewer.go:25`, appends/replaces lines, caps stored lines to 500 at `cli/uix/components/logviewer.go:51`, and exposes follow/page/top/bottom controls at `cli/uix/components/logviewer.go:116`.
- Public shell routes keys before handing to components: modal first, filepicker, dropdown, escape, active module, prompt at `cli/uix/app.go:183`; active modules only let `/`, `@`, and `#` enter shell prompt routing at `cli/uix/app.go:231`.
- Fast out-of-box Bubbles additions that match current shape: `textarea` for multiline prompt, `spinner` for local loading state, `progress` for long-running module/runner operations, `list` or `table` for richer command/module selectors, `help`/`key` for generated key help. The current surface can host these as wrappers under `components/` without changing `Module`.

**Lip Gloss**

- Lip Gloss is already the shared rendering layer across the shell and components: `Shell.View()` measures footer/prompt/overlay heights with `lipgloss.Height` and composes the layout with `lipgloss.JoinVertical` at `cli/uix/app.go:394`.
- `timeline.renderMessage` uses Lip Gloss borders, foreground colors, width wrapping, and `theme` colors for role-specific message blocks at `cli/uix/timeline.go:106`.
- Theme-level helpers centralize reusable borders and palette styles in `cli/uix/theme/styles.go:33` and `cli/uix/theme/styles.go:44`.
- `theme.Truncate` uses `lipgloss.Width` and rune-aware backtracking at `cli/uix/theme/styles.go:9`; the `uix` spec requires this for visible truncation at `.trellis/spec/backend/uix-framework.md:150`.
- Fast out-of-box fit is styling/layout only; no new dependency or shell contract is needed.

**Harmonica**

- No current `cli/uix` code imports Harmonica or runs a general frame loop; repository-wide search found only `tea.NewProgram(app, tea.WithAltScreen()).Run()` at `cli/uix/app.go:430`, with no `tea.Tick`, `tea.Every`, or `tea.WithFPS` usages.
- Current animation-compatible surfaces are `StatusMsg` at `cli/uix/app.go:31`, `StateView` loading state at `cli/uix/components/state.go:11`, `LogViewer.SetLoading` at `cli/uix/components/logviewer.go:62`, and future runner/module updates through `tea.Cmd`.
- Harmonica fits only behind a concrete component that owns its tick lifecycle, because its spring update must run on each frame and the delta should match the actual frame rate.
- Fast out-of-box fit is indirect through Bubbles `progress` animation support, or a narrow shell loading/progress component; it does not belong in the core `Runner` or `Module` interface.

**BubbleZone**

- Current shell has a `tea.MouseMsg` case at `cli/uix/app.go:159` and `handleMouse` forwards mouse messages only to the file picker when open at `cli/uix/app.go:171`, but `Run()` does not enable mouse mode: it uses `tea.NewProgram(app, tea.WithAltScreen()).Run()` at `cli/uix/app.go:430`.
- Project rules explicitly preserve native text selection: `.trellis/spec/backend/uix-framework.md:284` says no mouse capture by default, and `.trellis/spec/guides/bubble-tea-tui-guide.md:43` explains `tea.WithMouseCellMotion()` is all-or-nothing and disables native terminal selection.
- BubbleZone requires root-level `zone.Scan()` and marked regions. Its docs state `zone.Scan()` should be used only at the outermost/root model, and examples enable alt-screen plus mouse cell motion.
- Fast out-of-box fit is optional click support for inline overlays/modals or chart widgets, but only behind an explicit opt-in run option; it should not be a default component dependency while text-selection friendliness is a requirement.

**ntcharts**

- Current shell already has the right host surface for charts: module screens render in the body through `Module.View()` at `cli/uix/app.go:555`, receive safe dimensions via `Module.SetSize(width, height)` at `cli/uix/app.go:550`, and can sit inside `components.Panel` at `cli/uix/components/state.go:70`.
- The existing Docker-free test module already sizes a log area relative to module dimensions at `cli/dtui/plugins/test/plugin.go:109`; chart components can follow the same `SetSize` pattern.
- ntcharts provides embeddable terminal chart packages such as `sparkline`, `barchart`, `linechart/streamlinechart`, `linechart/timeserieslinechart`, `linechart/wavelinechart`, `heatmap`, and `canvas`; most constructors take explicit width/height and require redraw after data or size changes.
- Fast out-of-box fit is module-local charts for metrics/log summaries, not core shell rendering. A small wrapper can accept width/height/data, call `Draw()` after resize/data changes, and expose `View()`.

#### Minimal safe additions to implement now

1. **Keep Bubbles wrappers as the primary component pattern**: add only missing wrappers that are immediately used by the shell/test host, likely `Prompt` based on `textarea` if multiline input is required, `Spinner`/`Progress` for runner/module loading feedback, and optionally a `List`-backed command palette if the current `Dropdown` becomes insufficient.
2. **Keep Lip Gloss centralized in `theme` and component wrappers**: reuse `theme.Border`, palette styles, `WidthStyle`, and `Truncate`; add new styles only where duplicated rendering appears in new wrappers.
3. **Use Harmonica only through a bounded animated component**: if progress/loading animation is added, own the tick message inside that component and start/stop ticks based on loading state; otherwise rely on Bubbles static states for this slice.
4. **Keep BubbleZone out of default shell startup**: if mouse is required for a specific module/chart, introduce an explicit opt-in run path and root-level scan point; default `Run()` should keep current `tea.WithAltScreen()` only.
5. **Add ntcharts only when a visible test module needs charts**: start with one low-scope wrapper around `sparkline` or `barchart`; keep it module-local or under `components/` with `SetSize`, `SetData`, and `View` only.

#### Libraries present vs missing

| Library | Current status in `go.mod` | Current version/path | Likely imports for current Bubble Tea v1 stack |
|---|---|---|---|
| Bubble Tea | Present | `github.com/charmbracelet/bubbletea v1.3.10` at `go.mod:12` | `tea "github.com/charmbracelet/bubbletea"` |
| Bubbles | Present | `github.com/charmbracelet/bubbles v1.0.0` at `go.mod:11` | Existing: `github.com/charmbracelet/bubbles/textinput`, `/filepicker`, `/viewport`; likely additions: `/textarea`, `/spinner`, `/progress`, `/list`, `/table`, `/help`, `/key` |
| Lip Gloss | Present | `github.com/charmbracelet/lipgloss v1.1.0` at `go.mod:13` | `github.com/charmbracelet/lipgloss`; optional subpackages from docs: `github.com/charmbracelet/lipgloss/table`, `/list`, `/tree` |
| Harmonica | Missing | Latest on pkg.go.dev: `github.com/charmbracelet/harmonica v0.2.0` | `github.com/charmbracelet/harmonica` |
| BubbleZone | Missing | v1 path available: `github.com/lrstanley/bubblezone v1.0.0`; v2 path available: `github.com/lrstanley/bubblezone/v2 v2.0.0` | For current stack, likely `zone "github.com/lrstanley/bubblezone"`; v2 uses `github.com/lrstanley/bubblezone/v2` and depends on `charm.land/bubbletea/v2`/`charm.land/lipgloss/v2`, so it does not match current imports |
| ntcharts | Missing | v1 path available: `github.com/NimbleMarkets/ntcharts v0.5.1`; v2 path available: `github.com/NimbleMarkets/ntcharts/v2 v2.2.0` | For current stack, likely `github.com/NimbleMarkets/ntcharts/sparkline`, `/barchart`, `/canvas`, `/heatmap`, `/linechart/streamlinechart`, `/linechart/timeserieslinechart`, `/linechart/wavelinechart`; v2 imports add `/v2/...` |

#### Risks and integration constraints

- **Mouse support**: `uix` currently handles `tea.MouseMsg` but does not enable mouse capture. Enabling `tea.WithMouseCellMotion()` or BubbleZone-required mouse mode would disable native terminal text selection, directly conflicting with PRD constraints at `.trellis/tasks/06-12-chat-like-uix-shell/prd.md:44` and specs at `.trellis/spec/backend/uix-framework.md:284`.
- **BubbleZone versioning**: BubbleZone v2 uses `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`; current repo uses `github.com/charmbracelet/bubbletea` v1 and `github.com/charmbracelet/lipgloss` v1, so v2 would imply a broader TUI dependency migration. BubbleZone v1 matches current import family better.
- **Animations in Bubble Tea**: Harmonica and animated Bubbles components require a tick loop. Bubble Tea `Tick`/`Every` commands send one message and must be returned again from `Update` to continue; uncontrolled global ticks can redraw continuously even when no animation is visible.
- **Harmonica timing**: `harmonica.NewSpring(harmonica.FPS(60), ...)` assumes the update cadence matches the chosen FPS. If Bubble Tea renderer FPS or tick duration differs, motion can feel wrong or waste redraws.
- **Chart sizing**: ntcharts constructors take explicit width/height; shell/module dimensions can be zero during initialization, and panels/borders consume cells. Chart wrappers must apply safe minimums before constructing/redrawing and call `Draw()` after data/size changes.
- **Chart redraw lifecycle**: ntcharts examples push data then call `Draw()` or `DrawBraille()` before `View()`. Data updates and `SetSize` need a clear redraw point, otherwise views can show stale geometry.
- **Dependency bloat**: Bubbles and Lip Gloss are already present, so wrappers around their subpackages add no new module. Harmonica is small but still new. BubbleZone and ntcharts add new modules; ntcharts also acknowledges BubbleZone for mouse support, and v2 `ntcharts` currently shows `UNKNOWN` license status on pkg.go.dev while v1 shows MIT.
- **Prompt and module routing**: richer Bubbles components must not compete with shell-owned global prefixes. The canonical contract says `/`, `@`, `#`, and `!` are shell-level at `.trellis/spec/backend/uix-framework.md:272`.

#### Proposed test/validation plan

1. **Existing contract regression**: run `go test ./cli/uix/...` after any component additions; keep current tests around module registration, mock runner lifecycle, disabled `!`, filepicker init, and modal cancel behavior.
2. **Host validation**: run `go test ./cli/dtui/...` and `go build -o /var/folders/0g/rl3htjrs1jdd9p9jb0sfz9fc0000gn/T/opencode/dtui-check ./cli/dtui` after any shell/module API changes.
3. **Bubbles wrappers**: add focused tests for default focus, prefix/query behavior, `SetSize(0,0)` fallback, and filepicker `Init()` command being returned on `#` activation.
4. **Animation wrappers**: test tick handling as pure message transitions, including start, one update, stop/no further tick when not loading; avoid timing-sensitive sleeps.
5. **BubbleZone opt-in path**: if added, test default startup remains keyboard-first/no mouse option, and only the opt-in run path wraps root `View()` in `zone.Scan()`.
6. **ntcharts wrappers**: test safe min dimensions, redraw on `SetSize` and `SetData`, non-empty `View()` output for small and normal sizes, and no panic for width/height <= 0.
7. **Static quality checks**: run `go vet ./cli/uix/... ./cli/dtui/...` and `git diff --check` per `.trellis/spec/backend/uix-framework.md:193` after implementation changes.

### External References

- [Bubbles package docs](https://pkg.go.dev/github.com/charmbracelet/bubbles) — confirms available components: spinner, textinput, textarea, table, progress, paginator, viewport, list, filepicker, timer, stopwatch, help, key; current repo uses v1.0.0.
- [Lip Gloss package docs](https://pkg.go.dev/github.com/charmbracelet/lipgloss) — confirms layout/style APIs used by `uix`, including width/height measurement, placement, borders, joins, and optional table/list/tree subpackages; current repo uses v1.1.0.
- [Harmonica package docs](https://pkg.go.dev/github.com/charmbracelet/harmonica) — confirms module path `github.com/charmbracelet/harmonica`, v0.2.0, `FPS`, `NewSpring`, and per-frame `Update` API.
- [BubbleZone v1 package docs](https://pkg.go.dev/github.com/lrstanley/bubblezone) — confirms v1 path `github.com/lrstanley/bubblezone`, root-level `Scan`, `Mark`, `Get`, `NewGlobal`, and mouse-mode expectations.
- [BubbleZone v2 package docs](https://pkg.go.dev/github.com/lrstanley/bubblezone/v2) — confirms v2 path and cautions that v2 switches to `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`, making it a poor fit for the current v1 stack without broader migration.
- [ntcharts v1 package docs](https://pkg.go.dev/github.com/NimbleMarkets/ntcharts) — confirms v1 path, MIT license on pkg.go.dev, chart packages, explicit width/height constructors, Bubble Tea/Lip Gloss orientation, and BubbleZone acknowledgement for mouse support.
- [ntcharts v2 package docs](https://pkg.go.dev/github.com/NimbleMarkets/ntcharts/v2) — confirms v2 path and packages, but pkg.go.dev currently reports license as `UNKNOWN`.
- [Bubble Tea package docs](https://pkg.go.dev/github.com/charmbracelet/bubbletea) — confirms current v1.3.10 path, program options such as `WithAltScreen`, `WithMouseCellMotion`, `WithFPS`, and command utilities `Tick`/`Every`.

### Related Specs

- `.trellis/spec/backend/uix-framework.md` — canonical shell/module/component contract, prompt modes, inline overlay layout, no mouse capture default, and validation commands.
- `.trellis/spec/guides/bubble-tea-tui-guide.md` — Bubble Tea component safety guide covering textinput focus, filepicker init, mouse-mode text selection, key routing, table columns, and safe sizing.
- `.trellis/tasks/06-12-chat-like-uix-shell/prd.md` — task-level requirement to prioritize a chat-like `cli/uix` shell and preserve text-selection friendliness.
- `.trellis/tasks/06-12-chat-like-uix-shell/design.md` — shell architecture with timeline/module body, prompt, inline overlays, status bar, runner, and module hosting.
- `.trellis/tasks/06-12-chat-like-uix-shell/implement.md` — implementation checklist and validation commands for the chat-like shell slice.

## Caveats / Not Found

- Research only; no code or `go.mod` changes were made.
- No local `go test`, `go build`, `go vet`, or `go list` commands were run for this research note.
- `go.mod` does not contain Harmonica, BubbleZone, or ntcharts. It does contain Bubble Tea, Bubbles, and Lip Gloss.
- Exact transitive dependency impact for Harmonica/BubbleZone/ntcharts was not measured with `go get` or `go mod graph` because this task is research-only.
- BubbleZone v2 and ntcharts v2 have available module paths, but both introduce version/licensing constraints that need explicit confirmation before choosing them over v1-family imports.
