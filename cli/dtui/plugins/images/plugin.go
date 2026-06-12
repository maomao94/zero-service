package images

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	dt "zero-service/cli/dtui/internal/docker"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

// Module manages Docker images with the new uix shell contract.
type Module struct {
	width  int
	height int

	client    *dt.Client
	clientErr error
	images    []dt.Image

	table   components.Table
	spinner components.Spinner
	log     components.LogViewer
	state   components.StateKind
	status  string

	historyMode bool
	tagMode     bool
	saveMode    bool

	history      []dt.ImageHistoryEntry
	tagInput     textinput.Model
	saveInput    textinput.Model
	pendingRef   string

	pendingRemoveRef string
	pendingPrune     bool
}

// New creates a new images module. Docker client is initialized lazily on Init().
func New() *Module {
	cols := []components.Column{
		{Title: "Repository", Width: 30},
		{Title: "Tag", Width: 12},
		{Title: "ID", Width: 14},
		{Title: "Size", Width: 10},
	}
	t := components.NewTable(cols, nil, 80)
	sp := components.NewSpinner()
	lv := components.NewLogViewer(80, 20)

	tagIn := textinput.New()
	tagIn.Focus()
	tagIn.Placeholder = "new-tag"
	tagIn.CharLimit = 128

	saveIn := textinput.New()
	saveIn.Focus()
	saveIn.Placeholder = "output.tar"
	saveIn.CharLimit = 256

	return &Module{
		width:     80,
		height:    20,
		table:     t,
		spinner:   sp,
		log:       lv,
		tagInput:  tagIn,
		saveInput: saveIn,
		state:     components.StateLoading,
		status:    "connecting...",
	}
}

func (m *Module) Name() string        { return "images" }
func (m *Module) Description() string { return "Manage Docker images" }
func (m *Module) Aliases() []string   { return []string{"img"} }
func (m *Module) IsRoot() bool        { return !m.historyMode && !m.tagMode && !m.saveMode }

func (m *Module) Init() tea.Cmd {
	return tea.Batch(m.spinner.Start(), m.loadImages())
}

func (m *Module) Bindings() []uix.HelpBinding {
	if m.historyMode {
		return []uix.HelpBinding{
			{Keys: []string{"↑↓"}, Desc: "滚动"},
			{Keys: []string{"esc"}, Desc: "返回"},
		}
	}
	if m.tagMode || m.saveMode {
		return []uix.HelpBinding{
			{Keys: []string{"enter"}, Desc: "确认"},
			{Keys: []string{"esc"}, Desc: "取消"},
		}
	}
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"h"}, Desc: "历史"},
		{Keys: []string{"T"}, Desc: "标签"},
		{Keys: []string{"e"}, Desc: "导出"},
		{Keys: []string{"x"}, Desc: "删除"},
		{Keys: []string{"p"}, Desc: "清理"},
		{Keys: []string{"r"}, Desc: "刷新"},
	}
}

func (m *Module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imagesLoadedMsg:
		return m.handleImagesLoaded(msg)
	case removeResultMsg:
		return m.handleRemoveResult(msg)
	case pruneResultMsg:
		return m.handlePruneResult(msg)
	case historyDoneMsg:
		return m.handleHistoryDone(msg)
	case tagDoneMsg:
		return m.handleTagDone(msg)
	case saveDoneMsg:
		return m.handleSaveDone(msg)
	case uix.ConfirmMsg:
		return m.handleConfirm(msg.Button)
	case tea.KeyMsg:
		if m.historyMode {
			return m.handleHistoryKey(msg)
		}
		if m.tagMode {
			return m.handleTagKey(msg)
		}
		if m.saveMode {
			return m.handleSaveKey(msg)
		}
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *Module) View() string {
	if m.historyMode {
		return m.renderHistory()
	}
	if m.tagMode {
		return m.renderTagForm()
	}
	if m.saveMode {
		return m.renderSaveForm()
	}

	if m.state == components.StateLoading && len(m.images) == 0 {
		return m.renderLoading()
	}
	if m.state == components.StateError && len(m.images) == 0 {
		return m.renderError()
	}
	if m.state == components.StateEmpty || len(m.images) == 0 {
		return m.renderEmpty()
	}
	return m.renderTable()
}

