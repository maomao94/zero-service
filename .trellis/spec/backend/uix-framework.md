# uix — Chat-like CLI/TUI Shell Code-Spec

> Canonical executable contract for `cli/uix/` and TUI modules hosted by it. Last updated: 2026-06-12 (dtui module rewrite + review fixes).

> **⚠️ 实验性代码**：`cli/uix/`、`cli/dtui/` 及其所有子模块由 AI 自动生成，未经人工审查，存在状态机边界问题、测试盲区和架构缺陷。**生产环境不可用**，仅供实验和参考。正式使用前必须人工逐文件 review 并重写关键逻辑。

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

### D7: Lazy client initialization

Business modules that need Docker (or other external services) must use lazy initialization via `ensureClient()` pattern. The client is created on first use, not at module construction or `Init()`. This allows `dtui` to start without Docker daemon running.

```go
type Module struct {
    client *client.Client
    // ...
}

func (m *Module) ensureClient() error {
    if m.client != nil {
        return nil
    }
    c, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return fmt.Errorf("docker not available: %w", err)
    }
    m.client = c
    return nil
}
```

### D8: Form mode with IsRoot() toggle

Modules with inline forms (text input, multi-field editing) should toggle `IsRoot()` based on form state. When form is active, `IsRoot()` returns `false` so ESC exits the form first before exiting the module.

```go
func (m *Module) IsRoot() bool {
    return !m.formMode
}
```

### D9: Automatic status/help refresh on module state changes

When a module's `Bindings()` can change dynamically (e.g., after selecting a resource, entering a subview, or completing an action), the shell automatically refreshes the status bar after every `routeToActive()` call and after non-root `esc` handling. Modules do not need to manually emit `StatusMsg` to update help text — the shell reads `Bindings()` and recomposes the status bar (module help + global shell help) automatically.

The shell calls `refreshActiveStatus()` in three places:
1. After `routeToActive()` returns an updated `Module` model
2. After `handleEscape()` forwards `esc` to a non-root module
3. After `EnterModule()` to set initial module help

```go
// Shell automatically refreshes status after module update
func (app *Shell) routeToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
    model, cmd := app.active.Update(msg)
    if module, ok := model.(Module); ok {
        app.active = module
        app.refreshActiveStatus() // reads active.Bindings()
    }
    return app, cmd
}
```

Modules only need to return updated `Bindings()` from their `Update()` method; the shell handles the rest. For explicit left/right status text overrides (e.g., showing loading state), modules should still use `StatusMsg`.

### D10: Dynamic Bindings() per subview mode

Modules with multiple subviews (detail, log, stats, history, forms) must return different `Bindings()` for each mode. The status bar automatically reflects the current mode's bindings via D9.

```go
func (m *Module) Bindings() []uix.HelpBinding {
    if m.detailMode {
        return []uix.HelpBinding{
            {Keys: []string{"↑↓"}, Desc: "滚动"},
            {Keys: []string{"esc"}, Desc: "返回"},
        }
    }
    if m.statsMode {
        return []uix.HelpBinding{
            {Keys: []string{"esc"}, Desc: "返回"},
        }
    }
    // Root table mode
    return []uix.HelpBinding{
        {Keys: []string{"↑↓"}, Desc: "选择"},
        {Keys: []string{"i"}, Desc: "详情"},
        {Keys: []string{"t"}, Desc: "统计"},
    }
}
```

Combined with D8 (`IsRoot()` toggling), this gives users clear, context-aware help text without manual `StatusMsg` management. Subview modes should use non-root `IsRoot()` so `esc` closes the subview before exiting the module.

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
| Module migrated but not registered | Module unreachable from shell | Wire in `main.go` with `app.RegisterModule()` |
| Config delete misses sections | Partial deletion, data inconsistency | Handle ALL sections in delete/confirm handlers |
| Module manually sets help on every update | Stale help after state changes; code duplication | Return updated `Bindings()` from `Update()`; shell refreshes status automatically (D9) |
| Deploy without backup | Data loss on failed deploy | Always backup via `CopyFromContainer` before `CopyToContainer` |
| Deploy without history | No audit trail for failures | Record success/failure via `config.RecordHistory` after every deploy |
| Destructive action without confirmation | Accidental data loss | Show `ShowModalMsg` with target identity; require `enter` to confirm |
| Zip extraction without path validation | Path traversal attack | Validate extracted paths stay within destination directory |

## Lessons Learned (2026-06-12)

### Config section handling

When a config module manages multiple sections (e.g., ComposeDirs + DeployTargets + DeployPackages), every operation (delete, add, edit) must handle ALL sections. The `currentEntry()` function must return correct section-aware indices, and `handleConfirm` must dispatch to the correct config method based on section name.

### Test module as validation host

`/test` module in `dtui` validates the entire `uix` shell contract without Docker: modal, file picker, log viewer, state views, chart demo, and module back behavior. This is the canonical pattern for validating `uix` before business module migration.

### ntcharts integration

- Use v1 path `github.com/NimbleMarkets/ntcharts` (not v2 which uses `charm.land/bubbletea/v2`)
- Chart components must call `Draw()` after `SetData()`, `SetSize()`, or style changes
- Sparkline and BarChart satisfy `ChartComponent` interface for polymorphic usage

### Deploy backup-before-copy pattern

Deploy operations must backup current container content before overwriting. Use `CopyFromContainer` to extract the target path to a timestamped backup directory, then proceed with `CopyToContainer`. If backup fails, abort and record failure in history. Old backups auto-clean via `CleanOldBackups(dir, keep)`.

```go
// Backup → Extract → Copy → Record → Cleanup
backupPath := filepath.Join(target.BackupDir, time.Now().Format("20060102-150405"))
if err := client.CopyFromContainer(target.Container, target.HtmlPath, backupPath); err != nil {
    config.RecordHistory(historyPath, config.HistoryEntry{Success: false, Error: err.Error()})
    return deployResultMsg{err: fmt.Errorf("backup failed: %w", err)}
}
// ... copy to container ...
config.RecordHistory(historyPath, config.HistoryEntry{Success: true})
config.CleanOldBackups(target.BackupDir, 5)
```

### Deploy history recording

All deploy actions (success and failure) must be recorded via `config.RecordHistory`. History entries include: time, action, target name, detail (source path), success flag, and error message. History is capped at 200 entries. The deploy module exposes a history view (`h` key, non-root) showing the last 20 entries with status icons.

### Second confirmation for destructive operations

All destructive operations across modules use the same pattern:
1. User presses action key (e.g., `x` for delete, `p` for prune, `d` for compose down)
2. Module sends `uix.ShowModalMsg` with target identity and impact description
3. User confirms via `enter` or cancels via `esc`
4. Module receives `uix.ConfirmMsg` and dispatches accordingly

This applies to: container remove, image remove, image prune, compose down, deploy overwrite, backup cleanup, and rollback.

### PathType classification for deploy safety

Deploy source paths must be classified before processing:
- `"folder"` — directory, copy directly
- `"zip"` — extract to temp dir first, then copy
- `"unknown"` — file, treat as single-file deploy
- `"invalid"` — path doesn't exist, abort with error

```go
func PathType(path string) string {
    info, err := os.Stat(path)
    if err != nil { return "invalid" }
    if info.IsDir() { return "folder" }
    if strings.HasSuffix(strings.ToLower(path), ".zip") { return "zip" }
    return "unknown"
}
```
