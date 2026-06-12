# Research: dtui Plugin Analysis for uix Shell Rewrite

- **Query**: Research existing dtui business modules to understand what needs to be rewritten using the new uix shell
- **Scope**: internal
- **Date**: 2026-06-12

## Findings

### 1. All Plugins with Current Responsibilities

| Plugin | File | Description | Dependencies |
|--------|------|-------------|--------------|
| **containers** | `cli/dtui/plugins/containers/plugin.go` | Manage Docker containers: list, start, stop, restart, delete, view logs | `dt.Client` (Docker SDK) |
| **compose** | `cli/dtui/plugins/compose/plugin.go` | Docker Compose orchestration: up/down compose projects | `dt.Client`, `config.Config` |
| **deploy** | `cli/dtui/plugins/deploy/plugin.go` | Frontend deployment: copy files to containers, zip extraction | `dt.Client`, `config.Config` |
| **images** | `cli/dtui/plugins/images/plugin.go` | Manage Docker images: list, remove, prune | `dt.Client` |
| **config** | `cli/dtui/plugins/settings/plugin.go` | Configuration management: add/remove compose dirs and deploy targets | `config.Config` (no Docker) |
| **test** | `cli/dtui/plugins/test/plugin.go` | Test module exercising uix shell features (reference implementation) | None (pure uix) |

### 2. Docker Dependencies per Plugin

#### containers
- `dt.Client.ListContainers(filter)` — list containers
- `dt.Client.StartContainer(id)` — start container
- `dt.Client.StopContainer(id)` — stop container
- `dt.Client.RestartContainer(id)` — restart container
- `dt.Client.RemoveContainer(id, force)` — remove container
- `dt.Client.FetchLogs(id, opts)` — fetch container logs

#### compose
- `dt.RunComposeUp(path, service)` — execute `docker compose up -d`
- `dt.RunComposeDown(path, service)` — execute `docker compose down`

#### deploy
- `dt.PathType(path)` — detect path type (folder/zip/invalid)
- `dt.UnzipToDir(zip, dest)` — extract zip to directory
- `dt.Client.CopyToContainer(container, dst, src)` — copy files to container via tar

#### images
- `dt.Client.ListImages(filter)` — list images
- `dt.Client.RemoveImage(ref, force)` — remove image
- `dt.Client.PruneImages()` — prune dangling images

#### config
- No Docker dependencies (pure config file operations)

### 3. Current UI Patterns

| Plugin | UI Components Used | Current Patterns |
|--------|-------------------|------------------|
| **containers** | `table.Model`, `viewport.Model`, `LogViewer` | Table + detail split pane, modal confirm, log viewer mode |
| **compose** | `table.Model` | Simple table, modal confirm for up/down |
| **deploy** | `table.Model` | Table + file picker flow, modal confirm, step state machine |
| **images** | `table.Model` | Simple table, modal confirm for remove/prune |
| **config** | `table.Model`, `textinput.Model` | Table + inline form for add entries, modal for type selection |

### 4. Key Patterns from Test Module (Reference Implementation)

The test module (`cli/dtui/plugins/test/plugin.go`) demonstrates the new uix shell patterns:

1. **Module Interface**: Implements `uix.Module` interface with `Name()`, `Description()`, `Aliases()`, `Init()`, `Update()`, `View()`, `SetSize()`, `Bindings()`, `IsRoot()`

2. **Component Usage**:
   - `components.NewPanel(title, width, height)` — main container
   - `components.RenderState(kind, title, msg, width)` — state display (empty/loading/success/warning/error)
   - `components.NewLogViewer(w, h)` — log output with scroll/follow
   - `components.NewSparkline(w, h)` — chart visualization
   - `components.LogHeader(title, follow)` — log panel header

3. **Message Patterns**:
   - `uix.ShowModalMsg{...}` — show modal dialog
   - `uix.ConfirmMsg{Button}` — modal button response
   - `uix.FileSelectedMsg{Path}` — file picker result
   - `uix.AppendMessageMsg{Role, Content}` — append to shared message log

4. **Mode Pattern**: Use boolean flags (`logMode`, `chartMode`) to switch between views; `IsRoot()` returns true when in main view

