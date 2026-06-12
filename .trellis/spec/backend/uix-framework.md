# uix — Chat-like CLI/TUI Shell Code-Spec

> Canonical executable contract for `cli/uix/` and TUI modules hosted by it. Last updated: 2026-06-12 (component enrichment).

## 1. Scope / Trigger

Read this spec before changing any of:

- `cli/uix/**`
- `cli/dtui/**` when it is used as a `uix` host or module package
- Bubble Tea modules that implement `uix.Module`

`uix` is a chat-like glue shell. It owns the prompt, command palette, timeline, modal/file overlays, shared status bar, module entry/exit, and future runner integration. Business CLIs such as `dtui` are hosts or modules; they must not own global prompt routing.

## 2. Signatures

### Shell construction

```go
app := uix.NewShell("dtui > ")
app.RegisterModule(test.New())
app.RegisterCommand(uix.Command{
    Name:        "doctor",
    Description: "Run local checks",
    Run: func(app *uix.Shell) tea.Cmd {
        app.AppendMessage(uix.RoleSystem, "doctor started")
        return nil
    },
})
app.SetRunner(customRunner)
return app.Run()
```

`NewApp(prompt string)` is only a constructor alias for `NewShell(prompt string)`.

### Module contract

```go
type Module interface {
    Name() string
    Description() string
    Aliases() []string
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetSize(width, height int)
    Bindings() []HelpBinding
    IsRoot() bool
}
```

`RegisterModule(module)` registers both the module and a slash command using `Name()` plus `Aliases()`.

### Command contract

```go
type Command struct {
    Name        string
    Description string
    Aliases     []string
    Run         func(*Shell) tea.Cmd
}
```

Commands with empty `Name` or nil `Run` are ignored by the registry.

### Runner contract

```go
type Runner interface {
    Run(input string, history []Message) tea.Cmd
}
```

The default is `MockRunner`. It emits a local `RoleTool` message and a `RoleAssistant` message; it must not require provider credentials.

### Shell messages

```go
type ShowModalMsg struct {
    Title   string
    Message string
    Buttons []components.ModalButton
}

type ConfirmMsg struct { Button string }
type FileSelectedMsg struct { Path string }

type AppendMessageMsg struct {
    Role    MessageRole
    Content string
}

type StatusMsg struct {
    Left  string
    Right string
}
```

Modules send `ShowModalMsg`, `AppendMessageMsg`, and `StatusMsg` to the shell. The shell sends `ConfirmMsg` and `FileSelectedMsg` to the active module when applicable.

### Timeline messages

```go
const (
    RoleUser      MessageRole = "user"
    RoleAssistant MessageRole = "assistant"
    RoleSystem    MessageRole = "system"
    RoleTool      MessageRole = "tool"
    RoleModule    MessageRole = "module"
)
```

## 3. Contracts

### Layout contract

```text
┌──────────────────────────────┐
│ Timeline or active Module    │
├──────────────────────────────┤
│ Dropdown / FilePicker        │  inline overlay area
├──────────────────────────────┤
│ StatusBar                    │  current mode + help
├──────────────────────────────┤
│ CmdBar prompt                │  always focused
└──────────────────────────────┘
```

Modal is the only full-screen overlay. Command palette and file picker stay inline above the status/prompt area.

### Prompt mode contract

| Prefix | Mode | Required behavior |
| --- | --- | --- |
| none | chat | `enter` appends `RoleUser` and calls `Runner.Run` |
| `/` | command | Show command palette; `enter` runs selection or typed command |
| `@` | reference | Show reference placeholders; no file IO required yet |
| `#` | resource | Open file picker; call `filepicker.Init()` on activation |
| `!` | shell | Disabled by default; append system message instead of executing |

When a module is active, normal keys go to the module. Only `/`, `@`, and `#` are routed to the shell prompt.

### Module root contract

- `IsRoot() == true` + `esc`: shell exits the module and appends a `RoleModule` timeline event.
- `IsRoot() == false` + `esc`: shell forwards `esc` to the module first.
- Modules should use non-root mode for subviews such as log viewers or forms.

### Component contract

#### Core shell components

