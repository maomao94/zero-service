# Technical Design: uix CLI Framework

## Design Principles

1. **Follow Bubble Tea official patterns** — state enum for view switching, model composition, lipgloss.JoinVertical for layout
2. **uix is a pure framework** — zero domain knowledge (no Docker, no container concepts)
3. **Plugin-based extensibility** — modules implement a simple interface, framework handles routing
4. **Command-driven UX** — `/` prefix triggers command palette, fuzzy search for discovery

## Architecture

```
┌─────────────────────────────────────────────────┐
│  FrameworkApp (tea.Model)                       │
│  ┌───────────────────────────────────────────┐  │
│  │  CmdBar (textinput.Model)                 │  │  ← always visible
│  ├───────────────────────────────────────────┤  │
│  │                                           │  │
│  │  Content Area                             │  │  ← current plugin.View()
│  │  (delegated to active Plugin)            │  │
│  │                                           │  │
│  ├───────────────────────────────────────────┤  │
│  │  StatusBar                                │  │  ← always visible
│  └───────────────────────────────────────────┘  │
│  Overlays (z-order):                            │
│    CommandPalette  (shown on `/` input)          │
│    Modal           (shown on confirm/form)       │
└─────────────────────────────────────────────────┘
```

### uix Package Structure

```
cli/uix/
  app.go              # FrameworkApp: top-level tea.Model
  plugin.go           # Plugin interface definition
  registry.go         # PluginRegistry: register, lookup, list commands
  navigation.go       # NavStack: Push/Pop for panels/modals
  components/
    cmdbar.go         # CLI command input bar
    palette.go        # Command palette overlay (fuzzy search)
    modal.go          # Modal overlay (centered, lipgloss.Place)
    statusbar.go      # Status bar
    table.go          # Generic table helper
    help.go           # Key binding help helper
  theme/
    theme.go          # Tokyo Night color tokens
    styles.go         # Common style factories
```

### D1: FrameworkApp (cli/uix/app.go)

The top-level `tea.Model` that all uix applications use.

```go
type FrameworkApp struct {
    width  int
    height int

    // Persistent components
    cmdbar    CmdBar
    palette   Palette
    statusbar StatusBar
    help      help.Model

    // Plugin system
    registry  *PluginRegistry
    active    Plugin              // current active plugin
    stack     *NavStack[Plugin]   // for panels/modals

    // UI state
    showPalette bool
    showModal   bool
}
```

**View layout** (official Bubble Tea pattern with lipgloss.JoinVertical):

```go
func (app FrameworkApp) View() string {
    // z-order: palette > modal > normal
    if app.showPalette {
        return app.renderPaletteOverlay()
    }
    if app.showModal {
        return app.renderModalOverlay()
    }

    // Normal: header(cmdbar) + body(plugin.View()) + footer(statusbar)
    header := app.cmdbar.View()
    body := app.active.View()
    footer := app.statusbar.View()

    return lipgloss.JoinVertical(lipgloss.Left,
        header,
        lipgloss.NewStyle().Height(app.bodyHeight()).MaxHeight(app.bodyHeight()).Render(body),
        footer,
    )
}
```

**Update routing** (official pattern: global keys first, then delegate):

```go
func (app FrameworkApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        return app.handleResize(msg)
    case tea.KeyMsg:
        return app.handleKey(msg)
    }
    // Delegate to active plugin
    return app.delegateToPlugin(msg)
}
```

### D2: Plugin Interface (cli/uix/plugin.go)

Simplified `tea.Model` + registration metadata. This is the contract every module implements.

```go
type Plugin interface {
    // Metadata (for command registration)
    Name() string           // "containers"
    Description() string    // "Manage Docker containers"
    Aliases() []string      // ["c", "cnt"]

    // Lifecycle
    Init(PluginContext) tea.Cmd
    OnActivate() tea.Cmd
    OnDeactivate()

    // Bubble Tea Model
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetSize(width, height int)

    // Key bindings for help bar
    Bindings() []HelpBinding
}

type PluginContext struct {
    Width  int
    Height int
    Bus    interface{}  // future: antsx.EventEmitter for inter-plugin communication
}

type HelpBinding struct {
    Keys []string
    Desc string
}
```

