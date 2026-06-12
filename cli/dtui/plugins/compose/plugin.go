package compose

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/dtui/internal/config"
	dt "zero-service/cli/dtui/internal/docker"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

// Module manages Docker Compose projects with the new uix shell contract.
type Module struct {
	width  int
	height int

	cfg     config.Config
	entries []composeEntry

	table   components.Table
	spinner components.Spinner
	log     components.LogViewer
	state   components.StateKind
	status  string

	logMode      bool
	pendingAction string
}

type composeEntry struct {
	Name string
	Path string
}

// New creates a new compose module. Config is passed in; no Docker daemon required at startup.
func New(cfg config.Config) *Module {
	cols := []components.Column{
		{Title: "Name", Width: 25},
		{Title: "Path", Width: 50},
	}
	t := components.NewTable(cols, nil, 80)
	sp := components.NewSpinner()
	log := components.NewLogViewer(80, 10)
	return &Module{
		width:   80,
		height:  20,
		cfg:     cfg,
		table:   t,
		spinner: sp,
		log:     log,
		state:   components.StateLoading,
		status:  "loading...",
	}
}

func (m *Module) Name() string        { return "compose" }
func (m *Module) Description() string { return "Manage Docker Compose projects" }
func (m *Module) Aliases() []string   { return []string{"dc"} }
func (m *Module) IsRoot() bool        { return !m.logMode }

func (m *Module) Init() tea.Cmd {
	return tea.Batch(m.spinner.Start(), m.loadEntries())
}

func (m *Module) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"u"}, Desc: "up"},
		{Keys: []string{"d"}, Desc: "down"},
		{Keys: []string{"r"}, Desc: "刷新"},
		{Keys: []string{"l"}, Desc: "日志"},
	}
}

func (m *Module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case composeLoadedMsg:
		return m.handleComposeLoaded(msg)
	case actionResultMsg:
		return m.handleActionResult(msg)
	case uix.ConfirmMsg:
		return m.handleConfirm(msg.Button)
	case tea.KeyMsg:
		if m.logMode {
			return m.handleLogKey(msg)
		}
		return m.handleKey(msg)
	}

	// Forward spinner ticks.
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *Module) View() string {
	if m.logMode {
		return components.LogHeader("compose output", m.log.IsFollowing()) + "\n" + m.log.View()
	}

	if m.state == components.StateLoading && len(m.entries) == 0 {
		return m.renderLoading()
	}
	if m.state == components.StateError && len(m.entries) == 0 {
		return m.renderError()
	}
	if m.state == components.StateEmpty || len(m.entries) == 0 {
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
	logWidth := width - 8
	if logWidth < 20 {
		logWidth = 20
	}
	logHeight := height - 4
	if logHeight < 5 {
		logHeight = 5
	}
	if m.logMode {
		logHeight = height - 2
		if logHeight < 5 {
			logHeight = 5
		}
	}
	m.log.SetSize(logWidth, logHeight)
}

// --- Rendering ---

func (m *Module) renderLoading() string {
	var b strings.Builder
	b.WriteString(m.spinner.View() + " Loading compose projects...")
	if m.status != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(m.status))
	}
	panel := components.NewPanel("compose", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderError() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateError, "Failed to load compose projects", m.status, m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("r retry | esc back"))
	panel := components.NewPanel("compose", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderEmpty() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateEmpty, "No compose projects", "No Docker Compose projects configured. Add compose directories to your config.", m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("r refresh | esc back"))
	panel := components.NewPanel("compose", m.width, m.height)
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
	panel := components.NewPanel("compose", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

// --- Message handlers ---

func (m *Module) handleComposeLoaded(msg composeLoadedMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = msg.err.Error()
		return m, nil
	}
	m.entries = msg.entries
	if len(m.entries) == 0 {
		m.state = components.StateEmpty
	} else {
		m.state = components.StateSuccess
	}
	m.updateTableRows()
	return m, nil
}

func (m *Module) handleActionResult(msg actionResultMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = "Action failed: " + msg.err.Error()
		m.log.AppendLine("error: " + msg.err.Error())
		return m, nil
	}
	m.state = components.StateSuccess
	m.status = "Action completed"
	m.log.AppendLine("output:\n" + msg.output)
	return m, nil
}

