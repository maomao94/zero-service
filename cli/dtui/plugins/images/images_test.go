package images

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
	if m.Name() != "images" {
		t.Errorf("expected name %q, got %q", "images", m.Name())
	}
	if m.Description() != "Manage Docker images" {
		t.Errorf("expected description %q, got %q", "Manage Docker images", m.Description())
	}
	aliases := m.Aliases()
	if len(aliases) != 1 || aliases[0] != "img" {
		t.Errorf("expected aliases [img], got %v", aliases)
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
		name        string
		historyMode bool
		tagMode     bool
		saveMode    bool
		expected    bool
	}{
		{"default", false, false, false, true},
		{"history mode", true, false, false, false},
		{"tag mode", false, true, false, false},
		{"save mode", false, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.historyMode = tt.historyMode
			m.tagMode = tt.tagMode
			m.saveMode = tt.saveMode
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
	expectedKeys := []string{"↑↓", "h", "T", "e", "x", "p", "r"}
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("missing expected key binding %q in root mode", k)
		}
	}
}

func TestBindingsHistoryMode(t *testing.T) {
	m := New()
	m.historyMode = true
	bindings := m.Bindings()
	if len(bindings) != 2 {
		t.Errorf("expected 2 bindings in history mode, got %d", len(bindings))
	}
}

func TestBindingsTagMode(t *testing.T) {
	m := New()
	m.tagMode = true
	bindings := m.Bindings()
	if len(bindings) != 2 {
		t.Errorf("expected 2 bindings in tag mode, got %d", len(bindings))
	}
}

func TestBindingsSaveMode(t *testing.T) {
	m := New()
	m.saveMode = true
	bindings := m.Bindings()
	if len(bindings) != 2 {
		t.Errorf("expected 2 bindings in save mode, got %d", len(bindings))
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
	m.images = nil
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in loading state")
	}
}

func TestViewErrorState(t *testing.T) {
	m := New()
	m.state = components.StateError
	m.images = nil
	m.status = "connection failed"
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in error state")
	}
}

func TestViewEmptyState(t *testing.T) {
	m := New()
	m.state = components.StateEmpty
	m.images = []dt.Image{}
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in empty state")
	}
}

func TestViewTableState(t *testing.T) {
	m := New()
	m.state = components.StateSuccess
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123", Size: "100MB"},
	}
	m.updateTableRows()
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in table state")
	}
}

func TestViewHistoryModeNoHistory(t *testing.T) {
	m := New()
	m.historyMode = true
	m.history = nil
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in history mode with no history")
	}
}

func TestViewHistoryModeWithHistory(t *testing.T) {
	m := New()
	m.historyMode = true
	m.history = []dt.ImageHistoryEntry{
		{ID: "abc123", Size: 1024},
	}
	m.log.SetLines([]string{"#1  abc123  1.0KiB"})
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in history mode with data")
	}
}

func TestViewTagMode(t *testing.T) {
	m := New()
	m.tagMode = true
	m.pendingRef = "nginx:latest"
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in tag mode")
	}
}

func TestViewSaveMode(t *testing.T) {
	m := New()
	m.saveMode = true
	m.pendingRef = "nginx:latest"
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in save mode")
	}
}