### D3: PluginRegistry (cli/uix/registry.go)

Central registry for discovering and activating plugins.

```go
type PluginRegistry struct {
    plugins map[string]Plugin          // name -> factory
    aliases map[string]string          // alias -> name
    order   []string                   // registration order
}

func (r *PluginRegistry) Register(p Plugin)
func (r *PluginRegistry) Resolve(input string) Plugin       // exact name or alias match
func (r *PluginRegistry) Search(query string) []Plugin      // fuzzy match
func (r *PluginRegistry) List() []Plugin                    // all in registration order
```

### D4: CmdBar (cli/uix/components/cmdbar.go)

A text input that supports `/` command mode.

```go
type CmdBar struct {
    input       textinput.Model
    prompt      string      // "dtui > "
    history     []string
    historyIdx  int
    inCommand   bool        // true when first char is /
}

func (c *CmdBar) Init() tea.Cmd { return textinput.Blink }
func (c *CmdBar) Update(msg tea.Msg) (CmdBar, tea.Cmd)
func (c *CmdBar) View() string
func (c *CmdBar) Value() string
func (c *CmdBar) SetValue(s string)
func (c *CmdBar) Focus() tea.Cmd
func (c *CmdBar) Blur()
```

**Behavior**:
- Normal mode: free text input (for future chat/command features)
- When first char is `/`: switch to command mode, palette opens
- Enter: if command mode → execute command; otherwise → delegate to plugin
- ↑↓: browse command history
- Tab: autocomplete command name

### D5: Command Palette (cli/uix/components/palette.go)

Fuzzy-search overlay, similar to VS Code Ctrl+P / bubbletea-commandpalette.

```go
type Palette struct {
    query     textinput.Model
    actions   []PaletteAction
    results   []PaletteAction    // filtered
    cursor    int
    width     int
    maxHeight int               // max results to show
}

type PaletteAction struct {
    Label       string    // "/containers"
    Description string    // "Manage Docker containers"
    Keywords    []string  // hidden search terms: "docker, ps, container"
    Run         func() tea.Cmd
}
```

**Behavior**:
- Live fuzzy filtering on every keystroke
- ↑↓ to navigate results, Enter to select
- Esc to close
- Results capped at maxHeight (no scroll state)
- Search matches Label + Keywords

**Rendering** (official lipgloss.Place overlay pattern):

```go
func (p Palette) View() string {
    // Render filtered results as a bordered box
    // Center on screen using lipgloss.Place
    content := p.renderResults()
    return lipgloss.Place(p.width, p.maxHeight+4,
        lipgloss.Center, lipgloss.Center,
        content,
    )
}
```

### D6: NavStack (cli/uix/navigation.go)

Simple push/pop stack for panels and modals (not full LazyGit-style context system).

```go
type NavStack[T any] struct {
    items []T
}

func (s *NavStack[T]) Push(item T)
func (s *NavStack[T]) Pop() (T, bool)
func (s *NavStack[T]) Current() (T, bool)
func (s *NavStack[T]) Depth() int
```

Used for:
- Opening a detail panel on top of current plugin
- Showing a modal (confirm dialog, form)
- Plugin doesn't need to know about stack - framework handles routing

### D7: Modal Component (cli/uix/components/modal.go)

```go
type Modal struct {
    title   string
    message string
    buttons []ModalButton
    active  int     // selected button index
    width   int
}

type ModalButton struct {
    Label string
    Run   func() tea.Cmd
}
```

Rendered as centered overlay using `lipgloss.Place`.

### D8: Layout Engine (cli/uix/app.go - integrated into FrameworkApp)

No separate layout package. Simple calculation in FrameworkApp:

