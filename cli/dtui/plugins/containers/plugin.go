package containers

import (
	"context"
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

	logMode    bool
	detailMode bool
	statsMode  bool

	detail       *dt.ContainerDetail
	statsHistory []dt.StatsEntry
	statsCancel  func()
	statsCh      <-chan dt.StatsEntry
	statsErrCh   <-chan error

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
func (m *Module) IsRoot() bool        { return !m.logMode && !m.detailMode && !m.statsMode }

func (m *Module) Init() tea.Cmd {
	return tea.Batch(m.spinner.Start(), m.loadContainers())
}

func (m *Module) Bindings() []uix.HelpBinding {
	if m.detailMode {
		return []uix.HelpBinding{
			{Keys: []string{"↑↓"}, Desc: "滚动"},
			{Keys: []string{"esc"}, Desc: "返回"},
		}
	}
	if m.statsMode {
		return []uix.HelpBinding{
			{Keys: []string{"esc"}, Desc: "返回"},
		}
	}
	if m.logMode {
		return []uix.HelpBinding{
			{Keys: []string{"↑↓"}, Desc: "滚动"},
			{Keys: []string{"f"}, Desc: "跟随"},
			{Keys: []string{"esc"}, Desc: "返回"},
		}
	}
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"s"}, Desc: "启停"},
		{Keys: []string{"S"}, Desc: "停止"},
		{Keys: []string{"r"}, Desc: "重启"},
		{Keys: []string{"x"}, Desc: "删除"},
		{Keys: []string{"i"}, Desc: "详情"},
		{Keys: []string{"t"}, Desc: "统计"},
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
	case inspectDoneMsg:
		return m.handleInspectDone(msg)
	case statsStreamStartedMsg:
		return m.handleStatsStreamStarted(msg)
	case statsEntryMsg:
		return m.handleStatsEntry(msg)
	case statsDoneMsg:
		return m.handleStatsDone(msg)
	case uix.ConfirmMsg:
		return m.handleConfirm(msg.Button)
	case tea.KeyMsg:
		if m.detailMode {
			return m.handleDetailKey(msg)
		}
		if m.statsMode {
			return m.handleStatsKey(msg)
		}
		if m.logMode {
			return m.handleLogKey(msg)
		}
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *Module) View() string {
	if m.detailMode {
		return m.renderDetail()
	}
	if m.statsMode {
		return m.renderStats()
	}
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
	case "i":
		return m.openDetail()
	case "t":
		return m.openStats()
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

func (m *Module) openDetail() (tea.Model, tea.Cmd) {
	if len(m.containers) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.containers) {
		return m, nil
	}
	c := m.containers[idx]
	m.detailMode = true
	m.detail = nil
	m.status = fmt.Sprintf("Inspecting %s...", c.Name)
	return m, m.fetchInspect(c.ID)
}

func (m *Module) openStats() (tea.Model, tea.Cmd) {
	if len(m.containers) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.containers) {
		return m, nil
	}
	c := m.containers[idx]
	if !c.Running() {
		m.status = "Container not running"
		return m, nil
	}
	m.statsMode = true
	m.statsHistory = nil
	m.status = fmt.Sprintf("Streaming stats for %s...", c.Name)
	return m, m.startStatsStream(c.ID)
}

func (m *Module) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.detailMode = false
		m.detail = nil
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

func (m *Module) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.statsMode = false
		if m.statsCancel != nil {
			m.statsCancel()
			m.statsCancel = nil
		}
		m.statsCh = nil
		m.statsErrCh = nil
		m.statsHistory = nil
		m.status = ""
		return m, nil
	}
	return m, nil
}

