# dtui Module Development Plan

## Current Status (v0.1)

| Module | Status | Notes |
|--------|--------|-------|
| Welcome | ✅ Done | ASCII logo + command list |
| Containers | ✅ Basic | Table+detail, start/stop/restart/remove with modal |
| Images | ✅ Basic | Table, remove/prune with modal |
| Compose | ✅ Basic | Entry list, up/down confirm (execution pending) |
| Deploy | ✅ Basic | Target list, deploy confirm (execution pending) |
| Settings | ✅ Basic | Config overview, external editor |

## v0.2 — Container Deep Features

### Log Viewer Panel
- Press `l` on a container → full-screen log panel
- Streaming log view with follow mode (tail -f)
- Search within logs (`/`)
- Scroll: ↑↓/PgUp/PgDn/g/G
- Color-coded timestamps

### Stats Monitor Panel
- Press `p` on a running container → real-time stats
- CPU/Memory/Network/PIDs gauges
- Historical trend display (last N samples)
- Auto-refresh every 2s

### Inspect Panel
- Press `i` → container metadata
- Tabs: Overview / Network / Mounts / Env / Config
- Scrollable viewport

### Exec Terminal
- Press `e` → interactive shell
- `/bin/sh` or `/bin/bash` exec
- Output streamed to viewport
- Command history

## v0.3 — Image Management

### Save Image
- Save selected image as tar.gz
- Directory picker for output location
- Progress spinner during save

### Tag Image
- Tag dialog: source → target
- Validation of tag format

### Image History
- Layer history with created/size/comment
- Scrollable viewport

## v0.4 — Compose Execution

### Actual Compose Operations
- `docker compose up -d` execution
- `docker compose down` execution
- Operation output captured and displayed
- Error handling with user-friendly messages

### Service Status
- Parse `docker compose ps` output
- Show running/stopped/error status per service

## v0.5 — Deploy Flow

### Full Deploy Pipeline
1. File picker to select local archive
2. Confirm dialog with target details
3. Backup existing deployment
4. Extract archive
5. Copy to container
6. Show progress at each step

### Status Feedback
- Progress spinner per step
- Success/failure messages
- Backup path displayed on completion

## v0.6 — Settings Management

### Add/Edit/Delete
- Form modal for adding new compose dirs
- Form modal for adding new deploy targets
- Delete with confirmation
- Save to config.json

### Form Fields
- Compose: Name, Path (with dir picker)
- Deploy: Name, Container, HTML Path, Backup Dir

## v0.7 — Visual Polish

### UI Enhancements
- Smooth modal open/close transitions (harmonica)
- Loading spinners for async operations
- Color-coded status indicators
- Better table column widths
- Responsive layout fixes for small terminals

### Quality
- Error boundary recovery
- Graceful Docker daemon disconnect handling
- Config file missing handling

## v0.8 — Future Modules (post v1.0)

| Module | Description | Command |
|--------|-------------|---------|
| modbus | Modbus TCP/RTU testing | `/modbus` |
| mqtt | MQTT publish/subscribe | `/mqtt` |
| kafka | Kafka produce/consume | `/kafka` |
| chat | LLM chat terminal | `/chat` |
| iec104 | IEC 104 protocol testing | `/iec104` |