- `components.CmdBar`: wraps focused `textinput`; exposes `Prefix()`, `Query()`, `Value()`, `SetValue()`, `SetWidth()`.
- `components.Dropdown`: command/reference palette; supports filtering, cursor movement, selected entry, safe width.
- `components.FilePicker`: wraps `bubbles/filepicker`; initializes directory loading through `Init()`.
- `components.Modal`: defaults active button to the first button with `Key == "enter"`; `esc` cancels via shell behavior.
- `components.LogViewer`: scrollable output/log viewport; supports follow mode, line append, full replacement, paging.
- `components.StateView` / `components.Panel`: reusable empty/loading/success/warning/error and module panel surfaces.
- `theme.Truncate`: required for visible text truncation; do not byte-slice user-visible strings.

#### Bubbles wrapper components

- `components.Spinner`: wraps `bubbles/spinner`; `NewSpinner()`, `Start() tea.Cmd`, `Stop()`, `Update(tea.Msg) tea.Cmd`, `View() string`. Supports custom spinner style and theme colors.
- `components.Progress`: wraps `bubbles/progress`; `NewProgress(width int)`, `SetPercent(float64)`, `Update(tea.Msg) tea.Cmd`, `View() string`. Clamps percent to [0, 1].
- `components.TextArea`: wraps `bubbles/textarea`; `NewTextArea(width, height int)`, `Focus()`, `Blur()`, `Value() string`, `SetValue(string)`, `Update(tea.Msg)`, `View() string`. Supports placeholder, char limit, line count display.
- `components.DataTable`: wraps `bubbles/table`; `NewTable(columns []Column, rows []Row, width int)`, `SetRows()`, `SetColumns()`, `Update(tea.Msg)`, `View() string`. Must initialize columns before rows.
- `components.List`: wraps `bubbles/list`; `NewList(items []Item, width, height int)`, `SetFilteringEnabled(bool)`, `Update(tea.Msg)`, `View() string`. Supports custom item rendering.
- `components.Help`: wraps `bubbles/help`; `NewHelp()`, `AddBinding(Binding)`, `View() string`. Supports short/full help toggle.

#### ntcharts chart components

- `components.Sparkline`: wraps `ntcharts/sparkline`; `NewSparkline(width, height int)`, `SetData([]float64)`, `AddData(...float64)`, `SetSize(width, height int)`, `View() string`. Must call `Draw()` after data/size changes.
- `components.BarChart`: wraps `ntcharts/barchart`; `NewBarChart(width, height int, labels []string, values []float64)`, `SetData(labels []string, values []float64)`, `SetSize(width, height int)`, `View() string`. Must call `Draw()` after data/size changes.
- `components.ChartComponent`: interface with `SetSize(width, height int)` and `View() string`; satisfied by Sparkline and BarChart.

## 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Shell receives width/height <= 0 | Use 80x24 or component safe defaults |
| CmdBar constructed | `textinput.Focus()` must be called immediately |
| `#` opens file picker | Construct `FilePicker`, store it in shell, return `Init()` command |
| Empty prompt submitted | No-op; no timeline message |
| Unknown slash command | Append `RoleSystem` message; do not panic |
| `!anything` submitted | Append disabled-shell system message; never execute |
| Modal active | Modal handles `enter`, `esc`, `left/right`, `h/l`; active module does not receive keys |
| Modal confirm | Shell emits `ConfirmMsg{Button: label}` to active module |
| File selected with active module | Shell emits `FileSelectedMsg{Path: path}` to active module |
| File selected without active module | Shell appends `RoleTool` selected-file message |
| Active module root + `esc` | Clear prompt/dropdown, exit module, restore chat status |
| Active module non-root + `esc` | Forward key to module; module decides subview close behavior |
| Module returns non-`Module` model | Keep previous active module |
| `AppendMessageMsg` with empty role | Default to `RoleModule` |
| `StatusMsg` empty side | Do not overwrite that status side |

## 5. Good/Base/Bad Cases

- Good: `dtui` registers only `test.New()` while validating `uix`; no Docker daemon is required to start the host.
- Good: A module opens a modal by returning `ShowModalMsg` and handles `ConfirmMsg` in `Update`.
- Good: A module enters log view with `IsRoot() == false`; `esc` exits log view before shell exits the module.
- Good: Normal prompt text appends `RoleUser`, then `MockRunner` appends tool and assistant messages.
- Base: `@` displays placeholder reference entries until real reference providers exist.
- Base: `#` selects a local file and forwards the path to the active module.
- Bad: A module calls Docker or any external service during host startup.
- Bad: A module creates its own global command palette or consumes `/` before shell routing.
- Bad: A command executes `!` shell input without a permission/output-streaming design.
- Bad: Visible text truncates with `s[:n]` instead of `theme.Truncate`.

