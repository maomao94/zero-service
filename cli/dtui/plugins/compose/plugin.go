package compose

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/dtui/internal/config"
	dt "zero-service/cli/dtui/internal/docker"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

var border = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(theme.ColorBorder)).
	Padding(0, 1)

type composeEntry struct {
	Name string
	Path string
}

type Plugin struct {
	client  *dt.Client
	cfg     config.Config
	table   table.Model
	width   int
	height  int
	entries      []composeEntry
	cursor       int
	pendingAction string
	status        string
}

func New(client *dt.Client, cfg config.Config) *Plugin {
	cols := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Path", Width: 50},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderForeground(lipgloss.Color(theme.ColorBorder)).BorderBottom(true).Foreground(lipgloss.Color(theme.ColorAccent))
	s.Selected = s.Selected.Background(lipgloss.Color(theme.ColorSelected)).Foreground(lipgloss.Color(theme.ColorFg))
	s.Cell = s.Cell.Foreground(lipgloss.Color(theme.ColorFg))
	t.SetStyles(s)
	return &Plugin{client: client, cfg: cfg, table: t}
}

func (p *Plugin) Name() string        { return "compose" }
func (p *Plugin) Description() string { return "Docker Compose orchestration" }
func (p *Plugin) Aliases() []string   { return []string{"co"} }
func (p *Plugin) IsRoot() bool        { return true }
func (p *Plugin) OnActivate() tea.Cmd { return p.loadEntries() }
func (p *Plugin) OnDeactivate()       {}

func (p *Plugin) Init() tea.Cmd { return p.loadEntries() }

func (p *Plugin) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case composeLoadedMsg:
		p.entries = msg.entries
		p.updateTable()
		return p, nil

	case uix.ConfirmMsg:
		return p.handleConfirm(msg.Button)

	case actionResultMsg:
		p.status = msg.output
		if msg.err != nil {
			p.status = "Error: " + msg.err.Error()
		}
		return p, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.entries)-1 {
				p.cursor++
			}
		case "r":
			return p, p.loadEntries()
		case "s":
			return p.confirmUp()
		case "d":
			return p.confirmDown()
		}
	}
	var cmd tea.Cmd
	p.table, cmd = p.table.Update(msg)
	return p, cmd
}

func (p *Plugin) View() string {
	var result string
	if p.status != "" {
		result += lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render(p.status) + "\n\n"
	}
	result += border.Width(p.width - 2).Render(p.table.View())
	return result
}

func (p *Plugin) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.table.SetWidth(w - 6)
	p.table.SetHeight(h - 4)
}

func (p *Plugin) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"s"}, Desc: "Up"},
		{Keys: []string{"d"}, Desc: "Down"},
		{Keys: []string{"r"}, Desc: "刷新"},
	}
}

func (p *Plugin) updateTable() {
	rows := make([]table.Row, len(p.entries))
	for i, e := range p.entries {
		rows[i] = table.Row{e.Name, e.Path}
	}
	p.table.SetRows(rows)
	if p.cursor >= len(p.entries) {
		p.cursor = max(0, len(p.entries)-1)
	}
	p.table.SetCursor(p.cursor)
}

func (p *Plugin) confirmUp() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.entries) {
		return p, nil
	}
	e := p.entries[p.cursor]
	return p, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Compose Up",
			Message: fmt.Sprintf("docker compose -f %s up -d", e.Path),
			Buttons: []components.ModalButton{{Label: "Cancel", Key: "esc"}, {Label: "Execute", Key: "enter"}},
		}
	}
}

func (p *Plugin) confirmDown() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.entries) {
		return p, nil
	}
	e := p.entries[p.cursor]
	return p, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Compose Down",
			Message: fmt.Sprintf("docker compose -f %s down", e.Path),
			Buttons: []components.ModalButton{{Label: "Cancel", Key: "esc"}, {Label: "Execute", Key: "enter"}},
		}
	}
}

func (p *Plugin) loadEntries() tea.Cmd {
	return func() tea.Msg {
		entries := make([]composeEntry, len(p.cfg.ComposeDirs))
		for i, d := range p.cfg.ComposeDirs {
			entries[i] = composeEntry{Name: d.Name, Path: d.Path}
		}
		return composeLoadedMsg{entries: entries}
	}
}

type composeLoadedMsg struct{ entries []composeEntry }

type actionResultMsg struct {
	output string
	err    error
}

func (p *Plugin) handleConfirm(button string) (tea.Model, tea.Cmd) {
	if button == "Cancel" || p.pendingAction == "" {
		p.pendingAction = ""
		return p, nil
	}

	action := p.pendingAction
	p.pendingAction = ""
	p.status = "Executing..."

	switch action {
	case "up":
		return p, p.runComposeUp()
	case "down":
		return p, p.runComposeDown()
	}
	return p, nil
}

func (p *Plugin) runComposeUp() tea.Cmd {
	if p.cursor < 0 || p.cursor >= len(p.entries) {
		return nil
	}
	e := p.entries[p.cursor]
	return func() tea.Msg {
		out, err := dt.RunComposeUp(e.Path, "")
		return actionResultMsg{output: out, err: err}
	}
}

func (p *Plugin) runComposeDown() tea.Cmd {
	if p.cursor < 0 || p.cursor >= len(p.entries) {
		return nil
	}
	e := p.entries[p.cursor]
	return func() tea.Msg {
		out, err := dt.RunComposeDown(e.Path, "")
		return actionResultMsg{output: out, err: err}
	}
}
