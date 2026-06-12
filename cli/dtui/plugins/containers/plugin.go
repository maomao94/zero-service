package containers

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	dt "zero-service/cli/dtui/internal/docker"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

var (
	detailTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorAccent)).
				Bold(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color(theme.ColorBorder)).
				Padding(0, 1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorDim))

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorFg))

	panelBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.ColorBorder)).
			Padding(0, 1)
)

type Plugin struct {
	client          *dt.Client
	table           table.Model
	detail          viewport.Model
	tableW          int
	detailW         int
	height          int
	containers      []dt.Container
	cursor          int
	loading         bool
	status          string
	logViewer       components.LogViewer
	showLog         bool
	pendingRemoveID string
}

func New(client *dt.Client) *Plugin {
	cols := []table.Column{
		{Title: "Name", Width: 18},
		{Title: "Image", Width: 22},
		{Title: "Status", Width: 16},
		{Title: "Ports", Width: 22},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	t.SetStyles(containerTableStyles())

	vp := viewport.New(30, 20)

	return &Plugin{client: client, table: t, detail: vp, logViewer: components.NewLogViewer(80, 20)}
}

func (p *Plugin) Name() string        { return "containers" }
func (p *Plugin) Description() string { return "Manage Docker containers" }
func (p *Plugin) Aliases() []string   { return []string{"c", "cnt"} }
func (p *Plugin) IsRoot() bool        { return !p.showLog }
func (p *Plugin) OnActivate() tea.Cmd { return p.loadContainers() }
func (p *Plugin) OnDeactivate()       {}

func (p *Plugin) Init() tea.Cmd { return p.loadContainers() }

func (p *Plugin) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case containersLoadedMsg:
		p.loading = false
		if msg.err != nil {
			p.status = msg.err.Error()
		} else {
			p.status = msg.status
			p.containers = msg.containers
			p.updateTable()
		}
		return p, nil

	case logLineMsg:
		p.logViewer.AppendLine(msg.line)
		return p, nil

	case logDoneMsg:
		if msg.err != nil {
			p.logViewer.AppendLine("Error: " + msg.err.Error())
		} else {
			p.logViewer.SetLines(msg.lines)
		}
		return p, nil

	case uix.ConfirmMsg:
		return p.handleConfirm(msg.Button)

	case tea.KeyMsg:
		return p.handleKey(msg)
	}

	var cmd tea.Cmd
	if p.showLog {
		p.logViewer, cmd = p.logViewer.Update(msg)
		return p, cmd
	}
	p.table, cmd = p.table.Update(msg)
	return p, cmd
}

func (p *Plugin) View() string {
	if p.showLog {
		return p.logView()
	}
	status := ""
	if p.status != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render(p.status) + "\n\n"
	}
	left := panelBorder.Width(p.tableW).Render(p.table.View())
	if p.detailW < 20 {
		return status + left
	}
	right := panelBorder.Width(p.detailW).Height(p.height - 2).Render(p.buildDetail())
	return status + lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (p *Plugin) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	p.height = height
	p.tableW = width * 55 / 100
	p.detailW = width - p.tableW - 2
	if p.detailW < 20 {
		p.tableW = width
		p.detailW = 0
	}
	p.table.SetWidth(p.tableW - 4)
	p.table.SetHeight(height - 4)
	p.detail.Width = p.detailW - 4
	p.detail.Height = height - 4
	p.logViewer.SetSize(width-6, height-4)

	if p.tableW > 20 {
		cols := p.table.Columns()
		tw := p.tableW - 6
		cols[0].Width = max(10, tw*22/100)
		cols[1].Width = max(10, tw*28/100)
		cols[2].Width = max(10, tw*22/100)
		cols[3].Width = max(10, tw*28/100)
		p.table.SetColumns(cols)
	}
}

func (p *Plugin) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"s"}, Desc: "启停"},
		{Keys: []string{"R"}, Desc: "重启"},
		{Keys: []string{"x"}, Desc: "删除"},
		{Keys: []string{"l"}, Desc: "日志"},
		{Keys: []string{"r"}, Desc: "刷新"},
		{Keys: []string{"/"}, Desc: "模块"},
	}
}

func (p *Plugin) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if p.showLog {
		return p.handleLogKey(msg)
	}

	switch msg.String() {
	case "up", "k":
		p.moveCursor(-1)
	case "down", "j":
		p.moveCursor(1)
	case "r":
		return p, p.loadContainers()
	case "s":
		return p.toggleContainer()
	case "R":
		return p.restartContainer()
	case "x":
		return p.showRemoveConfirm()
	case "l":
		return p.openLogs()
	default:
		var cmd tea.Cmd
		p.table, cmd = p.table.Update(msg)
		return p, cmd
	}
	p.updateDetail()
	return p, nil
}

func (p *Plugin) moveCursor(delta int) {
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if len(p.containers) > 0 && p.cursor >= len(p.containers) {
		p.cursor = len(p.containers) - 1
	}
	p.table.SetCursor(p.cursor)
}