5. **Key Handling**: Route keys through `handleKey()` dispatchers; delegate to sub-handlers in modes

### 5. Recommended Rewrite Order

| Order | Plugin | Complexity | Rationale |
|-------|--------|------------|-----------|
| 1 | **images** | Low | Simple table + modal pattern, no file picker, minimal state |
| 2 | **compose** | Low | Simple table + modal, but adds Docker compose exec calls |
| 3 | **containers** | Medium | Table + detail pane + log viewer mode, multiple Docker operations |
| 4 | **deploy** | Medium | File picker + step state machine + Docker copy |
| 5 | **config** | Medium | Form inputs + multiple modal types, no Docker but complex state |

### 6. Estimated Complexity per Plugin

| Plugin | Lines | UI Complexity | State Complexity | Docker API Surface |
|--------|-------|---------------|------------------|-------------------|
| **images** | 195 | Low (table + modal) | Low (cursor + pending ref) | 3 calls |
| **compose** | 231 | Low (table + modal) | Medium (pending action) | 2 calls (exec) |
| **containers** | 433 | High (table + detail + log viewer) | Medium (cursor + pending ID + log mode) | 6 calls |
| **deploy** | 271 | Medium (table + file picker + steps) | Medium (step FSM + pending target) | 3 calls |
| **config** | 399 | Medium (table + forms + multiple modals) | High (form steps + multiple pending) | 0 calls |

### 7. uix Components to Use After Rewrite

| Current Pattern | New uix Component |
|-----------------|-------------------|
| `table.Model` (bubbles) | `components.Table` (custom wrapper) or keep `table.Model` |
| `viewport.Model` | `components.LogViewer` |
| Modal via `uix.ShowModalMsg` | Same (already uix-native) |
| File picker via `uix.FileSelectedMsg` | Same (already uix-native) |
| Inline status text | `components.RenderState()` |
| Log output | `components.LogViewer` |
| Charts | `components.Sparkline` or `components.BarChart` |
| Progress | `components.Progress` |
| Text input forms | `components.TextArea` or `textinput.Model` |
| Spinner | `components.Spinner` |

## Caveats / Not Found

1. **Old Plugin Interface**: The existing plugins implement a slightly different interface than the new `uix.Module`. They have `OnActivate()` and `OnDeactivate()` hooks that the test module does not have. Need to check if these are still needed in the new shell.

2. **Plugin Registration**: The old plugins are registered in a different way than the new uix shell expects. The rewrite needs to update the registration mechanism.

3. **Theme Imports**: All plugins import from `cli/uix/theme` already, which is good — the theme is already unified.

4. **Table Component Available**: `components.Table` wraps `bubbles/table` with project theme styling. Available for rewrite.

5. **No Stats Display**: The `docker/stats.go` provides `StreamStats()` but no current plugin uses it. Could be a future feature.

## Appendix: Available uix Components

| Component | File | Description |
|-----------|------|-------------|
| `Table` | `components/table.go` | Wraps `bubbles/table` with project theme styling |
| `Spinner` | `components/spinner.go` | Loading spinner with project theme |
| `Progress` | `components/progress.go` | Progress bar (0.0-1.0) with theme colors |
| `LogViewer` | `components/logviewer.go` | Log output viewer with scroll/follow |
| `Sparkline` | `components/sparkline.go` | Mini chart visualization |
| `BarChart` | `components/barchart.go` | Bar chart visualization |
| `Modal` | `components/modal.go` | Centered overlay dialog with buttons |
| `Panel` | `components/state.go` | Bordered panel container |
| `StateView` | `components/state.go` | State display (empty/loading/success/warning/error) |
| `WelcomeScreen` | `components/welcome.go` | Centered welcome/home screen |
| `TextArea` | `components/textarea.go` | Multi-line text input |
| `Dropdown` | `components/dropdown.go` | Dropdown selection |
| `FilePicker` | `components/filepicker.go` | File selection dialog |
| `Help` | `components/help.go` | Help/keybinding display |
| `StatusBar` | `components/statusbar.go` | Status bar display |
| `CmdBar` | `components/cmdbar.go` | Command input bar |
| `List` | `components/list.go` | List component |
