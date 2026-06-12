package containers

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	dt "zero-service/cli/dtui/internal/docker"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
)

func TestNewModule(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
	if m.state != components.StateLoading {
		t.Errorf("expected state %q, got %q", components.StateLoading, m.state)
	}
	if m.status != "connecting..." {
		t.Errorf("expected status %q, got %q", "connecting...", m.status)
	}
	if m.width != 80 {
		t.Errorf("expected width 80, got %d", m.width)
	}
	if m.height != 20 {
		t.Errorf("expected height 20, got %d", m.height)
	}
}

func TestModuleIdentity(t *testing.T) {
	m := New()
	if m.Name() != "containers" {
		t.Errorf("expected name %q, got %q", "containers", m.Name())
	}
	if m.Description() != "Manage Docker containers" {
		t.Errorf("expected description %q, got %q", "Manage Docker containers", m.Description())
	}
	aliases := m.Aliases()
	if len(aliases) != 1 || aliases[0] != "ctr" {
		t.Errorf("expected aliases [ctr], got %v", aliases)
	}
}

func TestIsRootDefault(t *testing.T) {
	m := New()
	if !m.IsRoot() {
		t.Error("expected IsRoot() true in default state")
	}
}

func TestIsRootInSubmodes(t *testing.T) {
	tests := []struct {
		name       string
		logMode    bool
		detailMode bool
		statsMode  bool
		expected   bool
	}{
		{"default", false, false, false, true},
		{"log mode", true, false, false, false},
		{"detail mode", false, true, false, false},
		{"stats mode", false, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.logMode = tt.logMode
			m.detailMode = tt.detailMode
			m.statsMode = tt.statsMode
			if got := m.IsRoot(); got != tt.expected {
				t.Errorf("IsRoot() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBindingsRootMode(t *testing.T) {
	m := New()
	bindings := m.Bindings()
	if len(bindings) == 0 {
		t.Fatal("expected non-empty bindings in root mode")
	}
	keys := make(map[string]bool)
	for _, b := range bindings {
		for _, k := range b.Keys {
			keys[k] = true
		}
	}
	expectedKeys := []string{"↑↓", "s", "S", "r", "x", "i", "t", "l", "R"}
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("missing expected key binding %q in root mode", k)
		}
	}
}

func TestBindingsDetailMode(t *testing.T) {
	m := New()
	m.detailMode = true
	bindings := m.Bindings()
	if len(bindings) != 2 {
		t.Errorf("expected 2 bindings in detail mode, got %d", len(bindings))
	}
}

func TestBindingsStatsMode(t *testing.T) {
	m := New()
	m.statsMode = true
	bindings := m.Bindings()
	if len(bindings) != 1 {
		t.Errorf("expected 1 binding in stats mode, got %d", len(bindings))
	}
}

func TestBindingsLogMode(t *testing.T) {
	m := New()
	m.logMode = true
	bindings := m.Bindings()
	if len(bindings) != 3 {
		t.Errorf("expected 3 bindings in log mode, got %d", len(bindings))
	}
}

func TestSetSizeValid(t *testing.T) {
	m := New()
	m.SetSize(120, 40)
	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestSetSizeZeroDefaults(t *testing.T) {
	m := New()
	m.SetSize(0, 0)
	if m.width != 80 {
		t.Errorf("expected width 80 for zero input, got %d", m.width)
	}
	if m.height != 20 {
		t.Errorf("expected height 20 for zero input, got %d", m.height)
	}
}

func TestSetSizeNegativeDefaults(t *testing.T) {
	m := New()
	m.SetSize(-10, -5)
	if m.width != 80 {
		t.Errorf("expected width 80 for negative input, got %d", m.width)
	}
	if m.height != 20 {
		t.Errorf("expected height 20 for negative input, got %d", m.height)
	}
}

func TestViewLoadingState(t *testing.T) {
	m := New()
	m.state = components.StateLoading
	m.containers = nil
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in loading state")
	}
}

func TestViewErrorState(t *testing.T) {
	m := New()
	m.state = components.StateError
	m.containers = nil
	m.status = "connection failed"
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in error state")
	}
}

func TestViewEmptyState(t *testing.T) {
	m := New()
	m.state = components.StateEmpty
	m.containers = []dt.Container{}
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in empty state")
	}
}

