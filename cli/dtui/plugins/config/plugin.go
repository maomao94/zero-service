package config

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/dtui/internal/config"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

var sectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true)

type formStep int

const (
	formNone formStep = iota
	formAddCompose
	formAddDeploy
)

// Module manages application configuration with the new uix shell contract.
type Module struct {
	width  int
	height int

	cfg        config.Config
	configPath string

	table   components.Table
	spinner components.Spinner
	state   components.StateKind
	status  string

	cursor int

	formStep   formStep
	formInputs []textinput.Model
	formFocus  int

	pending string
}

// New creates a new config module.
func New(cfg config.Config, configPath string) *Module {
	cols := []components.Column{
		{Title: "Section", Width: 12},
		{Title: "Name", Width: 20},
		{Title: "Value", Width: 50},
	}
	t := components.NewTable(cols, nil, 80)
	sp := components.NewSpinner()

	if configPath == "" {
		configPath = config.DefaultPath()
	}

	return &Module{
		width:      80,
		height:     20,
		cfg:        cfg,
		configPath: configPath,
		table:      t,
		spinner:    sp,
		state:      components.StateLoading,
		status:     "loading...",
	}
}

func (m *Module) Name() string        { return "config" }
func (m *Module) Description() string { return "Manage application configuration" }
func (m *Module) Aliases() []string   { return []string{"cfg"} }
func (m *Module) IsRoot() bool        { return m.formStep == formNone }

func (m *Module) Init() tea.Cmd {
	return tea.Batch(m.spinner.Start(), m.reload())
}

func (m *Module) Bindings() []uix.HelpBinding {
	if m.formStep != formNone {
		return []uix.HelpBinding{
			{Keys: []string{"Tab/Enter"}, Desc: "下一个"},
			{Keys: []string{"Esc"}, Desc: "取消"},
		}
	}
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"a"}, Desc: "新增"},
		{Keys: []string{"d"}, Desc: "删除"},
		{Keys: []string{"e"}, Desc: "编辑JSON"},
		{Keys: []string{"r"}, Desc: "刷新"},
	}
}

func (m *Module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case configLoadedMsg:
		return m.handleConfigLoaded(msg)

	case uix.ConfirmMsg:
		return m.handleConfirm(msg.Button)

	case tea.KeyMsg:
		if m.formStep != formNone {
			return m.handleFormKey(msg)
		}
		return m.handleKey(msg)
	}

	// Forward spinner ticks.
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *Module) View() string {
	if m.formStep != formNone {
		return m.renderForm()
	}

	if m.state == components.StateLoading && m.totalRows() == 0 {
		return m.renderLoading()
	}
	if m.state == components.StateError && m.totalRows() == 0 {
		return m.renderError()
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
	b.WriteString(m.spinner.View() + " Loading configuration...")
	if m.status != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(m.status))
	}
	panel := components.NewPanel("config", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderError() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateError, "Failed to load configuration", m.status, m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("r retry | esc back"))
	panel := components.NewPanel("config", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderTable() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(" Configuration "))
	b.WriteString(fmt.Sprintf("    Compose: %d  |  Deploy: %d", len(m.cfg.ComposeDirs), len(m.cfg.DeployTargets)))
	b.WriteString("\n\n")
	if m.status != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render(m.status))
		b.WriteString("\n\n")
	}
	b.WriteString(m.table.View())
	panel := components.NewPanel("config", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderForm() string {
	labels := []string{}
	switch m.formStep {
	case formAddCompose:
		labels = []string{"Name", "Path"}
	case formAddDeploy:
		labels = []string{"Name", "Container", "HTML Path", "Backup Dir"}
	}

	var b strings.Builder
	b.WriteString(sectionStyle.Render(" New Entry "))
	b.WriteString("\n\n")
	for i, input := range m.formInputs {
		prefix := "  "
		if i == m.formFocus {
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

// --- Message handlers ---

func (m *Module) handleConfigLoaded(msg configLoadedMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	m.cfg = msg.cfg
	m.state = components.StateSuccess
	m.status = ""
	m.updateTableRows()
	return m, nil
}

func (m *Module) handleConfirm(button string) (tea.Model, tea.Cmd) {
	defer func() { m.pending = "" }()

	switch m.pending {
	case "add_type":
		switch button {
		case "Compose Dir":
			return m.startAddCompose()
		case "Deploy Target":
			return m.startAddDeploy()
		}
	case "delete":
		if button == "Delete" {
			section, idx := m.currentEntry()
			switch section {
			case "Compose":
				config.RemoveComposeDir(m.configPath, idx)
				m.status = "Deleted compose dir"
			case "Deploy":
				config.RemoveDeployTarget(m.configPath, idx)
				m.status = "Deleted deploy target"
			}
			return m, m.reload()
		}
	}
	return m, nil
}

func (m *Module) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.table.SetCursor(m.cursor)
		}
		return m, nil
	case "down", "j":
		if m.cursor < m.totalRows()-1 {
			m.cursor++
			m.table.SetCursor(m.cursor)
		}
		return m, nil
	case "r":
		m.status = "refreshing..."
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.reload())
	case "a":
		return m.chooseAddType()
	case "d":
		return m.confirmDelete()
	case "e":
		return m.openEditor()
	}
	return m, nil
}