func (m *Module) renderDetail() string {
	if m.detail == nil {
		var b strings.Builder
		b.WriteString(m.spinner.View() + " Loading container details...")
		panel := components.NewPanel("container detail", m.width, m.height)
		panel.Body = b.String()
		return panel.View()
	}

	d := m.detail
	var b strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))

	addRow := func(label, value string) {
		if value == "" {
			value = "-"
		}
		b.WriteString(labelStyle.Render(label+": ") + valueStyle.Render(value) + "\n")
	}

	addRow("ID", d.ID)
	addRow("Name", d.Name)
	addRow("Image", d.Image)
	addRow("Platform", d.Platform)
	addRow("Created", d.Created)
	addRow("Status", d.State.Status)
	addRow("Running", fmt.Sprintf("%v", d.State.Running))
	if d.State.ExitCode != 0 {
		addRow("Exit Code", fmt.Sprintf("%d", d.State.ExitCode))
	}
	if d.State.Error != "" {
		addRow("Error", d.State.Error)
	}
	addRow("Started At", d.State.StartedAt)
	addRow("Finished At", d.State.FinishedAt)
	addRow("Working Dir", d.WorkingDir)
	addRow("Restart Policy", d.RestartPolicy)

	if len(d.Cmd) > 0 {
		addRow("Command", strings.Join(d.Cmd, " "))
	}
	if len(d.Entrypoint) > 0 {
		addRow("Entrypoint", strings.Join(d.Entrypoint, " "))
	}

	if len(d.Mounts) > 0 {
		b.WriteString("\n" + labelStyle.Render("Mounts:") + "\n")
		for _, mt := range d.Mounts {
			mode := mt.Mode
			if mode == "" {
				mode = "rw"
			}
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s → %s [%s/%s]", mt.Source, mt.Destination, mt.Type, mode)) + "\n")
		}
	}

	if len(d.Network) > 0 {
		b.WriteString("\n" + labelStyle.Render("Networks:") + "\n")
		for _, n := range d.Network {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s: %s (gw: %s)", n.Name, n.IPAddress, n.Gateway)) + "\n")
		}
	}

	if len(d.Ports) > 0 {
		b.WriteString("\n" + labelStyle.Render("Ports:") + "\n")
		for _, p := range d.Ports {
			host := p.HostIP
			if host == "" {
				host = "0.0.0.0"
			}
			if p.HostPort != "" {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  %s:%s → %s/%s", host, p.HostPort, p.ContainerPort, p.Protocol)) + "\n")
			} else {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  %s/%s", p.ContainerPort, p.Protocol)) + "\n")
			}
		}
	}

	if len(d.Env) > 0 {
		b.WriteString("\n" + labelStyle.Render("Environment:") + "\n")
		show := d.Env
		if len(show) > 15 {
			show = show[:15]
		}
		for _, e := range show {
			b.WriteString(dimStyle.Render("  "+theme.Truncate(e, m.width-10)) + "\n")
		}
		if len(d.Env) > 15 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", len(d.Env)-15)) + "\n")
		}
	}

	b.WriteString("\n" + dimStyle.Render("esc/q back"))
	panel := components.NewPanel("container detail", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderStats() string {
	if len(m.statsHistory) == 0 {
		var b strings.Builder
		b.WriteString(m.spinner.View() + " Waiting for stats data...")
		panel := components.NewPanel("container stats", m.width, m.height)
		panel.Body = b.String()
		return panel.View()
	}

	latest := m.statsHistory[len(m.statsHistory)-1]
	var b strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorGreen))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorRed))

	addRow := func(label, value string) {
		b.WriteString(labelStyle.Render(label+": ") + valueStyle.Render(value) + "\n")
	}

	cpuColor := greenStyle
	if latest.CPUPercent > 80 {
		cpuColor = redStyle
	} else if latest.CPUPercent > 50 {
		cpuColor = yellowStyle
	}
	addRow("CPU", cpuColor.Render(fmt.Sprintf("%.1f%%", latest.CPUPercent)))

	memColor := greenStyle
	if latest.MemPercent > 80 {
		memColor = redStyle
	} else if latest.MemPercent > 50 {
		memColor = yellowStyle
	}
	addRow("Memory", memColor.Render(fmt.Sprintf("%.1f%% (%s / %s)",
		latest.MemPercent,
		formatBytes(latest.MemUsage),
		formatBytes(latest.MemLimit))))

	addRow("Network RX", formatBytes(latest.NetRx))
	addRow("Network TX", formatBytes(latest.NetTx))
	addRow("Block Read", formatBytes(latest.BlockRead))
	addRow("Block Write", formatBytes(latest.BlockWrite))
	addRow("PIDs", fmt.Sprintf("%d", latest.PIDs))

	if len(m.statsHistory) > 1 {
		b.WriteString("\n" + labelStyle.Render("Recent History:") + "\n")
		show := m.statsHistory
		if len(show) > 10 {
			show = show[len(show)-10:]
		}
		for i := len(show) - 1; i >= 0; i-- {
			entry := show[i]
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s  CPU %5.1f%%  MEM %5.1f%%",
				entry.Timestamp.Format("15:04:05"),
				entry.CPUPercent,
				entry.MemPercent)) + "\n")
		}
	}

	b.WriteString("\n" + dimStyle.Render("esc/q stop and back"))
	panel := components.NewPanel("container stats", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) handleInspectDone(msg inspectDoneMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	if msg.err != nil {
		m.detailMode = false
		m.state = components.StateError
		m.status = "Inspect failed: " + msg.err.Error()
		return m, nil
	}
	m.detail = msg.detail
	m.status = ""
	return m, nil
}

