package containers

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

// Module manages Docker containers with the new uix shell contract.
type Module struct {
	width  int
	height int

	client    *dt.Client
	clientErr error
	containers []dt.Container

	table   components.Table
	spinner components.Spinner
	log     components.LogViewer
	state   components.StateKind
	status  string

	logMode bool

	pendingRemoveID string
}

// New creates a new containers module. Docker client is initialized lazily on Init().
func New() *Module {
	cols := []components.Column{
		{Title: "Name", Width: 18},
		{Title: "Image", Width: 22},
		{Title: "Status", Width: 16},
		{Title: "Ports", Width: 22},
	}
	t := components.NewTable(cols, nil, 80)
	sp := components.NewSpinner()
	lv := components.NewLogViewer(80, 20)
	return &Module{
		width:   80,
		height:  20,
		table:   t,
		spinner: sp,
		log:     lv,
		state:   components.StateLoading,
		status:  "connecting...",
	}
}

func (m *Module) Name() string        { return "containers" }
func (m *Module) Description() string { return "Manage Docker containers" }
func (m *Module) Aliases() []string   { return []string{"ctr"} }
func (m *Module) IsRoot() bool        { return !m.logMode }

func (m *Module) Init() tea.Cmd {
	return tea.Batch(m.spinner.Start(), m.loadContainers())
}

func (m *Module) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"s"}, Desc: "启停"},
		{Keys: []string{"S"}, Desc: "停止"},
		{Keys: []string{"r"}, Desc: "重启"},
		{Keys: []string{"x"}, Desc: "删除"},
		{Keys: []string{"l"}, Desc: "日志"},
		{Keys: []string{"R"}, Desc: "刷新"},
	}
}

func (m *Module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case containersLoadedMsg:
		return m.handleContainersLoaded(msg)
	case actionResultMsg:
		return m.handleActionResult(msg)
	case logDoneMsg:
		return m.handleLogDone(msg)
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
		return components.LogHeader("container logs", m.log.IsFollowing()) + "\n" + m.log.View()
	}

	if m.state == components.StateLoading && len(m.containers) == 0 {
		return m.renderLoading()
	}
	if m.state == components.StateError && len(m.containers) == 0 {
		return m.renderError()
	}
	if m.state == components.StateEmpty || len(m.containers) == 0 {
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
	b.WriteString(m.spinner.View() + " Loading containers...")
	if m.status != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(m.status))
	}
	panel := components.NewPanel("containers", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderError() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateError, "Failed to load containers", m.status, m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("R retry | esc back"))
	panel := components.NewPanel("containers", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderEmpty() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateEmpty, "No containers", "No Docker containers found. Run a container to get started.", m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("R refresh | esc back"))
	panel := components.NewPanel("containers", m.width, m.height)
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
	panel := components.NewPanel("containers", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

// --- Message handlers ---

func (m *Module) handleContainersLoaded(msg containersLoadedMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = msg.err.Error()
		return m, nil
	}
	m.containers = msg.containers
	if len(m.containers) == 0 {
		m.state = components.StateEmpty
	} else {
		m.state = components.StateSuccess
	}
	m.status = msg.status
	m.updateTableRows()
	return m, nil
}

func (m *Module) handleActionResult(msg actionResultMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.state = components.StateError
		m.status = "Action failed: " + msg.err.Error()
		return m, nil
	}
	m.containers = msg.containers
	m.state = components.StateSuccess
	m.status = msg.status
	m.updateTableRows()
	return m, nil
}

func (m *Module) handleLogDone(msg logDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.log.AppendLine("Error: " + msg.err.Error())
	} else {
		m.log.SetLines(msg.lines)
	}
	return m, nil
}

func (m *Module) handleConfirm(button string) (tea.Model, tea.Cmd) {
	id := m.pendingRemoveID
	m.pendingRemoveID = ""
	if button != "Force Delete" || id == "" {
		m.status = "cancelled"
		return m, nil
	}
	m.status = "Deleting container..."
	m.state = components.StateLoading
	return m, tea.Batch(m.spinner.Start(), m.removeContainer(id))
}