```go
func (app *FrameworkApp) recalculate(width, height int) {
    app.width = width
    app.height = height
    app.cmdbar.SetWidth(width)
    app.statusbar.SetWidth(width)
    app.bodyHeight_ = height - cmdbarHeight - statusbarHeight
    if app.bodyHeight_ < 1 {
        app.bodyHeight_ = 1
    }
    app.active.SetSize(width, app.bodyHeight_)
}
```

### D9: Theme (cli/uix/theme/)

```go
// Tokyo Night palette
const (
    ColorBg       = "#1a1b26"
    ColorFg       = "#c0caf5"
    ColorAccent   = "#7aa2f7"
    ColorGreen    = "#9ece6a"
    ColorRed      = "#f7768e"
    ColorYellow   = "#e0af68"
    ColorDim      = "#565f89"
    ColorBorder   = "#3b4261"
    ColorSelected = "#364a82"
)

// Factory functions
func WidthStyle(w int) lipgloss.Style
func Truncate(s string, maxWidth int) string
func Border(title string) lipgloss.Style
```

---

## Docker Module Design (cli/dtui/)

### Package Structure

```
cli/dtui/
  main.go                    # Entry: create FrameworkApp, register plugins, run
  docker/                    # Docker SDK wrapper (REUSE existing, don't modify)
    client.go, container.go, image.go, compose.go, stats.go, logs.go, inspect.go
  plugins/
    containers/
      plugin.go              # Container plugin (implements uix.Plugin)
      table.go               # Container table model
      detail.go              # Detail viewport
    images/
      plugin.go              # Image plugin
    compose/
      plugin.go              # Compose plugin
    deploy/
      plugin.go              # Deploy plugin
    settings/
      plugin.go              # Settings plugin
  config/
    config.go                # Config load/save (REUSE existing)
```

### Plugin: containers

```go
type ContainerPlugin struct {
    client      *dt.Client
    table       table.Model      // bubbles/table
    detail      viewport.Model   // bubbles/viewport
    containers  []dt.Container
    cursor      int
    width       int
    height      int
    // panels
    showLog     *LogPanel
    showStats   *StatsPanel
}
```

**Key patterns** (from official Bubble Tea examples):
- `Update`: global keys first (esc to go back), then delegate to table
- `View`: left table + right detail (JoinHorizontal)
- Panel opening: push to NavStack in parent FrameworkApp
- Docker operations: async via `tea.Cmd` returning custom Msg types

### main.go (new dtui)

```go
func main() {
    client, _ := dt.NewClient()
    defer client.Close()

    app := uix.NewApp()
    app.Register(containers.New(client))
    app.Register(images.New(client))
    app.Register(compose.New(client, configPath))
    app.Register(deploy.New(client, configPath))
    app.Register(settings.New(configPath))

    app.Run()
}
```

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Plugin interface = simplified tea.Model | Follows official Bubble Tea pattern. Each plugin IS a tea.Model. |
| No separate ContextManager | Simpler NavStack. Official examples use simple state enums. |
| Command palette as overlay | VS Code Ctrl+P UX is proven. lipgloss.Place for centering. |
| No antsx.EventEmitter (initially) | Keep v1 simple. Add event bus later for inter-plugin communication. |
| bubbles/table + bubbles/viewport | Official components. Don't reinvent. |
| Layout in FrameworkApp, not separate package | Simple enough. Fewer packages = easier to understand. |
| Tokyo Night theme | Proven dark theme. Good contrast. Already in project. |

## Trade-offs

| Decision | Pro | Con |
|----------|-----|-----|
| State enum (not Context interface) | Simpler, official pattern | Less type-safe than interface |
| No event bus initially | Less complexity | Harder cross-plugin communication |
| Palette over Tab bar | Command-driven = natural for CLI users | Tab bar more discoverable |
| lipgloss.Place for overlays | Clean overlay UI | Overlay hides all content below |