func (p *Plugin) updateTable() {
	rows := make([]table.Row, len(p.containers))
	for i, c := range p.containers {
		status := statusIcon(c.State) + " " + c.Status
		rows[i] = table.Row{c.Name, theme.Truncate(c.Image, 20), status, c.Ports}
	}
	p.table.SetRows(rows)
	if p.cursor >= len(p.containers) {
		p.cursor = max(0, len(p.containers)-1)
	}
	p.table.SetCursor(p.cursor)
	p.updateDetail()
}

func (p *Plugin) updateDetail() {
	p.detail.SetContent(p.buildDetail())
}

func (p *Plugin) buildDetail() string {
	if p.cursor < 0 || p.cursor >= len(p.containers) {
		return ""
	}
	c := p.containers[p.cursor]
	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(" " + theme.Truncate(c.Name, 25) + " "))
	b.WriteString("\n\n")

	fields := [][2]string{
		{"ID", c.ID},
		{"Image", c.Image},
		{"Status", c.Status},
		{"State", c.State},
		{"Ports", c.Ports},
		{"Command", c.Command},
		{"Created", c.Created},
	}
	for _, f := range fields {
		if f[1] != "" {
			b.WriteString(detailLabelStyle.Render(f[0]))
			b.WriteString("  ")
			b.WriteString(detailValueStyle.Render(theme.Truncate(f[1], 35)))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (p *Plugin) toggleContainer() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.containers) {
		return p, nil
	}
	c := p.containers[p.cursor]
	if c.State == "running" {
		return p, p.stopContainer(c.ID)
	}
	return p, p.startContainer(c.ID)
}

func (p *Plugin) restartContainer() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.containers) {
		return p, nil
	}
	c := p.containers[p.cursor]
	p.status = fmt.Sprintf("Restarting %s...", c.Name)
	return p, p.runAction(func() error {
		return p.client.RestartContainer(c.ID)
	})
}

func (p *Plugin) showRemoveConfirm() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.containers) {
		return p, nil
	}
	c := p.containers[p.cursor]
	msg := fmt.Sprintf("Delete container %s (%s)?", c.Name, shortStr(c.ID, 12))
	p.pendingRemoveID = c.ID
	return p, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Delete",
			Message: msg,
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Force Delete", Key: "enter"},
			},
		}
	}
}

func (p *Plugin) handleConfirm(button string) (tea.Model, tea.Cmd) {
	id := p.pendingRemoveID
	p.pendingRemoveID = ""
	if button != "Force Delete" || id == "" {
		return p, nil
	}
	p.status = "Deleting container..."
	return p, p.runAction(func() error { return p.client.RemoveContainer(id, true) })
}

func (p *Plugin) loadContainers() tea.Cmd {
	p.loading = true
	return func() tea.Msg {
		containers, err := p.client.ListContainers("")
		return containersLoadedMsg{containers: containers, err: err}
	}
}

func (p *Plugin) startContainer(id string) tea.Cmd {
	return p.runAction(func() error { return p.client.StartContainer(id) })
}

func (p *Plugin) stopContainer(id string) tea.Cmd {
	return p.runAction(func() error { return p.client.StopContainer(id) })
}

func (p *Plugin) runAction(fn func() error) tea.Cmd {
	return func() tea.Msg {
		err := fn()
		containers, listErr := p.client.ListContainers("")
		if err == nil {
			err = listErr
		}
		return containersLoadedMsg{containers: containers, err: err, status: "Action complete"}
	}
}

type containersLoadedMsg struct {
	containers []dt.Container
	err        error
	status     string
}

type logLineMsg struct{ line string }

type logDoneMsg struct {
	lines []string
	err   error
}

func (p *Plugin) openLogs() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.containers) {
		return p, nil
	}
	c := p.containers[p.cursor]
	p.showLog = true
	return p, p.streamLogs(c.ID)
}

func (p *Plugin) logView() string {
	return components.LogHeader("Logs", p.logViewer.IsFollowing()) + "\n" + p.logViewer.View()
}

func (p *Plugin) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		p.showLog = false
		return p, p.loadContainers()
	case "f":
		p.logViewer.ToggleFollow()
	case "up", "k":
		p.logViewer.ScrollUp()
	case "down", "j":
		p.logViewer.ScrollDown()
	case "pgup":
		p.logViewer.PageUp()
	case "pgdown":
		p.logViewer.PageDown()
	case "g":
		p.logViewer.GotoTop()
	case "G":
		p.logViewer.GotoBottom()
	}
	return p, nil
}

func (p *Plugin) streamLogs(id string) tea.Cmd {
	return func() tea.Msg {
		lines, err := p.client.FetchLogs(id, dt.LogOptions{Tail: "200"})
		if err != nil {
			return logDoneMsg{err: err}
		}
		return logDoneMsg{lines: lines}
	}
}

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

func shortStr(s string, maxLen int) string {
	return theme.Truncate(s, maxLen)
}

func containerTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.ColorBorder)).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color(theme.ColorAccent))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(theme.ColorFg)).
		Background(lipgloss.Color(theme.ColorSelected)).
		Bold(false)
	s.Cell = s.Cell.Foreground(lipgloss.Color(theme.ColorFg))
	return s
}