func TestHandleImagesLoadedWithError(t *testing.T) {
	m := New()
	msg := imagesLoadedMsg{err: fmt.Errorf("docker not available")}
	model, cmd := m.handleImagesLoaded(msg)
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

func TestHandleImagesLoadedEmpty(t *testing.T) {
	m := New()
	msg := imagesLoadedMsg{images: []dt.Image{}}
	model, _ := m.handleImagesLoaded(msg)
	mod := model.(*Module)
	if mod.state != components.StateEmpty {
		t.Errorf("expected state %q, got %q", components.StateEmpty, mod.state)
	}
}

func TestHandleImagesLoadedWithData(t *testing.T) {
	m := New()
	msg := imagesLoadedMsg{
		images: []dt.Image{
			{Repository: "nginx", Tag: "latest", ID: "abc123"},
		},
	}
	model, _ := m.handleImagesLoaded(msg)
	mod := model.(*Module)
	if mod.state != components.StateSuccess {
		t.Errorf("expected state %q, got %q", components.StateSuccess, mod.state)
	}
	if len(mod.images) != 1 {
		t.Errorf("expected 1 image, got %d", len(mod.images))
	}
}

func TestHandleRemoveResultWithError(t *testing.T) {
	m := New()
	msg := removeResultMsg{err: fmt.Errorf("remove failed")}
	model, cmd := m.handleRemoveResult(msg)
	if cmd != nil {
		t.Error("expected nil cmd on error")
	}
	mod := model.(*Module)
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
}

func TestHandleRemoveResultSuccess(t *testing.T) {
	m := New()
	msg := removeResultMsg{
		images: []dt.Image{{Repository: "nginx", Tag: "latest"}},
	}
	model, _ := m.handleRemoveResult(msg)
	mod := model.(*Module)
	if mod.state != components.StateSuccess {
		t.Errorf("expected state %q, got %q", components.StateSuccess, mod.state)
	}
	if mod.status != "Image removed" {
		t.Errorf("expected status %q, got %q", "Image removed", mod.status)
	}
}

func TestHandlePruneResultWithError(t *testing.T) {
	m := New()
	msg := pruneResultMsg{err: fmt.Errorf("prune failed")}
	model, cmd := m.handlePruneResult(msg)
	if cmd != nil {
		t.Error("expected nil cmd on error")
	}
	mod := model.(*Module)
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
}

func TestHandlePruneResultWithReclaimed(t *testing.T) {
	m := New()
	msg := pruneResultMsg{
		images:    []dt.Image{},
		reclaimed: 1024,
	}
	model, _ := m.handlePruneResult(msg)
	mod := model.(*Module)
	if mod.state != components.StateSuccess {
		t.Errorf("expected state %q, got %q", components.StateSuccess, mod.state)
	}
	if mod.status == "" {
		t.Error("expected non-empty status")
	}
}

func TestHandlePruneResultNoReclaimed(t *testing.T) {
	m := New()
	msg := pruneResultMsg{
		images:    []dt.Image{},
		reclaimed: 0,
	}
	model, _ := m.handlePruneResult(msg)
	mod := model.(*Module)
	if mod.status != "No dangling images to prune" {
		t.Errorf("expected status %q, got %q", "No dangling images to prune", mod.status)
	}
}

func TestHandleHistoryDoneWithError(t *testing.T) {
	m := New()
	m.historyMode = true
	msg := historyDoneMsg{err: fmt.Errorf("history failed")}
	model, _ := m.handleHistoryDone(msg)
	mod := model.(*Module)
	if mod.historyMode {
		t.Error("expected historyMode false after error")
	}
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
}

func TestHandleHistoryDoneSuccess(t *testing.T) {
	m := New()
	m.historyMode = true
	msg := historyDoneMsg{
		entries: []dt.ImageHistoryEntry{
			{ID: "abc123", Size: 1024, CreatedBy: "RUN apt-get update"},
		},
	}
	model, _ := m.handleHistoryDone(msg)
	mod := model.(*Module)
	if mod.status != "" {
		t.Errorf("expected empty status, got %q", mod.status)
	}
	if len(mod.history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(mod.history))
	}
}

func TestHandleTagDoneWithError(t *testing.T) {
	m := New()
	msg := tagDoneMsg{err: fmt.Errorf("tag failed")}
	model, _ := m.handleTagDone(msg)
	mod := model.(*Module)
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
}

func TestHandleTagDoneSuccess(t *testing.T) {
	m := New()
	msg := tagDoneMsg{tag: "myapp:v2"}
	model, cmd := m.handleTagDone(msg)
	mod := model.(*Module)
	if mod.status != "Tagged as myapp:v2" {
		t.Errorf("expected status %q, got %q", "Tagged as myapp:v2", mod.status)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd to reload images")
	}
}

func TestHandleSaveDoneWithError(t *testing.T) {
	m := New()
	msg := saveDoneMsg{err: fmt.Errorf("save failed")}
	model, _ := m.handleSaveDone(msg)
	mod := model.(*Module)
	if mod.state != components.StateError {
		t.Errorf("expected state %q, got %q", components.StateError, mod.state)
	}
}

func TestHandleSaveDoneSuccess(t *testing.T) {
	m := New()
	msg := saveDoneMsg{path: "/tmp/nginx.tar"}
	model, _ := m.handleSaveDone(msg)
	mod := model.(*Module)
	if mod.status != "Image saved to /tmp/nginx.tar" {
		t.Errorf("expected status %q, got %q", "Image saved to /tmp/nginx.tar", mod.status)
	}
}

func TestHandleConfirmRemove(t *testing.T) {
	m := New()
	m.pendingRemoveRef = "nginx:latest"
	model, cmd := m.handleConfirm("Remove")
	mod := model.(*Module)
	if mod.pendingRemoveRef != "" {
		t.Error("expected pendingRemoveRef cleared")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for remove")
	}
	if mod.state != components.StateLoading {
		t.Errorf("expected state %q, got %q", components.StateLoading, mod.state)
	}
}

