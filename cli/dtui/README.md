# dtui

Docker Terminal UI — a production terminal app for managing Docker containers, images, compose projects, and deployments. Built on the `uix` chat-like TUI shell framework.

## Quick Start

```bash
go run ./cli/dtui
```

The app starts without the Docker daemon running. Docker operations (container/image/compose/deploy) require the daemon to be available; config and test modules work without it.

## Command Palette

Press `/` to open the command palette. Type a module name or alias to enter it.

| Module | Key | Aliases | Description |
|--------|-----|---------|-------------|
| `/test` | `test` | `t`, `demo` | Exercise uix shell features without Docker |
| `/containers` | `containers` | `ctr` | Manage Docker containers |
| `/images` | `images` | `img` | Manage Docker images |
| `/compose` | `compose` | `dc` | Manage Docker Compose projects |
| `/deploy` | `deploy` | `dep` | Deploy applications to Docker containers |
| `/config` | `config` | `cfg` | Manage application configuration |

## Prompt Modes

| Prefix | Mode | Behavior |
|--------|------|----------|
| (none) | chat | Send message to mock runner |
| `/` | command | Open command/module palette |
| `@` | reference | Show reference placeholders |
| `#` | resource | Open file picker; selection sent to active module |
| `!` | shell | Disabled — appends system warning, never executes |

## Module Keys

### /containers (ctr)

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Select container |
| `s` | Start/stop toggle |
| `S` | Stop |
| `r` | Restart |
| `x` | Delete (requires confirmation) |
| `i` | Inspect/details |
| `t` | Stats (live stream) |
| `l` | Logs |
| `R` | Refresh |

Subviews: `↑↓` scroll, `f` follow (logs), `esc/q` back.

### /images (img)

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Select image |
| `h` | History |
| `T` | Tag (opens form) |
| `e` | Export/save (opens form) |
| `x` | Remove (requires confirmation) |
| `p` | Prune dangling images (requires confirmation) |
| `r` | Refresh |

### /compose (dc)

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Select project |
| `u` | Compose up (requires confirmation) |
| `d` | Compose down (requires confirmation) |
| `o` / `l` | View output/logs |
| `r` | Refresh |

### /deploy (dep)

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Select target |
| `d` | Start deploy (select file with `#`, then confirm) |
| `#` | Select deployment file (from prompt) |
| `h` | View deploy history |
| `l` | View deploy output log |
| `r` | Refresh |

### /config (cfg)

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Select entry |
| `a` | Add entry (compose dir / deploy target / deploy package) |
| `d` | Delete entry (requires confirmation) |
| `e` | Open config in editor |
| `r` | Refresh |

### /test (t, demo)

| Key | Action |
|-----|--------|
| `m` | Open test modal |
| `#` | Open file picker |
| `l` | Log view |
| `c` | Chart demo |
| `a` | Append log line |
| `e` | Show error state |
| `r` | Reset |

## Global Keys

| Key | Action |
|-----|--------|
| `esc` | Close overlay, or exit active module |
| `ctrl+c` / `/exit` | Exit dtui |

## Docker Requirements

Docker daemon is only required for operations that touch Docker:

- **No daemon needed**: app startup, `/test`, `/config` (view/edit config)
- **Daemon needed**: `/containers` (list/inspect/stats/logs/actions), `/images` (list/history/tag/export/remove/prune), `/compose` (up/down), `/deploy` (deploy execution)

All Docker modules use lazy client initialization — the client is created on first use, not at startup.

## Safety Model

Destructive and overwrite-style operations require a second confirmation modal before execution:

- **Container remove** (`x`) — shows container name and ID
- **Image remove** (`x`) — shows image repository name
- **Image prune** (`p`) — warns about removing all dangling images
- **Compose up** (`u`) — shows the full `docker compose up -d` command
- **Compose down** (`d`) — shows the full `docker compose down` command
- **Deploy** (`d`) — shows target, container, HTML path, backup dir, and source file

Deploy operations include automatic safety controls:

1. **Backup** — container content is copied to the backup directory before overwrite
2. **History** — every deploy is recorded with timestamp, target, action, success/failure, and error detail
3. **Cleanup** — old backups are auto-cleaned (keeps the 5 most recent)
4. **History view** — `h` key shows deploy history with status icons

Read-only operations (list, inspect, logs, stats, history, config display) do not require confirmation.

## Configuration

Config file: `~/.dtui/config.json`

The config module (`/config`) manages three sections:

- **ComposeDirs** — directories containing `docker-compose.yml` files
- **DeployTargets** — deployment target definitions (name, container, HTML path, backup dir)
- **DeployPackages** — reusable deployment package paths

Config can be edited through the TUI forms (`a` add, `d` delete) or opened in an external editor (`e`).

## Build

```bash
# Build for current platform
go build -o cli/dtui/bin/dtui ./cli/dtui

# Build all platforms (darwin/linux, amd64/arm64)
./cli/dtui/build.sh
```

Build outputs in `cli/dtui/bin/`:

| Binary | Platform |
|--------|----------|
| `dtui` | Current platform |
| `dtui-darwin-amd64` | macOS Intel |
| `dtui-darwin-arm64` | macOS Apple Silicon |
| `dtui-linux-amd64` | Linux x86_64 |
| `dtui-linux-arm64` | Linux ARM64 |

## Validation

```bash
go test ./cli/uix/... ./cli/dtui/...
go build ./cli/dtui
go vet ./cli/uix/... ./cli/dtui/...
git diff --check
```

## Architecture

```text
cli/dtui/
  main.go                    # Shell setup, module registration, startup messages
  build.sh                   # Cross-platform build script
  internal/config/           # Config file management (JSON persistence)
  internal/docker/           # Docker client wrapper (lazy init)
  plugins/containers/        # Container management module
  plugins/images/            # Image management module
  plugins/compose/           # Docker Compose module
  plugins/deploy/            # Deployment module (backup/history/rollback)
  plugins/config/            # Configuration management module
  plugins/test/              # Shell feature validation module (no Docker)

cli/uix/
  app.go                     # Shell routing, prompt modes, overlays, commands, modules
  plugin.go                  # Module, Command, HelpBinding contracts
  registry.go                # Module and command registries
  timeline.go                # Message timeline rendering
  runner.go                  # Runner interface and mock runner
  components/                # Prompt, dropdown, modal, file picker, log viewer, table, spinner, etc.
  theme/                     # Colors, borders, truncation helpers
```
