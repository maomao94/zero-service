# uix — CLI/TUI Framework Conventions

> Canonical source for `cli/uix/` and `cli/dtui/` development. Last updated: 2026-06-11.

## Architecture

### FrameworkApp (`cli/uix/app.go`)

Top-level `tea.Model`. Manages layout, plugin lifecycle, modal, and prefix-based input modes.

```go
app := uix.NewApp("dtui > ")
app.Register(containerPlugin)
app.Register(deployPlugin)
app.SetHome(func() string { return homeScreen.View() })
app.Run()
```

**Layout** (bottom CmdBar, opencode-style):

```
┌──────────────────────────────┐
│  Body (plugin.View / home)   │
├──────────────────────────────┤
│  Dropdown / FilePicker       │  ← appears when / or # typed
├──────────────────────────────┤
│  plugin  ↑↓选择 | s启停 | .. │  ← StatusBar
├──────────────────────────────┤
│  dtui > │ /containers        │  ← CmdBar (always focused)
└──────────────────────────────┘
```

Z-order: modal > filepicker > dropdown > normal layout.
No full-screen overlays for command/file selection.

### Input Modes

The CmdBar is a single textinput that switches behavior by prefix:

| Prefix | Mode | Behavior |
|--------|------|----------|
| (none) | `ModeFree` | Free text input (future LLM use) |
| `/` | `ModeCommand` | Dropdown shows registered plugins, filterable |
| `#` | `ModeFile` | bubbles/filepicker opens for directory browsing + file selection |

The `syncMode()` method detects the prefix and activates the appropriate UI. When switching to `#`, it calls `filepicker.Init()` (CRITICAL — without this, the filepicker never loads).

### Key Routing

```
handleKey
  ├─ modal active?     → handleModalKey
  ├─ filepicker active? → handleFilepickerKey
  ├─ dropdown active?   → handleDropdownKey
  ├─ esc?              → handleEscape (parent-child nesting via IsRoot())
  ├─ plugin active?    → routePluginKey (plugin gets key; only / # also → CmdBar)
  └─ home              → routeCmdBarKey (all keys → CmdBar)
```

When a plugin is active, regular keys (s, r, l, etc.) go ONLY to the plugin. Only `/` and `#` are additionally forwarded to the CmdBar for mode switching. This prevents the CmdBar textinput from eating plugin key bindings.

## Plugin Interface

```go
type Plugin interface {
    Name() string
    Description() string
    Aliases() []string
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetSize(width, height int)
    Bindings() []HelpBinding
    IsRoot() bool  // CRITICAL for ESC parent-child nesting
}
```

### IsRoot() Contract

`IsRoot()` returns `true` when the plugin is at its top-level view (no sub-view open). The framework uses this for ESC handling:

- `IsRoot() == true` + ESC → deactivate plugin, go home
- `IsRoot() == false` + ESC → forward ESC to plugin to close sub-view (log viewer, form, etc.)

Implementation examples:

| Plugin | IsRoot() returns |
|--------|-----------------|
| containers | `!showLog` |
| deploy | `step == stepSelect` |
| settings | `formStep == formNone` |
| images, compose | always `true` |

### Bindings Display

`Bindings()` are shown in the StatusBar when the plugin is activated:

```go
func (p *Plugin) Bindings() []uix.HelpBinding {
    return []uix.HelpBinding{
        {Keys: []string{"↑↓"}, Desc: "选择"},
        {Keys: []string{"s"}, Desc: "启停"},
        {Keys: []string{"l"}, Desc: "日志"},
        {Keys: []string{"/"}, Desc: "指令"},
    }
}
```

Output: `↑↓ 选择 | s 启停 | l 日志 | / 指令 | esc 返回 | / 指令 | # 文件`

## Messages

| Type | Direction | Purpose |
|------|-----------|---------|
| `ShowModalMsg` | Plugin → Framework | Request modal dialog |
| `ConfirmMsg` | Framework → Plugin | Modal button pressed |
| `FileSelectedMsg` | Framework → Plugin | File selected via `#` mode |

```go
// Plugin sends modal:
return p, func() tea.Msg {
    return uix.ShowModalMsg{
        Title:   "Confirm",
        Message: "Are you sure?",
        Buttons: []components.ModalButton{
            {Label: "Cancel", Key: "esc"},
            {Label: "Execute", Key: "enter"},
        },
    }
}

// Plugin receives file selection:
case uix.FileSelectedMsg:
    p.selectedPath = msg.Path
    return p.showConfirm()
```

## Components (`cli/uix/components/`)

### CmdBar
Wraps `bubbles/textinput` at bottom of screen. CRITICAL: must call `ti.Focus()` in constructor — unfocused textinput silently ignores all keys.

