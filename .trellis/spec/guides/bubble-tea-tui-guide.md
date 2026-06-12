# Bubble Tea TUI Development Guide

> **⚠️ 实验性**：本指南中的代码示例和模式由 AI 生成，未经人工验证。实际使用前请对照 Bubble Tea 官方文档确认正确性。

## When to Use This Guide

When working with `cli/dtui/`, `cli/uix/`, or any Bubble Tea + bubbles TUI component.

## Critical: textinput Focus

**Problem**: `textinput.New()` creates an UNFOCUSED input. Its `Update()` method returns immediately without processing any keys. The input appears completely non-functional.

**Rule**: ALWAYS call `ti.Focus()` after creating a textinput.

```go
// CORRECT: Focus the input immediately
ti := textinput.New()
ti.Focus()
ti.Placeholder = "Type here..."

// WRONG: No Focus() — all keys silently dropped
ti := textinput.New()
ti.Placeholder = "Type here..."  // Will never receive input
```

**Symptom**: CmdBar/input visible but ignores all keystrokes. No errors, no warnings — just silent failure.

## Critical: bubbles/filepicker Init

**Problem**: `filepicker.New()` creates the model but does NOT load any files. The async directory read is triggered by `filepicker.Init()`. Without calling it, `View()` shows an empty directory.

**Rule**: ALWAYS call `fp.Init()` when activating a filepicker and batch the returned `tea.Cmd`.

```go
// CORRECT: Call Init and batch the cmd
fp := filepicker.New()
fp.CurrentDirectory = homeDir
cmds = append(cmds, fp.Init())

// WRONG: Filepicker shows "Bummer. No Files Found." forever
fp := filepicker.New()
// Never called Init()
```

## Critical: Mouse Mode vs Text Selection

**Problem**: `tea.WithMouseCellMotion()` enables SGR mouse protocol. This is all-or-nothing — once enabled, the terminal sends ALL mouse events to the app, and native text selection (copy-paste) is disabled.

**Rule**: Do NOT use `tea.WithMouseCellMotion()` unless mouse interaction is essential. For keyboard-navigable TUIs, omit it to preserve terminal text selection.

```go
// CORRECT: Text selection works
tea.NewProgram(model, tea.WithAltScreen())

// WRONG: Text selection disabled — users can't copy container IDs, paths, etc.
tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
```

## Critical: Key Routing with Active Plugin

**Problem**: When both a plugin and the CmdBar textinput receive the same key, the CmdBar "eats" characters meant for the plugin. Pressing `s` to start a container also adds "s" to the CmdBar.

**Rule**: When a plugin is active, route keys to the plugin first. Only forward `/` and `#` to the CmdBar for mode switching.

```go
// CORRECT: Plugin gets key; only / and # go to CmdBar
if app.active != nil {
    model, cmd := app.active.Update(msg)
    if key == "/" || key == "#" {
        cbar, _ := app.cmdbar.Update(msg)
    }
}

// WRONG: Both plugin AND CmdBar get every key
model, _ := app.active.Update(msg)
cbar, _ := app.cmdbar.Update(msg)  // CmdBar eats plugin keys!
```

## Critical: Table Column Initialization

**Problem**: `bubbles/table` panics with "index out of range" if `SetRows()` is called before columns are initialized.

**Root Cause**: `SetRows()` triggers `UpdateViewport()` → `renderRow()`, which accesses `m.columns[col]`. If columns is empty (length 0), this panics.

**Rule**: ALWAYS initialize columns in the table constructor.

```go
// CORRECT: Initialize columns in constructor
func NewMyTable() *MyTable {
    t := &MyTable{width: 80}
    t.model = table.New(
        table.WithColumns(defaultColumns(80)),  // MUST set columns here
        table.WithFocused(true),
        table.WithStyles(tableStyles()),
    )
    return t
}

// WRONG: No columns in constructor - will panic on SetRows()
func NewMyTable() *MyTable {
    t := &MyTable{}
    t.model = table.New(
        table.WithFocused(true),
        table.WithStyles(tableStyles()),
    )
    return t
}
```

## Critical: Window Size Handling

**Problem**: TUI components may receive width/height = 0 during initialization or resize.

**Rule**: Always validate and use default values for width/height.

```go
func (p *MyPage) SetSize(width, height int) {
    if width < 40 {
        width = 80  // Use sensible default
    }
    if height < 10 {
        height = 24  // Use sensible default
    }
    p.width = width
    p.height = height
    // ... update child components
}
```

## Critical: Column Width Calculation

**Problem**: Percentage-based column widths can result in 0 or negative values for small terminals.

**Rule**: Always use `max()` to enforce minimum column widths.

```go
func myColumns(width int) []table.Column {
    if width < 40 {
        width = 80  // Minimum table width
    }
    return []table.Column{
        {Title: "Name", Width: max(10, width*30/100)},  // At least 10 chars
        {Title: "Status", Width: max(8, width*20/100)},  // At least 8 chars
    }
}
```

## Pattern: Stats History with Timestamp

When displaying time-series data (like container stats), always:
1. Add `Timestamp time.Time` to the entry struct
2. Set `Timestamp: time.Now()` when creating entries
3. Display in reverse chronological order (newest first)
4. Format timestamp as `HH:MM:SS` for readability

```go
type StatsEntry struct {
    Timestamp  time.Time
    CPUPercent float64
    // ...
}

func renderHistory(history []StatsEntry) string {
    var b strings.Builder
    show := history
    if len(history) > 10 {
        show = history[len(history)-10:]
    }
    // Reverse order: newest first
    for i := len(show) - 1; i >= 0; i-- {
        entry := show[i]
        b.WriteString(fmt.Sprintf("  %s  CPU %.1f%%\n",
            entry.Timestamp.Format("15:04:05"),
            entry.CPUPercent))
    }
    return b.String()
}
```

## Pattern: Dual Editor Mode (TUI + External)

For configuration editing, support both TUI form editing and external editor:

```go
func (p *SettingsPage) Update(msg tea.Msg) (context.Context, tea.Cmd) {
    switch msg.String() {
    case "e":
        return p.openEditForm(), nil    // TUI form editing
    case "c":
        return p, p.openConfigEditor()  // External editor
    }
}

func (p *SettingsPage) openEditForm() context.Context {
    row, ok := p.table.Selected()
    if !ok {
        p.status = styles.Warn.Render("请选择可编辑的配置项")
        return p
    }
    fields := editFieldsForSection(row.Section, row)  // Pre-fill current values
    return NewFormPanel("编辑", fields, func(values map[string]string) {
        p.updateBySection(row.Section, row.Index, values)
    })
}
```

## Checklist for New Table Components

- [ ] Initialize columns in constructor with `table.WithColumns()`
- [ ] Handle width=0 in constructor with default value (80)
- [ ] Use `max()` for minimum column widths (at least 8-10 chars)
- [ ] Handle height=0 in constructor with default value (20)
- [ ] `SetSize()` validates width/height before passing to child components
- [ ] `SetRows()` checks cursor bounds after updating rows

## Checklist for New Pages

- [ ] `SetSize()` handles width/height = 0 with defaults
- [ ] `SetSize()` propagates size to all child components (table, detail, etc.)
- [ ] Table is created with default columns in constructor
- [ ] Window resize updates all child components
