package images

import (
	"fmt"
	"strings"

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
	state   components.StateKind
	status  string

	pendingRemoveRef string
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
	return &Module{
		width:   80,
		height:  20,
		table:   t,
		spinner: sp,
		state:   components.StateLoading,
		status:  "connecting...",
	}
}

func (m *Module) Name() string        { return "images" }
func (m *Module) Description() string { return "Manage Docker images" }
func (m *Module) Aliases() []string   { return []string{"img"} }
func (m *Module) IsRoot() bool        { return true }

func (m *Module) Init() tea.Cmd {
	return tea.Batch(m.spinner.Start(), m.loadImages())
}

func (m *Module) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
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
	case uix.ConfirmMsg:
		return m.handleConfirm(msg.Button)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward spinner ticks.
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *Module) View() string {
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
	case "x":
		return m.confirmRemove()
	case "p":
		m.status = "pruning..."
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.pruneImages())
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
	// Find the image by matching the row data.
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