## 6. Tests Required

Run these after any `uix` shell/module contract change:

```bash
go test ./cli/uix/... ./cli/dtui/...
go build -o /var/folders/0g/rl3htjrs1jdd9p9jb0sfz9fc0000gn/T/opencode/dtui-check ./cli/dtui
go vet ./cli/uix/... ./cli/dtui/...
git diff --check
```

Focused assertion points:

- `CmdBar.Prefix()` detects `/`, `@`, `#`, `!`, and free text.
- `CmdBar.Query()` strips the prefix and one leading space.
- `theme.Truncate()` preserves valid UTF-8 and visual width.
- Shell slash commands can enter a registered module by name and alias.
- `!` prompt input does not execute shell commands.
- `ShowModalMsg` defaults Enter to the action button.
- `dtui` host builds without Docker daemon setup.

## 7. Wrong vs Correct

### Wrong: host startup depends on business infrastructure

```go
client, err := docker.NewClient()
if err != nil { os.Exit(1) }
app.RegisterModule(containers.New(client))
```

This makes `uix` skeleton validation depend on Docker and hides shell regressions behind daemon failures.

### Correct: host startup validates shell through a test module

```go
app := uix.NewShell("dtui > ")
app.RegisterModule(test.New())
app.AppendMessage(uix.RoleSystem, "DTUI test host ready. Type /test to exercise the uix shell; Docker is not required.")
return app.Run()
```

### Wrong: module owns global slash/file routing

```go
case "/":
    return module.openPalette()
case "#":
    return module.openFilePicker()
```

This competes with shell prompt modes and breaks consistent interaction.

### Correct: shell owns global prefixes; module handles local keys

```go
func (m *Module) Bindings() []uix.HelpBinding {
    return []uix.HelpBinding{{Keys: []string{"m"}, Desc: "modal"}}
}

func (m *Module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case uix.FileSelectedMsg:
        m.selectedFile = msg.Path
    case uix.ConfirmMsg:
        return m.handleConfirm(msg.Button)
    case tea.KeyMsg:
        return m.handleLocalKey(msg)
    }
    return m, nil
}
```

## Design Decisions

### D1: Chat-like shell before business modules

`uix` must be validated independently before Docker or other business modules are rewritten. `dtui` is currently a test host, not the canonical Docker UI.

### D2: Prompt prefixes are shell-level

`/`, `@`, `#`, and `!` are global shell concepts. Modules may document them in help text but must not reimplement the global routing.

### D3: Disabled shell execution by default

Shell command execution needs separate permission, streaming, cancellation, and audit contracts. Until then, `!` is only a visible reserved prefix.

### D4: Inline overlays over full-screen palettes

Dropdown and file picker render inline to preserve context. Full-screen modal is reserved for explicit confirmations.

### D5: No mouse capture by default

Do not enable `tea.WithMouseCellMotion()` unless a task requires mouse interaction. Native terminal selection must keep working.

### D6: Component enrichment strategy

Enrich `uix` with Bubbles wrappers first (no new dependencies), then ntcharts charts (MIT, v1-compatible). All components follow `SetSize(width, height int)` contract with safe defaults (80x24 minimum). Charts must call `Draw()` after data/size changes. No mouse capture by default.

## Bubble Tea Gotchas

- `textinput.New()` is unfocused; call `Focus()` in prompt constructors.
- `filepicker.Init()` is required to load directories.
- `bubbles/table` columns must exist before `SetRows()`.
- `SetSize()` must guard width/height <= 0 and propagate safe dimensions to children.
- Use `theme.Truncate` for user-visible truncation to avoid corrupting multibyte text.
- ntcharts components must call `Draw()` after `SetData()`, `SetSize()`, or style changes.
- ntcharts v2 uses `charm.land/bubbletea/v2` (incompatible); use v1 path `github.com/NimbleMarkets/ntcharts`.

## Anti-patterns

| Pattern | Problem | Solution |
| --- | --- | --- |
| Business daemon required at host startup | Blocks shell validation | Use test module; initialize business clients inside business modules later |
| Module owns `/`, `@`, `#`, `!` | Conflicts with Shell | Shell routes prefixes; module receives local keys |
| Full-screen command palette | Loses context | Inline Dropdown above status/prompt |
| Byte slicing visible strings | Breaks UTF-8 and width | `theme.Truncate` |
| Real provider logic in `uix` core | Couples shell to LLM runtime | Use `Runner` interface |
