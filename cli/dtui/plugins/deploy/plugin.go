package deploy

import (
	"fmt"
	"strings"

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

type deployStep int

const (
	stepSelect deployStep = iota
	stepRunning
)

type Plugin struct {
	client        *dt.Client
	cfg           config.Config
	table         table.Model
	width         int
	height        int
	cursor        int
	targets       []config.DeployTarget
	step          deployStep
	selectedPath  string
	pendingTarget config.DeployTarget
	status        string
}

func New(client *dt.Client, cfg config.Config) *Plugin {
	cols := []table.Column{
		{Title: "Name", Width: 18},
		{Title: "Container", Width: 18},
		{Title: "HTML Path", Width: 28},
		{Title: "Backup Dir", Width: 18},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderForeground(lipgloss.Color(theme.ColorBorder)).BorderBottom(true).Foreground(lipgloss.Color(theme.ColorAccent))
	s.Selected = s.Selected.Background(lipgloss.Color(theme.ColorSelected)).Foreground(lipgloss.Color(theme.ColorFg))
	t.SetStyles(s)

	return &Plugin{client: client, cfg: cfg, table: t}
}

func (p *Plugin) Name() string        { return "deploy" }
func (p *Plugin) Description() string { return "Frontend deployment" }
func (p *Plugin) Aliases() []string   { return []string{"dep"} }
func (p *Plugin) IsRoot() bool        { return p.step == stepSelect && p.pendingTarget.Name == "" }
func (p *Plugin) OnActivate() tea.Cmd { return p.loadTargets() }
func (p *Plugin) OnDeactivate()       {}

func (p *Plugin) Init() tea.Cmd { return p.loadTargets() }

func (p *Plugin) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case targetsLoadedMsg:
		p.targets = msg.targets
		p.updateTable()
		return p, nil

	case deployResultMsg:
		p.step = stepSelect
		if msg.err != nil {
			p.status = "Deploy failed: " + msg.err.Error()
		} else {
			p.status = "Deploy completed: " + msg.output
		}
		return p, nil

	case uix.ConfirmMsg:
		if msg.Button == "Deploy" && p.selectedPath != "" {
			return p.executeDeploy()
		}
		p.selectedPath = ""
		return p, nil

	case uix.FileSelectedMsg:
		if p.pendingTarget.Name != "" {
			p.selectedPath = msg.Path
			return p.showConfirm()
		}
		return p, nil

	case tea.KeyMsg:
		return p.handleKey(msg)
	}

	var cmd tea.Cmd
	p.table, cmd = p.table.Update(msg)
	return p, cmd
}

func (p *Plugin) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "j":
		if p.cursor < len(p.targets)-1 {
			p.cursor++
		}
	case "r":
		return p, p.loadTargets()
	case "d":
		return p.startDeploy()
	case "esc":
		if p.pendingTarget.Name != "" {
			p.pendingTarget = config.DeployTarget{}
			p.selectedPath = ""
			p.status = "Deploy cancelled"
		}
	}
	return p, nil
}

func (p *Plugin) View() string {
	switch p.step {
	case stepRunning:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorYellow)).
			Render(p.status)
	default:
		var b strings.Builder
		if p.status != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render(p.status))
			b.WriteString("\n\n")
		}
		if p.pendingTarget.Name != "" {
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorAccent)).
				Bold(true).
				Render(fmt.Sprintf(" Target: %s (%s)", p.pendingTarget.Name, p.pendingTarget.Container)))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorYellow)).
				Render(" Type # to select deployment file"))
			b.WriteString("\n\n")
			if p.selectedPath != "" {
				b.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color(theme.ColorGreen)).
					Render(fmt.Sprintf(" File: %s", p.selectedPath)))
				b.WriteString("\n")
				b.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color(theme.ColorDim)).
					Render(" Press Enter to confirm deploy"))
				b.WriteString("\n\n")
			}
		}
		b.WriteString(border.Width(p.width - 2).Render(p.table.View()))
		return b.String()
	}
}

func (p *Plugin) SetSize(w, h int) {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 20
	}
	p.width = w
	p.height = h
	p.table.SetWidth(max(20, w-6))
	p.table.SetHeight(max(5, h-4))
}

func (p *Plugin) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"d"}, Desc: "部署"},
		{Keys: []string{"#"}, Desc: "选择文件"},
		{Keys: []string{"r"}, Desc: "刷新"},
	}
}

func (p *Plugin) updateTable() {
	rows := make([]table.Row, len(p.targets))
	for i, t := range p.targets {
		rows[i] = table.Row{t.Name, t.Container, t.HtmlPath, t.BackupDir}
	}
	p.table.SetRows(rows)
	if p.cursor >= len(p.targets) {
		p.cursor = max(0, len(p.targets)-1)
	}
	p.table.SetCursor(p.cursor)
}

func (p *Plugin) startDeploy() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.targets) {
		return p, nil
	}
	p.pendingTarget = p.targets[p.cursor]
	p.selectedPath = ""
	p.status = ""
	return p, nil
}

func (p *Plugin) showConfirm() (tea.Model, tea.Cmd) {
	t := p.pendingTarget
	fileType := dt.PathType(p.selectedPath)
	msg := fmt.Sprintf("Target: %s\nContainer: %s\nHTML: %s\nBackup: %s\nSource: %s (%s)",
		t.Name, t.Container, t.HtmlPath, t.BackupDir, p.selectedPath, fileType)
	return p, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Deploy",
			Message: msg,
			Buttons: []components.ModalButton{{Label: "Cancel", Key: "esc"}, {Label: "Deploy", Key: "enter"}},
		}
	}
}

func (p *Plugin) executeDeploy() (tea.Model, tea.Cmd) {
	p.step = stepRunning
	p.status = "Deploying..."
	t := p.pendingTarget
	path := p.selectedPath

	p.pendingTarget = config.DeployTarget{}
	p.selectedPath = ""

	return p, func() tea.Msg {
		fileType := dt.PathType(path)
		if fileType == "invalid" {
			return deployResultMsg{err: fmt.Errorf("invalid path: %s", path)}
		}

		srcDir := path
		if fileType == "zip" {
			tmpDir := t.BackupDir + "/_extract"
			if err := dt.UnzipToDir(path, tmpDir); err != nil {
				return deployResultMsg{err: fmt.Errorf("unzip failed: %w", err)}
			}
			srcDir = tmpDir
		}

		if err := p.client.CopyToContainer(t.Container, t.HtmlPath, srcDir); err != nil {
			return deployResultMsg{err: fmt.Errorf("copy to container failed: %w", err)}
		}

		return deployResultMsg{output: fmt.Sprintf("Deployed %s to %s:%s", path, t.Container, t.HtmlPath)}
	}
}

func (p *Plugin) loadTargets() tea.Cmd {
	return func() tea.Msg { return targetsLoadedMsg{targets: p.cfg.DeployTargets} }
}

type targetsLoadedMsg struct{ targets []config.DeployTarget }

type deployResultMsg struct {
	output string
	err    error
}