func TestViewTableState(t *testing.T) {
	m := New()
	m.state = components.StateSuccess
	m.containers = []dt.Container{
		{ID: "abc123", Name: "test-ctr", Image: "nginx:latest", State: "running", Status: "Up 5 minutes"},
	}
	m.updateTableRows()
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in table state")
	}
}

func TestViewDetailModeNilDetail(t *testing.T) {
	m := New()
	m.detailMode = true
	m.detail = nil
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in detail mode with nil detail")
	}
}

func TestViewDetailModeWithDetail(t *testing.T) {
	m := New()
	m.detailMode = true
	m.detail = &dt.ContainerDetail{
		ID:    "abc123",
		Name:  "test-ctr",
		Image: "nginx:latest",
		State: dt.ContainerState{
			Status:  "running",
			Running: true,
		},
	}
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in detail mode with data")
	}
}

func TestViewStatsModeNoHistory(t *testing.T) {
	m := New()
	m.statsMode = true
	m.statsHistory = nil
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in stats mode with no history")
	}
}

func TestViewStatsModeWithHistory(t *testing.T) {
	m := New()
	m.statsMode = true
	m.statsHistory = []dt.StatsEntry{
		{CPUPercent: 25.5, MemPercent: 40.0},
	}
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in stats mode with history")
	}
}

func TestViewLogMode(t *testing.T) {
	m := New()
	m.logMode = true
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in log mode")
	}
}

func TestHandleContainersLoadedWithError(t *testing.T) {
	m := New()
	msg := containersLoadedMsg{err: fmt.Errorf("docker not available")}
	model, cmd := m.handleContainersLoaded(msg)
	if cmd != nil {
		t.Error("expected nil cmd on error")
	}
	mod := model.(*Module)
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
	if mod.status != "docker not available" {
		t.Errorf("expected status %q, got %q", "docker not available", mod.status)
	}
}

func TestHandleContainersLoadedEmpty(t *testing.T) {
	m := New()
	msg := containersLoadedMsg{containers: []dt.Container{}}
	model, _ := m.handleContainersLoaded(msg)
	mod := model.(*Module)
	if mod.state != components.StateEmpty {
		t.Errorf("expected state %q, got %q", components.StateEmpty, mod.state)
	}
}

func TestHandleContainersLoadedWithData(t *testing.T) {
	m := New()
	msg := containersLoadedMsg{
		containers: []dt.Container{
			{ID: "abc", Name: "web", Image: "nginx", State: "running"},
		},
		status: "1 container",
	}
	model, _ := m.handleContainersLoaded(msg)
	mod := model.(*Module)
	if mod.state != components.StateSuccess {
		t.Errorf("expected state %q, got %q", components.StateSuccess, mod.state)
	}
	if mod.status != "1 container" {
		t.Errorf("expected status %q, got %q", "1 container", mod.status)
	}
	if len(mod.containers) != 1 {
		t.Errorf("expected 1 container, got %d", len(mod.containers))
	}
}

func TestHandleActionResultWithError(t *testing.T) {
	m := New()
	msg := actionResultMsg{err: fmt.Errorf("action failed")}
	model, cmd := m.handleActionResult(msg)
	if cmd != nil {
		t.Error("expected nil cmd on error")
	}
	mod := model.(*Module)
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
}

func TestHandleActionResultSuccess(t *testing.T) {
	m := New()
	msg := actionResultMsg{
		containers: []dt.Container{{ID: "abc", Name: "web"}},
		status:     "Action complete",
	}
	model, _ := m.handleActionResult(msg)
	mod := model.(*Module)
	if mod.state != components.StateSuccess {
		t.Errorf("expected state %q, got %q", components.StateSuccess, mod.state)
	}
}