func (m *Module) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	m.width = width
	m.height = height
	m.table.SetSize(max(20, width-6), max(5, height-6))
}

// --- Rendering ---

func (m *Module) renderLoading() string {
	var b strings.Builder
	b.WriteString(m.spinner.View() + " Loading images...")
	if m.status != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(m.status))
	}
	panel := components.NewPanel("images", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderError() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateError, "Failed to load images", m.status, m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("r retry | esc back"))
	panel := components.NewPanel("images", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderEmpty() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateEmpty, "No images", "No Docker images found. Pull an image or build one to get started.", m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("r refresh | esc back"))
	panel := components.NewPanel("images", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderTable() string {
	var b strings.Builder
	if m.status != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render(m.status))
		b.WriteString("\n\n")
	}
	b.WriteString(m.table.View())
	panel := components.NewPanel("images", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

// --- Message handlers ---

func (m *Module) handleImagesLoaded(msg imagesLoadedMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = msg.err.Error()
		return m, nil
	}
	m.images = msg.images
	if len(m.images) == 0 {
		m.state = components.StateEmpty
	} else {
		m.state = components.StateSuccess
	}
	m.updateTableRows()
	return m, nil
}

func (m *Module) handleRemoveResult(msg removeResultMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = "Remove failed: " + msg.err.Error()
		return m, nil
	}
	m.images = msg.images
	m.state = components.StateSuccess
	m.status = "Image removed"
	m.updateTableRows()
	return m, nil
}

func (m *Module) handlePruneResult(msg pruneResultMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = "Prune failed: " + msg.err.Error()
		return m, nil
	}
	m.images = msg.images
	m.state = components.StateSuccess
	if msg.reclaimed > 0 {
		m.status = fmt.Sprintf("Pruned images, reclaimed %d bytes", msg.reclaimed)
	} else {
		m.status = "No dangling images to prune"
	}
	m.updateTableRows()
	return m, nil
}

func (m *Module) handleConfirm(button string) (tea.Model, tea.Cmd) {
	// Handle prune confirmation
	if m.pendingPrune {
		m.pendingPrune = false
		if button != "Prune" {
			m.status = "cancelled"
			return m, nil
		}
		m.status = "pruning..."
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.pruneImages())
	}
	// Handle remove confirmation
	ref := m.pendingRemoveRef
	m.pendingRemoveRef = ""
	if button != "Remove" || ref == "" {
		m.status = "cancelled"
		return m, nil
	}
	m.status = "Removing image..."
	m.state = components.StateLoading
	return m, tea.Batch(m.spinner.Start(), m.removeImage(ref))
}

func (m *Module) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.table.SetCursor(max(0, m.table.Cursor()-1))
		return m, nil
	case "down", "j":
		m.table.SetCursor(min(len(m.images)-1, m.table.Cursor()+1))
		return m, nil
	case "r":
		m.status = "refreshing..."
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.loadImages())
	case "h":
		return m.openHistory()
	case "T":
		return m.openTagForm()
	case "e":
		return m.openSaveForm()
	case "x":
		return m.confirmRemove()
	case "p":
		return m.confirmPrune()
	}
	return m, nil
}

// --- Actions ---

func (m *Module) confirmRemove() (tea.Model, tea.Cmd) {
	if len(m.images) == 0 {
		return m, nil
	}
	row, ok := m.table.Selected()
	if !ok {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.images) {
		return m, nil
	}
	img := m.images[idx]
	m.pendingRemoveRef = img.Ref()
	return m, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Remove",
			Message: fmt.Sprintf("Remove image %s?", row[0]),
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Remove", Key: "enter"},
			},
		}
	}
}

