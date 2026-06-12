package settings

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/dtui/internal/config"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

var (
	border       = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(theme.ColorBorder)).Padding(0, 1)
	sectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true)
)

type formStep int

const (
	formNone formStep = iota
	formAddCompose
	formAddDeploy
)

type Plugin struct {
	cfg        config.Config
	table      table.Model
	width      int
	height     int
	cursor     int
	configPath string
	status     string
	pending    string

	formStep   formStep
	formInputs []textinput.Model
	formFocus  int
}

func New(cfg config.Config, configPath string) *Plugin {
	cols := []table.Column{
		{Title: "Section", Width: 12},
		{Title: "Name", Width: 20},
		{Title: "Value", Width: 50},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderForeground(lipgloss.Color(theme.ColorBorder)).BorderBottom(true).Foreground(lipgloss.Color(theme.ColorAccent))
	s.Selected = s.Selected.Background(lipgloss.Color(theme.ColorSelected)).Foreground(lipgloss.Color(theme.ColorFg))
	s.Cell = s.Cell.Foreground(lipgloss.Color(theme.ColorFg))
	t.SetStyles(s)
	if configPath == "" {
		configPath = config.DefaultPath()
	}
	return &Plugin{cfg: cfg, table: t, configPath: configPath}
}

func (p *Plugin) Name() string        { return "config" }
func (p *Plugin) Description() string { return "Configuration management" }
func (p *Plugin) Aliases() []string   { return []string{"cfg"} }
func (p *Plugin) IsRoot() bool        { return p.formStep == formNone }
func (p *Plugin) OnActivate() tea.Cmd { return p.reload() }
func (p *Plugin) OnDeactivate()       {}

func (p *Plugin) Init() tea.Cmd { return p.reload() }

func (p *Plugin) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case configLoadedMsg:
		p.cfg = msg.cfg
		p.updateTable()
		return p, nil

	case uix.ConfirmMsg:
		return p.handleConfirm(msg.Button)

	case tea.KeyMsg:
		if p.formStep != formNone {
			return p.handleFormKey(msg)
		}
		return p.handleKey(msg)
	}

	if p.formStep != formNone {
		var cmd tea.Cmd
		p.formInputs[p.formFocus], cmd = p.formInputs[p.formFocus].Update(msg)
		return p, cmd
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
		if p.cursor < p.rowCount()-1 {
			p.cursor++
		}
	case "r":
		return p, p.reload()
	case "a":
		return p.chooseAddType()
	case "d":
		return p.confirmDelete()
	case "e":
		return p.openEditor()
	}
	return p, nil
}

func (p *Plugin) View() string {
	if p.formStep != formNone {
		return p.renderForm()
	}

	var b strings.Builder
	b.WriteString(sectionStyle.Render(" Configuration "))
	b.WriteString(fmt.Sprintf("    Compose: %d  |  Deploy: %d", len(p.cfg.ComposeDirs), len(p.cfg.DeployTargets)))
	b.WriteString("\n\n")
	if p.status != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render(p.status))
		b.WriteString("\n\n")
	}
	b.WriteString(border.Width(p.width - 2).Render(p.table.View()))
	return b.String()
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
	p.table.SetWidth(max(20, w-8))
	p.table.SetHeight(max(5, h-10))
}

func (p *Plugin) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"a"}, Desc: "新增"},
		{Keys: []string{"d"}, Desc: "删除"},
		{Keys: []string{"e"}, Desc: "编辑JSON"},
		{Keys: []string{"r"}, Desc: "刷新"},
		{Keys: []string{"esc"}, Desc: "返回"},
	}
}

func (p *Plugin) updateTable() {
	var rows []table.Row
	for _, d := range p.cfg.ComposeDirs {
		rows = append(rows, table.Row{"Compose", d.Name, d.Path})
	}
	for _, t := range p.cfg.DeployTargets {
		rows = append(rows, table.Row{"Deploy", t.Name, t.Container + " -> " + t.HtmlPath})
	}
	p.table.SetRows(rows)
}

func (p *Plugin) rowCount() int {
	return len(p.cfg.ComposeDirs) + len(p.cfg.DeployTargets)
}

func (p *Plugin) chooseAddType() (tea.Model, tea.Cmd) {
	p.pending = "add_type"
	return p, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Add Entry",
			Message: "Choose type to add:",
			Buttons: []components.ModalButton{
				{Label: "Compose Dir", Key: "enter"},
				{Label: "Deploy Target", Key: "enter"},
				{Label: "Cancel", Key: "esc"},
			},
		}
	}
}

func (p *Plugin) confirmDelete() (tea.Model, tea.Cmd) {
	p.pending = "delete"
	section, _ := p.currentEntry()
	if section == "" {
		p.pending = ""
		return p, nil
	}
	return p, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Delete",
			Message: fmt.Sprintf("Delete %s entry?", section),
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Delete", Key: "enter"},
			},
		}
	}
}