func TestHandleLogDoneWithError(t *testing.T) {
	m := New()
	msg := logDoneMsg{err: fmt.Errorf("log fetch failed")}
	model, _ := m.handleLogDone(msg)
	if model == nil {
		t.Error("expected non-nil model")
	}
}

func TestHandleLogDoneWithLines(t *testing.T) {
	m := New()
	msg := logDoneMsg{lines: []string{"line1", "line2"}}
	model, _ := m.handleLogDone(msg)
	if model == nil {
		t.Error("expected non-nil model")
	}
}

func TestHandleInspectDoneWithError(t *testing.T) {
	m := New()
	m.detailMode = true
	msg := inspectDoneMsg{err: fmt.Errorf("inspect failed")}
	model, _ := m.handleInspectDone(msg)
	mod := model.(*Module)
	if mod.detailMode {
		t.Error("expected detailMode false after error")
	}
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
}

func TestHandleInspectDoneSuccess(t *testing.T) {
	m := New()
	m.detailMode = true
	detail := &dt.ContainerDetail{ID: "abc", Name: "web"}
	msg := inspectDoneMsg{detail: detail}
	model, _ := m.handleInspectDone(msg)
	mod := model.(*Module)
	if mod.detail == nil {
		t.Error("expected non-nil detail")
	}
	if mod.status != "" {
		t.Errorf("expected empty status, got %q", mod.status)
	}
}

func TestHandleStatsDone(t *testing.T) {
	m := New()
	m.statsMode = true
	m.statsCancel = func() {}
	m.statsCh = make(chan dt.StatsEntry)
	m.statsErrCh = make(chan error)
	msg := statsDoneMsg{}
	model, _ := m.handleStatsDone(msg)
	mod := model.(*Module)
	if mod.statsCancel != nil {
		t.Error("expected nil statsCancel after done")
	}
	if mod.statsCh != nil {
		t.Error("expected nil statsCh after done")
	}
}

func TestHandleStatsDoneWithError(t *testing.T) {
	m := New()
	m.statsMode = true
	msg := statsDoneMsg{err: fmt.Errorf("stream ended")}
	model, _ := m.handleStatsDone(msg)
	mod := model.(*Module)
	if mod.status == "" {
		t.Error("expected non-empty status after error")
	}
}

func TestHandleStatsEntry(t *testing.T) {
	m := New()
	m.statsMode = true
	entry := dt.StatsEntry{CPUPercent: 50.0, MemPercent: 60.0}
	msg := statsEntryMsg{entry: entry}
	model, _ := m.handleStatsEntry(msg)
	mod := model.(*Module)
	if len(mod.statsHistory) != 1 {
		t.Errorf("expected 1 stats entry, got %d", len(mod.statsHistory))
	}
}

func TestHandleStatsEntryMaxHistory(t *testing.T) {
	m := New()
	m.statsMode = true
	for i := 0; i < 65; i++ {
		m.statsHistory = append(m.statsHistory, dt.StatsEntry{CPUPercent: float64(i)})
	}
	entry := dt.StatsEntry{CPUPercent: 100.0}
	msg := statsEntryMsg{entry: entry}
	model, _ := m.handleStatsEntry(msg)
	mod := model.(*Module)
	if len(mod.statsHistory) > 60 {
		t.Errorf("expected max 60 history entries, got %d", len(mod.statsHistory))
	}
}

func TestHandleConfirmForceDelete(t *testing.T) {
	m := New()
	m.pendingRemoveID = "abc123"
	model, cmd := m.handleConfirm("Force Delete")
	mod := model.(*Module)
	if mod.pendingRemoveID != "" {
		t.Error("expected pendingRemoveID cleared")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for force delete")
	}
	if mod.state != components.StateLoading {
		t.Errorf("expected state %q, got %q", components.StateLoading, mod.state)
	}
}