func TestHandleConfirmCancel(t *testing.T) {
	m := New()
	m.pendingRemoveRef = "nginx:latest"
	model, cmd := m.handleConfirm("Cancel")
	mod := model.(*Module)
	if mod.pendingRemoveRef != "" {
		t.Error("expected pendingRemoveRef cleared")
	}
	if cmd != nil {
		t.Error("expected nil cmd for cancel")
	}
	if mod.status != "cancelled" {
		t.Errorf("expected status %q, got %q", "cancelled", mod.status)
	}
}

func TestHandleConfirmEmptyRef(t *testing.T) {
	m := New()
	m.pendingRemoveRef = ""
	model, cmd := m.handleConfirm("Remove")
	mod := model.(*Module)
	if cmd != nil {
		t.Error("expected nil cmd for empty ref")
	}
	if mod.status != "cancelled" {
		t.Errorf("expected status %q, got %q", "cancelled", mod.status)
	}
}

func TestConfirmRemoveWithImages(t *testing.T) {
	m := New()
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123"},
	}
	m.updateTableRows()
	model, cmd := m.confirmRemove()
	mod := model.(*Module)
	if mod.pendingRemoveRef != "nginx:latest" {
		t.Errorf("expected pendingRemoveRef %q, got %q", "nginx:latest", mod.pendingRemoveRef)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestConfirmRemoveEmpty(t *testing.T) {
	m := New()
	m.images = []dt.Image{}
	model, cmd := m.confirmRemove()
	if cmd != nil {
		t.Error("expected nil cmd for empty images")
	}
	_ = model
}

func TestOpenHistoryEmpty(t *testing.T) {
	m := New()
	m.images = []dt.Image{}
	model, cmd := m.openHistory()
	if cmd != nil {
		t.Error("expected nil cmd for empty images")
	}
	_ = model
}

func TestOpenTagFormEmpty(t *testing.T) {
	m := New()
	m.images = []dt.Image{}
	model, cmd := m.openTagForm()
	if cmd != nil {
		t.Error("expected nil cmd for empty images")
	}
	_ = model
}

func TestOpenSaveFormEmpty(t *testing.T) {
	m := New()
	m.images = []dt.Image{}
	model, cmd := m.openSaveForm()
	if cmd != nil {
		t.Error("expected nil cmd for empty images")
	}
	_ = model
}

func TestOpenHistoryWithImages(t *testing.T) {
	m := New()
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123"},
	}
	m.updateTableRows()
	model, cmd := m.openHistory()
	mod := model.(*Module)
	if !mod.historyMode {
		t.Error("expected historyMode true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestOpenTagFormWithImages(t *testing.T) {
	m := New()
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123"},
	}
	m.updateTableRows()
	model, _ := m.openTagForm()
	mod := model.(*Module)
	if !mod.tagMode {
		t.Error("expected tagMode true")
	}
	if mod.pendingRef != "nginx:latest" {
		t.Errorf("expected pendingRef %q, got %q", "nginx:latest", mod.pendingRef)
	}
}

func TestOpenSaveFormWithImages(t *testing.T) {
	m := New()
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123"},
	}
	m.updateTableRows()
	model, _ := m.openSaveForm()
	mod := model.(*Module)
	if !mod.saveMode {
		t.Error("expected saveMode true")
	}
	if mod.pendingRef != "nginx:latest" {
		t.Errorf("expected pendingRef %q, got %q", "nginx:latest", mod.pendingRef)
	}
}

func TestHandleHistoryKeyEsc(t *testing.T) {
	m := New()
	m.historyMode = true
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, _ := m.handleHistoryKey(msg)
	mod := model.(*Module)
	if mod.historyMode {
		t.Error("expected historyMode false after esc")
	}
}

func TestHandleTagKeyEsc(t *testing.T) {
	m := New()
	m.tagMode = true
	m.pendingRef = "nginx:latest"
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, _ := m.handleTagKey(msg)
	mod := model.(*Module)
	if mod.tagMode {
		t.Error("expected tagMode false after esc")
	}
	if mod.pendingRef != "" {
		t.Error("expected pendingRef cleared")
	}
}

func TestHandleTagKeyEnterEmpty(t *testing.T) {
	m := New()
	m.tagMode = true
	m.pendingRef = "nginx:latest"
	m.tagInput.SetValue("")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := m.handleTagKey(msg)
	mod := model.(*Module)
	if mod.status != "Tag cannot be empty" {
		t.Errorf("expected status %q, got %q", "Tag cannot be empty", mod.status)
	}
}

