# Bubble Tea Ecosystem: Libraries & Patterns

Research sources: Official Bubble Tea examples, bubbles library, community projects (shelfctl, Loki benchmark, chat-tails, bubbletea-commandpalette, ntcharts).

---

## Architecture Patterns

### Multi-View via State Enum (Official Pattern)

From `examples/composable-views` — the recommended way to switch between views:

```go
type ViewType int
type model struct {
    currentView ViewType
    homeModel   tea.Model
    settingsModel tea.Model
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.currentView {
    case Home: return delegate(m.homeModel, msg)
    case Settings: return delegate(m.settingsModel, msg)
    }
}
```

### Persistent Header/Body/Footer Layout

From Bubble Tea chat example + Loki benchmark:

```go
headerHeight := lipgloss.Height(header)
footerHeight := lipgloss.Height(footer)
bodyHeight := height - headerHeight - footerHeight

return lipgloss.JoinVertical(lipgloss.Left,
    header,
    lipgloss.NewStyle().Height(bodyHeight).MaxHeight(bodyHeight).Render(body),
    footer,
)
```

**CRITICAL**: Height/MaxHeight prevents old screen residue.

### WindowSizeMsg Pattern

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    m.viewport.SetWidth(msg.Width)
    m.viewport.SetHeight(msg.Height - inputHeight)
```

Always handle Width=0 with default 80.

### Key Bindings with bubbles/key + bubbles/help

```go
type KeyMap struct {
    Up   key.Binding
    Quit key.Binding
}
func (k KeyMap) ShortHelp() []key.Binding { return []key.Binding{k.Up, k.Quit} }

// View: m.help.View(m.keys)
// Update: key.Matches(msg, m.keys.Quit) -> tea.Quit
```

### Async Operations via tea.Cmd

```go
func loadData() tea.Cmd {
    return func() tea.Msg { data, err := fetch(); return DataLoadedMsg{data, err} }
}
```

---

## Bubbles Components

| Component | Key API |
|-----------|---------|
| textinput | Focus(), Blur(), SetValue(), Value(), Reset(), SetWidth(), SetSuggestions() |
| viewport | SetContent(), GotoTop(), GotoBottom(), SetWidth(), SetHeight() |
| table | SetColumns(), SetRows(), SelectedRow(), SetWidth(), SetHeight() |
| help | View(keyMap), ShowAll toggle |
| spinner | Tick (required), View() |
| key | NewBinding(WithKeys(), WithHelp()) |

---

## lipgloss Layout

| Function | Purpose |
|----------|---------|
| JoinVertical(Left, a, b, c) | Stack vertically |
| JoinHorizontal(Top, a, b) | Side-by-side |
| Place(w, h, Center, Center, content) | Center overlay (modals/palettes) |
| NewStyle().Width(n).MaxWidth(n) | Column sizing |
| NewStyle().Height(n).MaxHeight(n) | Prevent bleed-through |
| NewStyle().Border(RoundedBorder()) | Bordered panels |

---

## Command Palette Pattern

From bubbletea-commandpalette:
- Fuzzy search on Label + Keywords (hidden search terms)
- Live filtering on every keystroke
- Results capped at N (no scroll offset — narrow query further)
- ESC not handled by component → parent intercepts
- Run() not called by component → parent handles execution
- `lipgloss.Place` for centered overlay rendering

---

## Harmonica: Spring Animation

```go
import "github.com/charmbracelet/harmonica"

spring := harmonica.NewSpring(harmonica.FPS(60), 6.0, 0.5)

// Per frame (in tick handler):
x, xVel = spring.Update(x, xVel, targetX)
```

**Use in uix**: Smooth modal open/close transitions, panel slide-in animations.
**Status for v1**: Optional enhancement. Defer to post-v1.

---

## BubbleZone: Mouse Event Tracking

```go
import zone "github.com/lrstanley/bubblezone"

// In View(): wrap clickable areas
zone.Mark("btn-ok", okButtonStyle.Render("OK"))

// In Update():
case tea.MouseMsg:
    if zone.Get("btn-ok").InBounds(msg) { /* clicked */ }

// Root View() must wrap with zone.Scan():
return zone.Scan(allViews)

// Program must enable mouse:
tea.WithMouseCellMotion()
```

**Use in uix**: Clickable command palette items, modal buttons, table rows.
**Status for v1**: Optional. Mouse is out of scope for v1 PRD.

---

## ntcharts: Terminal Charts

Chart types: linechart, barchart, timeserieslinechart, canvas (braille/block runes).

```go
chart := tslc.New(80, 20, zoneManager)
chart.PushAll(dataPoints)
chart.DrawBrailleAll()
```

**Use in dtui**: Stats panel CPU/MEM real-time charts.
**Status for v1**: Defer. Stats panel can use text-based bars initially.

---

## DO NOT Use (Anti-patterns from old code)

- ❌ LazyGit-style Context interface with polymorphic stack — too complex
- ❌ PanelManager with Open/Close/HandleMsg/HandleKey interface — over-engineered
- ❌ Separate layout.Metrics package — calculation is simple enough
- ❌ Magic numbers (headerH=6, footerH=1) — use lipgloss.Height()
- ❌ antsx.EventEmitter for component communication — v1 uses tea.Cmd |