func (m *Module) confirmPrune() (tea.Model, tea.Cmd) {
	if len(m.images) == 0 {
		return m, nil
	}
	m.pendingPrune = true
	return m, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Prune",
			Message: "Prune all dangling images? This will remove unused images and reclaim disk space.",
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Prune", Key: "enter"},
			},
		}
	}
}

func (m *Module) openHistory() (tea.Model, tea.Cmd) {
	if len(m.images) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.images) {
		return m, nil
	}
	img := m.images[idx]
	m.historyMode = true
	m.history = nil
	m.status = fmt.Sprintf("Loading history for %s...", img.Ref())
	return m, m.fetchHistory(img.Ref())
}

func (m *Module) openTagForm() (tea.Model, tea.Cmd) {
	if len(m.images) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.images) {
		return m, nil
	}
	img := m.images[idx]
	m.tagMode = true
	m.pendingRef = img.Ref()
	m.tagInput.SetValue("")
	m.tagInput.Focus()
	m.status = ""
	return m, nil
}

func (m *Module) openSaveForm() (tea.Model, tea.Cmd) {
	if len(m.images) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.images) {
		return m, nil
	}
	img := m.images[idx]
	m.saveMode = true
	m.pendingRef = img.Ref()
	m.saveInput.SetValue(img.DefaultSaveFile())
	m.saveInput.Focus()
	m.status = ""
	return m, nil
}

func (m *Module) handleHistoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.historyMode = false
		m.history = nil
		m.status = ""
		return m, nil
	case "up", "k":
		m.log.ScrollUp()
	case "down", "j":
		m.log.ScrollDown()
	case "pgup":
		m.log.PageUp()
	case "pgdown":
		m.log.PageDown()
	}
	return m, nil
}

func (m *Module) handleTagKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.tagMode = false
		m.pendingRef = ""
		m.status = "cancelled"
		return m, nil
	case "enter":
		tag := strings.TrimSpace(m.tagInput.Value())
		if tag == "" {
			m.status = "Tag cannot be empty"
			return m, nil
		}
		ref := m.pendingRef
		m.tagMode = false
		m.pendingRef = ""
		m.status = fmt.Sprintf("Tagging %s → %s...", ref, tag)
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.tagImage(ref, tag))
	}
	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	return m, cmd
}

func (m *Module) handleSaveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.saveMode = false
		m.pendingRef = ""
		m.status = "cancelled"
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.saveInput.Value())
		if path == "" {
			m.status = "Path cannot be empty"
			return m, nil
		}
		ref := m.pendingRef
		m.saveMode = false
		m.pendingRef = ""
		m.status = fmt.Sprintf("Saving %s to %s...", ref, path)
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.saveImageCmd(ref, path))
	}
	var cmd tea.Cmd
	m.saveInput, cmd = m.saveInput.Update(msg)
	return m, cmd
}

func (m *Module) renderHistory() string {
	if len(m.history) == 0 {
		var b strings.Builder
		b.WriteString(m.spinner.View() + " Loading image history...")
		panel := components.NewPanel("image history", m.width, m.height)
		panel.Body = b.String()
		return panel.View()
	}

	panel := components.NewPanel("image history", m.width, m.height)
	panel.Body = m.log.View()
	panel.Footer = "esc/q back"
	return panel.View()
}