func (p *Plugin) handleConfirm(button string) (tea.Model, tea.Cmd) {
	defer func() { p.pending = "" }()

	switch p.pending {
	case "add_type":
		switch button {
		case "Compose Dir":
			return p.startAddCompose()
		case "Deploy Target":
			return p.startAddDeploy()
		}
	case "delete":
		if button == "Delete" {
			section, idx := p.currentEntry()
			switch section {
			case "Compose":
				config.RemoveComposeDir(p.configPath, idx)
				p.status = "Deleted compose dir"
			case "Deploy":
				config.RemoveDeployTarget(p.configPath, idx)
				p.status = "Deleted deploy target"
			}
			return p, p.reload()
		}
	}
	return p, nil
}

func (p *Plugin) currentEntry() (section string, index int) {
	i := p.cursor
	if i < len(p.cfg.ComposeDirs) {
		return "Compose", i
	}
	i -= len(p.cfg.ComposeDirs)
	if i < len(p.cfg.DeployTargets) {
		return "Deploy", i
	}
	return "", -1
}

func (p *Plugin) startAddCompose() (tea.Model, tea.Cmd) {
	p.formStep = formAddCompose
	p.formInputs = []textinput.Model{p.newTi("Name", ""), p.newTi("Path", "")}
	return p, p.focusFormInput(0)
}

func (p *Plugin) startAddDeploy() (tea.Model, tea.Cmd) {
	p.formStep = formAddDeploy
	p.formInputs = []textinput.Model{
		p.newTi("Name", ""),
		p.newTi("Container", ""),
		p.newTi("HTML Path", "/usr/share/nginx/html"),
		p.newTi("Backup Dir", config.DefaultBackupDir("")),
	}
	return p, p.focusFormInput(0)
}

func (p *Plugin) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		p.formStep = formNone
		return p, p.reload()
	case "enter", "tab":
		p.formFocus++
		if p.formFocus >= len(p.formInputs) {
			return p.submitForm()
		}
		return p, p.focusFormInput(p.formFocus)
	}
	var cmd tea.Cmd
	p.formInputs[p.formFocus], cmd = p.formInputs[p.formFocus].Update(msg)
	return p, cmd
}

func (p *Plugin) submitForm() (tea.Model, tea.Cmd) {
	defer func() { p.formStep = formNone }()

	switch p.formStep {
	case formAddCompose:
		name := p.formInputs[0].Value()
		path := p.formInputs[1].Value()
		if name == "" || path == "" {
			p.status = "Name and path required"
			return p, p.reload()
		}
		if err := config.AddComposeDir(p.configPath, name, path); err != nil {
			p.status = "Failed: " + err.Error()
		} else {
			p.status = fmt.Sprintf("Added compose dir: %s", name)
		}

	case formAddDeploy:
		name := p.formInputs[0].Value()
		container := p.formInputs[1].Value()
		htmlPath := p.formInputs[2].Value()
		backupDir := p.formInputs[3].Value()
		if name == "" || container == "" {
			p.status = "Name and container required"
			return p, p.reload()
		}
		if backupDir == "" {
			backupDir = config.DefaultBackupDir(container)
		}
		if err := config.AddDeployTarget(p.configPath, name, container, htmlPath, backupDir); err != nil {
			p.status = "Failed: " + err.Error()
		} else {
			p.status = fmt.Sprintf("Added deploy target: %s", name)
		}
	}
	return p, p.reload()
}

func (p *Plugin) newTi(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.Width = 40
	return ti
}

func (p *Plugin) focusFormInput(index int) tea.Cmd {
	if len(p.formInputs) == 0 {
		return nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(p.formInputs) {
		index = len(p.formInputs) - 1
	}
	for i := range p.formInputs {
		p.formInputs[i].Blur()
	}
	p.formFocus = index
	return p.formInputs[p.formFocus].Focus()
}

func (p *Plugin) renderForm() string {
	labels := []string{}
	switch p.formStep {
	case formAddCompose:
		labels = []string{"Name", "Path"}
	case formAddDeploy:
		labels = []string{"Name", "Container", "HTML Path", "Backup Dir"}
	}

	var b strings.Builder
	b.WriteString(sectionStyle.Render(" New Entry "))
	b.WriteString("\n\n")
	for i, input := range p.formInputs {
		prefix := "  "
		if i == p.formFocus {
			prefix = "▶ "
		}
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(prefix + labels[i]))
		b.WriteString("\n  ")
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("Tab/Enter next | Esc cancel"))
	return b.String()
}

func (p *Plugin) openEditor() (tea.Model, tea.Cmd) {
	editor := "vi"
	for _, e := range []string{"vim", "nano"} {
		if _, err := exec.LookPath(e); err == nil {
			editor = e
			break
		}
	}
	return p, tea.ExecProcess(
		exec.Command(editor, p.configPath),
		func(err error) tea.Msg {
			return configLoadedMsg{cfg: config.Load(p.configPath)}
		},
	)
}

func (p *Plugin) reload() tea.Cmd {
	return func() tea.Msg { return configLoadedMsg{cfg: config.Load(p.configPath)} }
}

type configLoadedMsg struct{ cfg config.Config }