func (m *Module) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.formStep = formNone
		return m, m.reload()
	case "enter", "tab":
		m.formFocus++
		if m.formFocus >= len(m.formInputs) {
			return m.submitForm()
		}
		return m, m.focusFormInput(m.formFocus)
	}
	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

// --- Actions ---

func (m *Module) chooseAddType() (tea.Model, tea.Cmd) {
	m.pending = "add_type"
	return m, func() tea.Msg {
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

func (m *Module) confirmDelete() (tea.Model, tea.Cmd) {
	section, _ := m.currentEntry()
	if section == "" {
		return m, nil
	}
	m.pending = "delete"
	return m, func() tea.Msg {
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

func (m *Module) currentEntry() (section string, index int) {
	i := m.cursor
	if i < len(m.cfg.ComposeDirs) {
		return "Compose", i
	}
	i -= len(m.cfg.ComposeDirs)
	if i < len(m.cfg.DeployTargets) {
		return "Deploy", i
	}
	return "", -1
}

func (m *Module) startAddCompose() (tea.Model, tea.Cmd) {
	m.formStep = formAddCompose
	m.formInputs = []textinput.Model{m.newTi("Name", ""), m.newTi("Path", "")}
	return m, m.focusFormInput(0)
}

func (m *Module) startAddDeploy() (tea.Model, tea.Cmd) {
	m.formStep = formAddDeploy
	m.formInputs = []textinput.Model{
		m.newTi("Name", ""),
		m.newTi("Container", ""),
		m.newTi("HTML Path", "/usr/share/nginx/html"),
		m.newTi("Backup Dir", config.DefaultBackupDir("")),
	}
	return m, m.focusFormInput(0)
}

func (m *Module) submitForm() (tea.Model, tea.Cmd) {
	defer func() { m.formStep = formNone }()

	switch m.formStep {
	case formAddCompose:
		name := m.formInputs[0].Value()
		path := m.formInputs[1].Value()
		if name == "" || path == "" {
			m.status = "Name and path required"
			return m, m.reload()
		}
		if err := config.AddComposeDir(m.configPath, name, path); err != nil {
			m.status = "Failed: " + err.Error()
		} else {
			m.status = fmt.Sprintf("Added compose dir: %s", name)
		}

	case formAddDeploy:
		name := m.formInputs[0].Value()
		container := m.formInputs[1].Value()
		htmlPath := m.formInputs[2].Value()
		backupDir := m.formInputs[3].Value()
		if name == "" || container == "" {
			m.status = "Name and container required"
			return m, m.reload()
		}
		if backupDir == "" {
			backupDir = config.DefaultBackupDir(container)
		}
		if err := config.AddDeployTarget(m.configPath, name, container, htmlPath, backupDir); err != nil {
			m.status = "Failed: " + err.Error()
		} else {
			m.status = fmt.Sprintf("Added deploy target: %s", name)
		}
	}
	return m, m.reload()
}

func (m *Module) openEditor() (tea.Model, tea.Cmd) {
	editor := "vi"
	for _, e := range []string{"vim", "nano"} {
		if _, err := exec.LookPath(e); err == nil {
			editor = e
			break
		}
	}
	return m, tea.ExecProcess(
		exec.Command(editor, m.configPath),
		func(err error) tea.Msg {
			return configLoadedMsg{cfg: config.Load(m.configPath)}
		},
	)
}

// --- Helpers ---

func (m *Module) newTi(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.Width = 40
	return ti
}

func (m *Module) focusFormInput(index int) tea.Cmd {
	if len(m.formInputs) == 0 {
		return nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(m.formInputs) {
		index = len(m.formInputs) - 1
	}
	for i := range m.formInputs {
		m.formInputs[i].Blur()
	}
	m.formFocus = index
	return m.formInputs[m.formFocus].Focus()
}

func (m *Module) totalRows() int {
	return len(m.cfg.ComposeDirs) + len(m.cfg.DeployTargets) + len(m.cfg.DeployPackages)
}

func (m *Module) updateTableRows() {
	var rows []components.Row
	for _, d := range m.cfg.ComposeDirs {
		rows = append(rows, components.Row{"Compose", d.Name, d.Path})
	}
	for _, t := range m.cfg.DeployTargets {
		rows = append(rows, components.Row{"Deploy", t.Name, t.Container + " -> " + t.HtmlPath})
	}
	for _, p := range m.cfg.DeployPackages {
		rows = append(rows, components.Row{"Package", p.Name, p.Path})
	}
	m.table.SetRows(rows)
	if m.cursor >= len(rows) {
		m.cursor = max(0, len(rows)-1)
		m.table.SetCursor(m.cursor)
	}
}

// --- Async commands ---

func (m *Module) reload() tea.Cmd {
	return func() tea.Msg { return configLoadedMsg{cfg: config.Load(m.configPath)} }
}

// --- Messages ---

type configLoadedMsg struct{ cfg config.Config }