func TestHandleTagKeyEnterValid(t *testing.T) {
	m := New()
	m.tagMode = true
	m.pendingRef = "nginx:latest"
	m.tagInput.SetValue("myapp:v2")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	model, cmd := m.handleTagKey(msg)
	mod := model.(*Module)
	if mod.tagMode {
		t.Error("expected tagMode false after enter")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
	if mod.state != components.StateLoading {
		t.Errorf("expected state %q, got %q", components.StateLoading, mod.state)
	}
}

func TestHandleSaveKeyEsc(t *testing.T) {
	m := New()
	m.saveMode = true
	m.pendingRef = "nginx:latest"
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, _ := m.handleSaveKey(msg)
	mod := model.(*Module)
	if mod.saveMode {
		t.Error("expected saveMode false after esc")
	}
	if mod.pendingRef != "" {
		t.Error("expected pendingRef cleared")
	}
}

func TestHandleSaveKeyEnterEmpty(t *testing.T) {
	m := New()
	m.saveMode = true
	m.pendingRef = "nginx:latest"
	m.saveInput.SetValue("")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := m.handleSaveKey(msg)
	mod := model.(*Module)
	if mod.status != "Path cannot be empty" {
		t.Errorf("expected status %q, got %q", "Path cannot be empty", mod.status)
	}
}

func TestHandleSaveKeyEnterValid(t *testing.T) {
	m := New()
	m.saveMode = true
	m.pendingRef = "nginx:latest"
	m.saveInput.SetValue("/tmp/nginx.tar")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	model, cmd := m.handleSaveKey(msg)
	mod := model.(*Module)
	if mod.saveMode {
		t.Error("expected saveMode false after enter")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
	if mod.state != components.StateLoading {
		t.Errorf("expected state %q, got %q", components.StateLoading, mod.state)
	}
}

func TestUpdateWithKeyMsgInRootMode(t *testing.T) {
	m := New()
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123"},
		{Repository: "redis", Tag: "7", ID: "def456"},
	}
	m.updateTableRows()

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := m.Update(msg)
	mod := model.(*Module)
	if mod.table.Cursor() != 1 {
		t.Errorf("expected cursor 1, got %d", mod.table.Cursor())
	}
}

func TestUpdateWithKeyMsgInHistoryMode(t *testing.T) {
	m := New()
	m.historyMode = true
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	model, _ := m.Update(msg)
	mod := model.(*Module)
	if mod.historyMode {
		t.Error("expected historyMode false after esc")
	}
}

func TestUpdateWithConfirmMsg(t *testing.T) {
	m := New()
	m.pendingRemoveRef = "nginx:latest"
	msg := uix.ConfirmMsg{Button: "Remove"}
	model, _ := m.Update(msg)
	mod := model.(*Module)
	if mod.pendingRemoveRef != "" {
		t.Error("expected pendingRemoveRef cleared after confirm")
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

func TestHandleKeyNavigation(t *testing.T) {
	m := New()
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123"},
		{Repository: "redis", Tag: "7", ID: "def456"},
		{Repository: "postgres", Tag: "15", ID: "ghi789"},
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

func TestUpdateTableRowsBoundsCheck(t *testing.T) {
	m := New()
	m.images = []dt.Image{
		{Repository: "nginx", Tag: "latest", ID: "abc123"},
	}
	m.updateTableRows()
	m.table.SetCursor(5)
	m.updateTableRows()
	if m.table.Cursor() >= len(m.images) {
		t.Errorf("cursor %d out of bounds for %d images", m.table.Cursor(), len(m.images))
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

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{1048576, "1.0MB"},
	}
	for _, tt := range tests {
		result := formatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestImageRef(t *testing.T) {
	tests := []struct {
		repo     string
		tag      string
		expected string
	}{
		{"nginx", "latest", "nginx:latest"},
		{"nginx", "<none>", "nginx"},
		{"nginx", "", "nginx"},
	}
	for _, tt := range tests {
		img := dt.Image{Repository: tt.repo, Tag: tt.tag}
		if got := img.Ref(); got != tt.expected {
			t.Errorf("Ref() for (%q, %q) = %q, want %q", tt.repo, tt.tag, got, tt.expected)
		}
	}
}

func TestImageDefaultSaveFile(t *testing.T) {
	tests := []struct {
		repo     string
		tag      string
		expected string
	}{
		{"nginx", "latest", "nginx-latest.tar"},
		{"nginx", "<none>", "nginx.tar"},
		{"", "", "..tar"},
	}
	for _, tt := range tests {
		img := dt.Image{Repository: tt.repo, Tag: tt.tag}
		got := img.DefaultSaveFile()
		if got != tt.expected {
			t.Errorf("DefaultSaveFile() for (%q, %q) = %q, want %q", tt.repo, tt.tag, got, tt.expected)
		}
	}
}