func (m *Module) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.table.SetCursor(max(0, m.table.Cursor()-1))
		return m, nil
	case "down", "j":
		m.table.SetCursor(min(len(m.containers)-1, m.table.Cursor()+1))
		return m, nil
	case "R":
		m.status = "refreshing..."
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.loadContainers())
	case "s":
		return m.toggleContainer()
	case "S":
		return m.stopContainer()
	case "r":
		return m.restartContainer()
	case "x":
		return m.confirmRemove()
	case "l":
		return m.openLogs()
	}
	return m, nil
}

func (m *Module) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.logMode = false
		m.SetSize(m.width, m.height)
		return m, tea.Batch(m.spinner.Start(), m.loadContainers())
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

func (m *Module) toggleContainer() (tea.Model, tea.Cmd) {
	if len(m.containers) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.containers) {
		return m, nil
	}
	c := m.containers[idx]
	if c.State == "running" {
		return m, m.stopContainerByID(c.ID)
	}
	return m, m.startContainer(c.ID)
}

func (m *Module) stopContainer() (tea.Model, tea.Cmd) {
	if len(m.containers) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.containers) {
		return m, nil
	}
	c := m.containers[idx]
	if c.State != "running" {
		return m, nil
	}
	return m, m.stopContainerByID(c.ID)
}

func (m *Module) restartContainer() (tea.Model, tea.Cmd) {
	if len(m.containers) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.containers) {
		return m, nil
	}
	c := m.containers[idx]
	m.status = fmt.Sprintf("Restarting %s...", c.Name)
	return m, m.runAction(func() error {
		return m.client.RestartContainer(c.ID)
	})
}

func (m *Module) confirmRemove() (tea.Model, tea.Cmd) {
	if len(m.containers) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.containers) {
		return m, nil
	}
	c := m.containers[idx]
	m.pendingRemoveID = c.ID
	return m, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Delete",
			Message: fmt.Sprintf("Delete container %s (%s)?", c.Name, theme.Truncate(c.ID, 12)),
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Force Delete", Key: "enter"},
			},
		}
	}
}

func (m *Module) openLogs() (tea.Model, tea.Cmd) {
	if len(m.containers) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.containers) {
		return m, nil
	}
	c := m.containers[idx]
	m.logMode = true
	m.SetSize(m.width, m.height)
	return m, m.fetchLogs(c.ID)
}

func (m *Module) updateTableRows() {
	rows := make([]components.Row, len(m.containers))
	for i, c := range m.containers {
		status := statusIcon(c.State) + " " + c.Status
		rows[i] = components.Row{c.Name, theme.Truncate(c.Image, 20), status, c.Ports}
	}
	m.table.SetRows(rows)
	cursor := m.table.Cursor()
	if cursor >= len(m.containers) {
		m.table.SetCursor(max(0, len(m.containers)-1))
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

func (m *Module) loadContainers() tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return containersLoadedMsg{err: m.clientErr}
		}
		containers, err := client.ListContainers("")
		return containersLoadedMsg{containers: containers, err: err}
	}
}

func (m *Module) startContainer(id string) tea.Cmd {
	return m.runAction(func() error { return m.client.StartContainer(id) })
}

func (m *Module) stopContainerByID(id string) tea.Cmd {
	return m.runAction(func() error { return m.client.StopContainer(id) })
}

func (m *Module) removeContainer(id string) tea.Cmd {
	return m.runAction(func() error { return m.client.RemoveContainer(id, true) })
}

func (m *Module) runAction(fn func() error) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return actionResultMsg{err: m.clientErr}
		}
		err := fn()
		containers, listErr := client.ListContainers("")
		if err == nil {
			err = listErr
		}
		return actionResultMsg{containers: containers, err: err, status: "Action complete"}
	}
}

func (m *Module) fetchLogs(id string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return logDoneMsg{err: m.clientErr}
		}
		lines, err := client.FetchLogs(id, dt.LogOptions{Tail: "200"})
		if err != nil {
			return logDoneMsg{err: err}
		}
		return logDoneMsg{lines: lines}
	}
}

// --- Messages ---

type containersLoadedMsg struct {
	containers []dt.Container
	err        error
	status     string
}

type actionResultMsg struct {
	containers []dt.Container
	err        error
	status     string
}

type logDoneMsg struct {
	lines []string
	err   error
}

// --- Helpers ---

func statusIcon(state string) string {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorGreen)).Render("▲")
	case "exited":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorRed)).Render("▼")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render("●")
	}
}