```go
func NewCmdBar(prompt string) CmdBar {
    ti := textinput.New()
    ti.Focus()  // REQUIRED — otherwise no keys are processed
    ti.Placeholder = "输入 / 选择指令 | # 选择文件"
    ...
}
```

Methods: `Prefix() string` (returns "/" or "#" or ""), `Query() string` (returns text after prefix).

### Dropdown
Inline command palette for `/` mode. Replaces old Palette overlay. Renders above CmdBar, does not take full screen.

```go
d := components.NewDropdown(width, maxHeight)
d.SetEntries(entries)
d.Filter(query)  // fuzzy filter by label/description
d.MoveUp() / d.MoveDown()
d.Selected() *DropdownEntry
```

### FilePicker
Wraps `bubbles/filepicker` for `#` mode. Directory browser with navigation.

```go
fp := components.NewFilePicker(width)
fp.Init()  // REQUIRED — triggers async directory read
fp.DidSelectFile(msg) (bool, string)
fp.Height() int  // returns fp.Height + 4 (header + border + hints)
```

Key bindings: `j/k` navigate, `l/enter` enter directory, `h/backspace` go back, `enter` on file selects, `esc` closes. Defaults to `$HOME` directory.

### Modal
Centered confirm dialog. `←→` switch buttons, `Enter` select, `Esc` cancel. Renders as full-screen overlay.

### StatusBar
Top border + left plugin name (accent) + right help text (dim). Shows plugin Bindings() when active.

### LogViewer
Wraps `bubbles/viewport`. Follow mode, scroll (j/k/pgup/pgdown/g/G), loading state.

### WelcomeScreen
Centered home screen. Logo + subtitle + `/` hint. Does NOT list commands (that's the Dropdown's job). Uses `lipgloss.Place(Center, Center)`.

## Design Decisions

### D1: Inline Dropdown over Full-Screen Overlay
Commands appear inline above CmdBar instead of as a full-screen overlay. The body area shrinks to accommodate. This preserves context and feels more natural.

### D2: Single CmdBar with Prefix Modes
One textinput handles all input. `/` triggers commands, `#` triggers files, default is free text. No separate overlays or focus management needed.

### D3: IsRoot() for ESC Nesting
Plugin declares whether it's at root view. Framework uses this for parent-child ESC: sub-view first, then root, then home.

### D4: bubbles/filepicker over Custom FilePicker
Official `bubbles/filepicker` provides directory navigation, file metadata, permissions display — all features we'd need to rebuild. Custom wrapper only adds theme styling and key remapping.

### D5: No tea.WithMouseCellMotion()
Enabling mouse capture disables terminal text selection. Since dropdown/filepicker work fully with keyboard, mouse support is not worth the copy-paste tradeoff.

## Gotchas

### textinput MUST be Focused
`textinput.New()` creates an unfocused input. Without `ti.Focus()`, `Update()` returns immediately without processing any keys. The entire CmdBar appears broken.

### filepicker MUST call Init()
`bubbles/filepicker.Init()` triggers the async directory read. Without it, `View()` shows an empty directory. Call `fp.Init()` when creating the filepicker and batch the returned `tea.Cmd`.

### CmdBar MUST NOT Eat Plugin Keys
When a plugin is active, the CmdBar should only receive `/` and `#` for mode switching. All other keys go to the plugin. Otherwise, pressing `s` in containers plugin would BOTH toggle a container AND add "s" to the CmdBar.

## Validation & Error Matrix

| Condition | Behavior |
|-----------|----------|
| `textinput` not focused | Key events silently dropped; CmdBar appears broken |
| `filepicker.Init()` not called | FilePicker shows empty directory |
| Width ≤ 0 in WindowSizeMsg | Default to 80 |
| Height ≤ 0 | Default to 24 |
| bodyHeight < 1 | Set to 1 |
| Plugin.Update returns non-Plugin Model | Type assertion fails silently, plugin unchanged |
| Dropdown opens with empty registry | Shows no results |
| Dropdown filter matches nothing | Shows "no matching commands" |
| FilePicker empty directory | Shows "Bummer. No Files Found." (bubbles default) |

## Anti-patterns

| Pattern | Problem | Solution |
|---------|---------|----------|
| `textinput.New()` without `Focus()` | CmdBar completely non-functional | Always call `ti.Focus()` in constructor |
| `tea.WithMouseCellMotion()` | Text selection disabled | Remove it; keyboard navigation suffices |
| Full-screen overlay for command palette | Context loss, jarring UX | Inline Dropdown above CmdBar |
| Plugin and CmdBar both receiving same key | State corruption, unexpected behavior | Route keys: plugin first, only /# to CmdBar |
| `filepicker` without `Init()` | Empty file list, appears broken | Call `Init()` and batch returned `tea.Cmd` |
| Welcome screen listing all commands | Duplicates Dropdown, clutters UI | Welcome shows logo + hint only |