func (m *Module) renderTagForm() string {
	var b strings.Builder
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true)

	b.WriteString(labelStyle.Render("Tag Image") + "\n\n")
	b.WriteString(fmt.Sprintf("Source: %s\n\n", m.pendingRef))
	b.WriteString("New tag (repository:tag):\n")
	b.WriteString(m.tagInput.View() + "\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("enter confirm | esc cancel"))

	panel := components.NewPanel("tag image", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderSaveForm() string {
	var b strings.Builder
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true)

	b.WriteString(labelStyle.Render("Export Image") + "\n\n")
	b.WriteString(fmt.Sprintf("Source: %s\n\n", m.pendingRef))
	b.WriteString("Output path:\n")
	b.WriteString(m.saveInput.View() + "\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("enter confirm | esc cancel"))

	panel := components.NewPanel("export image", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) handleHistoryDone(msg historyDoneMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.historyMode = false
		m.state = components.StateError
		m.status = "History failed: " + msg.err.Error()
		return m, nil
	}
	m.history = msg.entries
	var lines []string
	for i, entry := range m.history {
		sizeStr := formatSize(entry.Size)
		line := fmt.Sprintf("#%-3d  %s  %s", i+1, theme.Truncate(entry.ID, 12), sizeStr)
		lines = append(lines, line)
		if entry.CreatedBy != "" {
			lines = append(lines, "  "+entry.CreatedBy)
		}
	}
	m.log.SetLines(lines)
	m.status = ""
	return m, nil
}

func (m *Module) handleTagDone(msg tagDoneMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = "Tag failed: " + msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Tagged as %s", msg.tag)
	return m, tea.Batch(m.spinner.Start(), m.loadImages())
}

func (m *Module) handleSaveDone(msg saveDoneMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = "Export failed: " + msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Image saved to %s", msg.path)
	return m, nil
}

func (m *Module) fetchHistory(ref string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return historyDoneMsg{err: m.clientErr}
		}
		entries, err := client.ImageHistory(ref)
		return historyDoneMsg{entries: entries, err: err}
	}
}

func (m *Module) tagImage(source, tag string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return tagDoneMsg{err: m.clientErr}
		}
		err := client.TagImage(source, tag)
		return tagDoneMsg{tag: tag, err: err}
	}
}

func (m *Module) saveImageCmd(ref, path string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return saveDoneMsg{err: m.clientErr}
		}
		err := client.SaveImage(ref, path)
		return saveDoneMsg{path: path, err: err}
	}
}

func (m *Module) updateTableRows() {
	rows := make([]components.Row, len(m.images))
	for i, img := range m.images {
		rows[i] = components.Row{theme.Truncate(img.Repository, 30), img.Tag, theme.Truncate(img.ID, 12), img.Size}
	}
	m.table.SetRows(rows)
	cursor := m.table.Cursor()
	if cursor >= len(m.images) {
		m.table.SetCursor(max(0, len(m.images)-1))
	}
}

// --- Async commands ---

func (m *Module) ensureClient() *dt.Client {
	if m.client != nil || m.clientErr != nil {
		return m.client
	}
	c, err := dt.NewClient()
	if err != nil {
		m.clientErr = err
		return nil
	}
	m.client = c
	return m.client
}

func (m *Module) loadImages() tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return imagesLoadedMsg{err: m.clientErr}
		}
		imgs, err := client.ListImages("")
		return imagesLoadedMsg{images: imgs, err: err}
	}
}

func (m *Module) removeImage(ref string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return removeResultMsg{err: m.clientErr}
		}
		if err := client.RemoveImage(ref, true); err != nil {
			return removeResultMsg{err: err}
		}
		imgs, err := client.ListImages("")
		return removeResultMsg{images: imgs, err: err}
	}
}

func (m *Module) pruneImages() tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return pruneResultMsg{err: m.clientErr}
		}
		reclaimed, err := client.PruneImages()
		if err != nil {
			return pruneResultMsg{err: err}
		}
		imgs, err := client.ListImages("")
		return pruneResultMsg{images: imgs, err: err, reclaimed: reclaimed}
	}
}

// --- Messages ---

type imagesLoadedMsg struct {
	images []dt.Image
	err    error
}

type removeResultMsg struct {
	images []dt.Image
	err    error
}

type pruneResultMsg struct {
	images    []dt.Image
	err       error
	reclaimed uint64
}

type historyDoneMsg struct {
	entries []dt.ImageHistoryEntry
	err     error
}

type tagDoneMsg struct {
	tag string
	err error
}

type saveDoneMsg struct {
	path string
	err  error
}

// --- Helpers ---

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