func (m *Module) handleConfirm(button string) (tea.Model, tea.Cmd) {
	action := m.pendingAction
	m.pendingAction = ""
	if button == "Cancel" || action == "" {
		m.status = "cancelled"
		return m, nil
	}

	m.status = "Executing..."
	m.state = components.StateLoading
	m.log.AppendLine("starting " + action + "...")

	switch action {
	case "up":
		return m, tea.Batch(m.spinner.Start(), m.runComposeUp())
	case "down":
		return m, tea.Batch(m.spinner.Start(), m.runComposeDown())
	}
	return m, nil
}

func (m *Module) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.table.SetCursor(max(0, m.table.Cursor()-1))
		return m, nil
	case "down", "j":
		m.table.SetCursor(min(len(m.entries)-1, m.table.Cursor()+1))
		return m, nil
	case "r":
		m.status = "refreshing..."
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.loadEntries())
	case "u":
		return m.confirmUp()
	case "d":
		return m.confirmDown()
	case "l":
		m.logMode = true
		m.SetSize(m.width, m.height)
		return m, nil
	}
	return m, nil
}

func (m *Module) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.logMode = false
		m.SetSize(m.width, m.height)
	case "up", "k":
		m.log.ScrollUp()
	case "down", "j":
		m.log.ScrollDown()
	case "pgup":
		m.log.PageUp()
	case "pgdown":
		m.log.PageDown()
	case "g":
		m.log.GotoTop()
	case "G":
		m.log.GotoBottom()
	case "f":
		m.log.ToggleFollow()
	}
	return m, nil
}

// --- Actions ---

func (m *Module) confirmUp() (tea.Model, tea.Cmd) {
	if len(m.entries) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.entries) {
		return m, nil
	}
	e := m.entries[idx]
	m.pendingAction = "up"
	return m, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Compose Up",
			Message: fmt.Sprintf("docker compose -f %s up -d", e.Path),
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Execute", Key: "enter"},
			},
		}
	}
}

func (m *Module) confirmDown() (tea.Model, tea.Cmd) {
	if len(m.entries) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.entries) {
		return m, nil
	}
	e := m.entries[idx]
	m.pendingAction = "down"
	return m, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Compose Down",
			Message: fmt.Sprintf("docker compose -f %s down", e.Path),
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Execute", Key: "enter"},
			},
		}
	}
}

func (m *Module) updateTableRows() {
	rows := make([]components.Row, len(m.entries))
	for i, e := range m.entries {
		rows[i] = components.Row{e.Name, theme.Truncate(e.Path, 50)}
	}
	m.table.SetRows(rows)
	cursor := m.table.Cursor()
	if cursor >= len(m.entries) {
		m.table.SetCursor(max(0, len(m.entries)-1))
	}
}

// --- Async commands ---

func (m *Module) loadEntries() tea.Cmd {
	return func() tea.Msg {
		entries := make([]composeEntry, len(m.cfg.ComposeDirs))
		for i, d := range m.cfg.ComposeDirs {
			entries[i] = composeEntry{Name: d.Name, Path: d.Path}
		}
		return composeLoadedMsg{entries: entries}
	}
}

func (m *Module) runComposeUp() tea.Cmd {
	if len(m.entries) == 0 {
		return nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.entries) {
		return nil
	}
	e := m.entries[idx]
	return func() tea.Msg {
		out, err := dt.RunComposeUp(e.Path, "")
		return actionResultMsg{output: out, err: err}
	}
}

func (m *Module) runComposeDown() tea.Cmd {
	if len(m.entries) == 0 {
		return nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.entries) {
		return nil
	}
	e := m.entries[idx]
	return func() tea.Msg {
		out, err := dt.RunComposeDown(e.Path, "")
		return actionResultMsg{output: out, err: err}
	}
}

// --- Messages ---

type composeLoadedMsg struct {
	entries []composeEntry
	err     error
}

type actionResultMsg struct {
	output string
	err    error
}