func TestHandleConfirmCancel(t *testing.T) {
	m := New()
	m.pendingRemoveID = "abc123"
	model, cmd := m.handleConfirm("Cancel")
	mod := model.(*Module)
	if mod.pendingRemoveID != "" {
		t.Error("expected pendingRemoveID cleared")
	}
	if cmd != nil {
		t.Error("expected nil cmd for cancel")
	}
	if mod.status != "cancelled" {
		t.Errorf("expected status %q, got %q", "cancelled", mod.status)
	}
}

func TestHandleConfirmEmptyID(t *testing.T) {
	m := New()
	m.pendingRemoveID = ""
	model, cmd := m.handleConfirm("Force Delete")
	mod := model.(*Module)
	if cmd != nil {
		t.Error("expected nil cmd for empty ID")
	}
	if mod.status != "cancelled" {
		t.Errorf("expected status %q, got %q", "cancelled", mod.status)
	}
}

func TestConfirmRemoveWithContainers(t *testing.T) {
	m := New()
	m.containers = []dt.Container{
		{ID: "abc123", Name: "web", State: "running"},
	}
	m.updateTableRows()
	model, cmd := m.confirmRemove()
	mod := model.(*Module)
	if mod.pendingRemoveID != "abc123" {
		t.Errorf("expected pendingRemoveID %q, got %q", "abc123", mod.pendingRemoveID)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestConfirmRemoveEmpty(t *testing.T) {
	m := New()
	m.containers = []dt.Container{}
	model, cmd := m.confirmRemove()
	if cmd != nil {
		t.Error("expected nil cmd for empty containers")
	}
	_ = model
}

func TestUpdateWithKeyMsgInRootMode(t *testing.T) {
	m := New()
	m.containers = []dt.Container{
		{ID: "abc", Name: "web", State: "running"},
		{ID: "def", Name: "api", State: "exited"},
	}
	m.updateTableRows()

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := m.Update(msg)
	mod := model.(*Module)
	if mod.table.Cursor() != 1 {
		t.Errorf("expected cursor 1, got %d", mod.table.Cursor())
	}
}

func TestUpdateWithKeyMsgInLogMode(t *testing.T) {
	m := New()
	m.logMode = true
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, cmd := m.Update(msg)
	mod := model.(*Module)
	if mod.logMode {
		t.Error("expected logMode false after esc")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after exiting log mode")
	}
}

func TestUpdateWithConfirmMsg(t *testing.T) {
	m := New()
	m.pendingRemoveID = "abc123"
	msg := uix.ConfirmMsg{Button: "Force Delete"}
	model, _ := m.Update(msg)
	mod := model.(*Module)
	if mod.pendingRemoveID != "" {
		t.Error("expected pendingRemoveID cleared after confirm")
	}
}

func TestUpdateWithUnknownMsg(t *testing.T) {
	m := New()
	model, cmd := m.Update("unknown message")
	if model == nil {
		t.Error("expected non-nil model")
	}
	_ = cmd
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		state string
	}{
		{"running"},
		{"exited"},
		{"paused"},
	}
	for _, tt := range tests {
		icon := statusIcon(tt.state)
		if icon == "" {
			t.Errorf("expected non-empty icon for state %q", tt.state)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1048576, "1.0 MiB"},
	}
	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestToggleContainerEmpty(t *testing.T) {
	m := New()
	m.containers = []dt.Container{}
	model, cmd := m.toggleContainer()
	if cmd != nil {
		t.Error("expected nil cmd for empty containers")
	}
	_ = model
}

func TestStopContainerEmpty(t *testing.T) {
	m := New()
	m.containers = []dt.Container{}
	model, cmd := m.stopContainer()
	if cmd != nil {
		t.Error("expected nil cmd for empty containers")
	}
	_ = model
}

func TestRestartContainerEmpty(t *testing.T) {
	m := New()
	m.containers = []dt.Container{}
	model, cmd := m.restartContainer()
	if cmd != nil {
		t.Error("expected nil cmd for empty containers")
	}
	_ = model
}

func TestOpenLogsEmpty(t *testing.T) {
	m := New()
	m.containers = []dt.Container{}
	model, cmd := m.openLogs()
	if cmd != nil {
		t.Error("expected nil cmd for empty containers")
	}
	_ = model
}

func TestOpenDetailEmpty(t *testing.T) {
	m := New()
	m.containers = []dt.Container{}
	model, cmd := m.openDetail()
	if cmd != nil {
		t.Error("expected nil cmd for empty containers")
	}
	_ = model
}

func TestOpenStatsEmpty(t *testing.T) {
	m := New()
	m.containers = []dt.Container{}
	model, cmd := m.openStats()
	if cmd != nil {
		t.Error("expected nil cmd for empty containers")
	}
	_ = model
}

func TestOpenStatsNotRunning(t *testing.T) {
	m := New()
	m.containers = []dt.Container{
		{ID: "abc", Name: "web", State: "exited"},
	}
	m.updateTableRows()
	model, cmd := m.openStats()
	if cmd != nil {
		t.Error("expected nil cmd for non-running container")
	}
	mod := model.(*Module)
	if mod.status != "Container not running" {
		t.Errorf("expected status %q, got %q", "Container not running", mod.status)
	}
}

func TestHandleKeyNavigation(t *testing.T) {
	m := New()
	m.containers = []dt.Container{
		{ID: "abc", Name: "web", State: "running"},
		{ID: "def", Name: "api", State: "exited"},
		{ID: "ghi", Name: "db", State: "running"},
	}
	m.updateTableRows()

	tests := []struct {
		key      string
		expected int
	}{
		{"down", 1},
		{"j", 2},
		{"up", 1},
		{"k", 0},
	}
	for _, tt := range tests {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
		model, _ := m.handleKey(msg)
		mod := model.(*Module)
		if mod.table.Cursor() != tt.expected {
			t.Errorf("key %q: expected cursor %d, got %d", tt.key, tt.expected, mod.table.Cursor())
		}
	}
}

func TestHandleDetailKeyEsc(t *testing.T) {
	m := New()
	m.detailMode = true
	m.detail = &dt.ContainerDetail{ID: "abc"}
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, _ := m.handleDetailKey(msg)
	mod := model.(*Module)
	if mod.detailMode {
		t.Error("expected detailMode false after esc")
	}
	if mod.detail != nil {
		t.Error("expected nil detail after esc")
	}
}

func TestHandleStatsKeyEsc(t *testing.T) {
	m := New()
	m.statsMode = true
	m.statsCancel = func() {}
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, _ := m.handleStatsKey(msg)
	mod := model.(*Module)
	if mod.statsMode {
		t.Error("expected statsMode false after esc")
	}
	if mod.statsCancel != nil {
		t.Error("expected nil statsCancel after esc")
	}
}

func TestHandleLogKeyEsc(t *testing.T) {
	m := New()
	m.logMode = true
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, cmd := m.handleLogKey(msg)
	mod := model.(*Module)
	if mod.logMode {
		t.Error("expected logMode false after esc")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after exiting log mode")
	}
}

func TestUpdateTableRowsBoundsCheck(t *testing.T) {
	m := New()
	m.containers = []dt.Container{
		{ID: "abc", Name: "web", Image: "nginx", State: "running", Status: "Up"},
	}
	m.updateTableRows()
	m.table.SetCursor(5)
	m.updateTableRows()
	if m.table.Cursor() >= len(m.containers) {
		t.Errorf("cursor %d out of bounds for %d containers", m.table.Cursor(), len(m.containers))
	}
}

func TestEnsureClientCachesError(t *testing.T) {
	m := New()
	m.clientErr = fmt.Errorf("docker not available")
	client := m.ensureClient()
	if client != nil {
		t.Error("expected nil client when clientErr is set")
	}
}

func TestEnsureClientCachesClient(t *testing.T) {
	m := New()
	m.client = &dt.Client{}
	client := m.ensureClient()
	if client == nil {
		t.Error("expected non-nil client when cached")
	}
}