func (m *Module) handleStatsEntry(msg statsEntryMsg) (tea.Model, tea.Cmd) {
	m.statsHistory = append(m.statsHistory, msg.entry)
	if len(m.statsHistory) > 60 {
		m.statsHistory = m.statsHistory[len(m.statsHistory)-60:]
	}
	m.status = ""
	if m.statsCh != nil {
		ctx, cancel := context.WithCancel(context.Background())
		m.statsCancel = cancel
		return m, readStatsStream(ctx, m.statsCh, m.statsErrCh)
	}
	return m, nil
}

func (m *Module) handleStatsDone(msg statsDoneMsg) (tea.Model, tea.Cmd) {
	m.statsCancel = nil
	m.statsCh = nil
	m.statsErrCh = nil
	if msg.err != nil {
		m.status = "Stats stream ended: " + msg.err.Error()
	}
	return m, nil
}

func (m *Module) handleStatsStreamStarted(msg statsStreamStartedMsg) (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.statsCancel = cancel
	m.statsCh = msg.statsCh
	m.statsErrCh = msg.errCh
	return m, readStatsStream(ctx, msg.statsCh, msg.errCh)
}

func (m *Module) fetchInspect(id string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return inspectDoneMsg{err: m.clientErr}
		}
		detail, err := client.InspectContainer(id)
		return inspectDoneMsg{detail: detail, err: err}
	}
}

func (m *Module) startStatsStream(id string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			return statsDoneMsg{err: m.clientErr}
		}
		statsCh, errCh := client.StreamStats(id)
		return statsStreamStartedMsg{statsCh: statsCh, errCh: errCh}
	}
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

type inspectDoneMsg struct {
	detail *dt.ContainerDetail
	err    error
}

type statsStreamStartedMsg struct {
	statsCh <-chan dt.StatsEntry
	errCh   <-chan error
}

type statsEntryMsg struct {
	entry dt.StatsEntry
}

type statsDoneMsg struct {
	err error
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

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func readStatsStream(ctx context.Context, statsCh <-chan dt.StatsEntry, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case entry, ok := <-statsCh:
			if !ok {
				return statsDoneMsg{}
			}
			return statsEntryMsg{entry: entry}
		case err, ok := <-errCh:
			if !ok {
				return statsDoneMsg{}
			}
			return statsDoneMsg{err: err}
		case <-ctx.Done():
			return statsDoneMsg{}
		}
	}
}
