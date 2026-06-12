package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/dtui/internal/config"
	dt "zero-service/cli/dtui/internal/docker"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

// Module manages frontend deployment with the new uix shell contract.
type Module struct {
	width  int
	height int

	client    *dt.Client
	clientErr error
	cfg       config.Config
	targets   []config.DeployTarget

	table    components.Table
	spinner  components.Spinner
	logView  components.LogViewer
	state    components.StateKind
	status   string
	logMode  bool
	histMode bool

	selectedPath  string
	pendingTarget config.DeployTarget
	historyPath   string
}

// New creates a new deploy module. Docker client is initialized lazily on demand.
func New(cfg config.Config) *Module {
	cols := []components.Column{
		{Title: "Name", Width: 18},
		{Title: "Container", Width: 18},
		{Title: "HTML Path", Width: 28},
		{Title: "Backup Dir", Width: 18},
	}
	t := components.NewTable(cols, nil, 80)
	sp := components.NewSpinner()
	lv := components.NewLogViewer(80, 12)
	return &Module{
		width:       80,
		height:      20,
		cfg:         cfg,
		table:       t,
		spinner:     sp,
		logView:     lv,
		state:       components.StateSuccess,
		status:      "",
		historyPath: config.HistoryPath(),
	}
}

func (m *Module) Name() string        { return "deploy" }
func (m *Module) Description() string { return "Deploy applications to Docker" }
func (m *Module) Aliases() []string   { return []string{"dep"} }
func (m *Module) IsRoot() bool        { return !m.logMode && !m.histMode }

func (m *Module) Init() tea.Cmd {
	return tea.Batch(m.spinner.Start(), m.loadTargets())
}

func (m *Module) Bindings() []uix.HelpBinding {
	if m.logMode || m.histMode {
		return []uix.HelpBinding{
			{Keys: []string{"↑↓/pg"}, Desc: "scroll"},
			{Keys: []string{"f"}, Desc: "follow"},
			{Keys: []string{"esc"}, Desc: "close"},
		}
	}
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"d"}, Desc: "部署"},
		{Keys: []string{"#"}, Desc: "选择文件"},
		{Keys: []string{"h"}, Desc: "历史"},
		{Keys: []string{"l"}, Desc: "日志"},
		{Keys: []string{"r"}, Desc: "刷新"},
	}
}

func (m *Module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.logMode {
		return m.updateLogMode(msg)
	}
	if m.histMode {
		return m.updateHistMode(msg)
	}

	switch msg := msg.(type) {
	case targetsLoadedMsg:
		return m.handleTargetsLoaded(msg)
	case deployResultMsg:
		return m.handleDeployResult(msg)
	case uix.ConfirmMsg:
		return m.handleConfirm(msg.Button)
	case uix.FileSelectedMsg:
		return m.handleFileSelected(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward spinner ticks.
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *Module) View() string {
	if m.logMode {
		return m.renderLogView()
	}
	if m.histMode {
		return m.renderHistoryView()
	}
	if m.state == components.StateLoading && len(m.targets) == 0 {
		return m.renderLoading()
	}
	if m.state == components.StateError && len(m.targets) == 0 {
		return m.renderError()
	}
	if m.state == components.StateEmpty || len(m.targets) == 0 {
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
	m.logView.SetSize(max(20, width-6), max(5, height-6))
}

// --- Rendering ---

func (m *Module) renderLoading() string {
	var b strings.Builder
	b.WriteString(m.spinner.View() + " Loading deploy targets...")
	if m.status != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(m.status))
	}
	panel := components.NewPanel("deploy", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderError() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateError, "Failed to load targets", m.status, m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("r retry | esc back"))
	panel := components.NewPanel("deploy", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderEmpty() string {
	var b strings.Builder
	b.WriteString(components.RenderState(components.StateEmpty, "No targets", "No deploy targets configured. Add targets in config to get started.", m.width-8))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("r refresh | esc back"))
	panel := components.NewPanel("deploy", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderTable() string {
	var b strings.Builder
	if m.status != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow)).Render(m.status))
		b.WriteString("\n\n")
	}
	if m.pendingTarget.Name != "" {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorAccent)).
			Bold(true).
			Render(fmt.Sprintf(" Target: %s (%s)", m.pendingTarget.Name, m.pendingTarget.Container)))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorYellow)).
			Render(" Type # to select deployment file"))
		b.WriteString("\n\n")
		if m.selectedPath != "" {
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorGreen)).
				Render(fmt.Sprintf(" File: %s", m.selectedPath)))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorDim)).
				Render(" Press Enter to confirm deploy"))
			b.WriteString("\n\n")
		}
	}
	b.WriteString(m.table.View())
	panel := components.NewPanel("deploy", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) renderLogView() string {
	var b strings.Builder
	b.WriteString(components.LogHeader("deploy output", m.logView.IsFollowing()))
	b.WriteString("\n")
	b.WriteString(m.logView.View())
	panel := components.NewPanel("deploy", m.width, m.height)
	panel.Body = b.String()
	panel.Footer = "↑↓/pg scroll | f follow | esc close"
	return panel.View()
}

func (m *Module) renderHistoryView() string {
	entries := config.LoadHistory(m.historyPath)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true).Render(" Deploy History "))
	b.WriteString(fmt.Sprintf(" (%d entries)", len(entries)))
	b.WriteString("\n\n")
	if len(entries) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("No deploy history yet."))
	} else {
		showCount := len(entries)
		if showCount > 20 {
			showCount = 20
		}
		for i := len(entries) - 1; i >= len(entries)-showCount; i-- {
			e := entries[i]
			statusIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorGreen)).Render("✓")
			if !e.Success {
				statusIcon = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorRed)).Render("✗")
			}
			timeStr := e.Time.Format("01-02 15:04")
			b.WriteString(fmt.Sprintf("  %s %s  %s  %s\n", statusIcon, timeStr, e.Target, e.Action))
			if e.Detail != "" {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("      " + theme.Truncate(e.Detail, 60)))
				b.WriteString("\n")
			}
			if e.Error != "" {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorRed)).Render("      " + theme.Truncate(e.Error, 60)))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("esc/q close"))
	panel := components.NewPanel("deploy history", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

// --- Log mode ---

func (m *Module) updateLogMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.logMode = false
			return m, nil
		case "up", "k":
			m.logView.ScrollUp()
			return m, nil
		case "down", "j":
			m.logView.ScrollDown()
			return m, nil
		case "pgup":
			m.logView.PageUp()
			return m, nil
		case "pgdown":
			m.logView.PageDown()
			return m, nil
		case "home":
			m.logView.GotoTop()
			return m, nil
		case "end":
			m.logView.GotoBottom()
			return m, nil
		case "f":
			m.logView.ToggleFollow()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.logView, cmd = m.logView.Update(msg)
	return m, cmd
}

func (m *Module) updateHistMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "q":
			m.histMode = false
			return m, nil
		}
	}
	return m, nil
}

// --- Message handlers ---

func (m *Module) handleTargetsLoaded(msg targetsLoadedMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	m.targets = msg.targets
	if len(m.targets) == 0 {
		m.state = components.StateEmpty
	} else {
		m.state = components.StateSuccess
	}
	m.updateTableRows()
	return m, nil
}

func (m *Module) handleDeployResult(msg deployResultMsg) (tea.Model, tea.Cmd) {
	m.spinner.Stop()
	m.pendingTarget = config.DeployTarget{}
	m.selectedPath = ""
	m.logMode = true
	if msg.err != nil {
		m.logView.SetLines([]string{
			lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorRed)).Render("Deploy failed: " + msg.err.Error()),
		})
		m.status = "Deploy failed"
	} else {
		m.logView.SetLines([]string{
			lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorGreen)).Render("Deploy completed"),
			msg.output,
		})
		m.status = "Deploy completed"
	}
	return m, nil
}

func (m *Module) handleConfirm(button string) (tea.Model, tea.Cmd) {
	if button != "Deploy" || m.selectedPath == "" {
		m.selectedPath = ""
		m.pendingTarget = config.DeployTarget{}
		m.status = "Deploy cancelled"
		return m, nil
	}
	return m.executeDeploy()
}

func (m *Module) handleFileSelected(msg uix.FileSelectedMsg) (tea.Model, tea.Cmd) {
	if m.pendingTarget.Name == "" {
		return m, nil
	}
	m.selectedPath = msg.Path
	return m.showConfirm()
}

func (m *Module) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.table.SetCursor(max(0, m.table.Cursor()-1))
		return m, nil
	case "down", "j":
		m.table.SetCursor(min(len(m.targets)-1, m.table.Cursor()+1))
		return m, nil
	case "r":
		m.status = "refreshing..."
		m.state = components.StateLoading
		return m, tea.Batch(m.spinner.Start(), m.loadTargets())
	case "d":
		return m.startDeploy()
	case "h":
		m.histMode = true
		return m, nil
	case "l":
		if m.logView.LineCount() > 0 {
			m.logMode = true
		}
		return m, nil
	case "esc":
		if m.pendingTarget.Name != "" {
			m.pendingTarget = config.DeployTarget{}
			m.selectedPath = ""
			m.status = "Deploy cancelled"
		}
		return m, nil
	}
	return m, nil
}

// --- Actions ---

func (m *Module) startDeploy() (tea.Model, tea.Cmd) {
	if len(m.targets) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.targets) {
		return m, nil
	}
	m.pendingTarget = m.targets[idx]
	m.selectedPath = ""
	m.status = ""
	return m, nil
}

func (m *Module) showConfirm() (tea.Model, tea.Cmd) {
	t := m.pendingTarget
	fileType := dt.PathType(m.selectedPath)
	msg := fmt.Sprintf("Target: %s\nContainer: %s\nHTML: %s\nBackup: %s\nSource: %s (%s)",
		t.Name, t.Container, t.HtmlPath, t.BackupDir, m.selectedPath, fileType)
	return m, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Deploy",
			Message: msg,
			Buttons: []components.ModalButton{{Label: "Cancel", Key: "esc"}, {Label: "Deploy", Key: "enter"}},
		}
	}
}

func (m *Module) executeDeploy() (tea.Model, tea.Cmd) {
	m.state = components.StateLoading
	m.status = "Deploying..."
	t := m.pendingTarget
	path := m.selectedPath

	m.pendingTarget = config.DeployTarget{}
	m.selectedPath = ""

	return m, tea.Batch(m.spinner.Start(), m.deployCmd(t, path))
}

func (m *Module) deployCmd(target config.DeployTarget, path string) tea.Cmd {
	return func() tea.Msg {
		client := m.ensureClient()
		if client == nil {
			config.RecordHistory(m.historyPath, config.HistoryEntry{
				Time:    time.Now(),
				Action:  "deploy",
				Target:  target.Name,
				Detail:  path,
				Success: false,
				Error:   m.clientErr.Error(),
			})
			return deployResultMsg{err: m.clientErr}
		}

		fileType := dt.PathType(path)
		if fileType == "invalid" {
			err := fmt.Errorf("invalid path: %s", path)
			config.RecordHistory(m.historyPath, config.HistoryEntry{
				Time:    time.Now(),
				Action:  "deploy",
				Target:  target.Name,
				Detail:  path,
				Success: false,
				Error:   err.Error(),
			})
			return deployResultMsg{err: err}
		}

		if err := os.MkdirAll(target.BackupDir, 0755); err != nil {
			err = fmt.Errorf("backup dir creation failed: %w", err)
			config.RecordHistory(m.historyPath, config.HistoryEntry{
				Time:    time.Now(),
				Action:  "deploy",
				Target:  target.Name,
				Detail:  path,
				Success: false,
				Error:   err.Error(),
			})
			return deployResultMsg{err: err}
		}

		backupName := time.Now().Format("20060102-150405")
		backupPath := filepath.Join(target.BackupDir, backupName)
		if err := client.CopyFromContainer(target.Container, target.HtmlPath, backupPath); err != nil {
			err = fmt.Errorf("backup failed: %w", err)
			config.RecordHistory(m.historyPath, config.HistoryEntry{
				Time:    time.Now(),
				Action:  "deploy",
				Target:  target.Name,
				Detail:  path,
				Success: false,
				Error:   err.Error(),
			})
			return deployResultMsg{err: err}
		}

		srcDir := path
		if fileType == "zip" {
			tmpDir := filepath.Join(target.BackupDir, "_extract")
			if err := dt.UnzipToDir(path, tmpDir); err != nil {
				config.RecordHistory(m.historyPath, config.HistoryEntry{
					Time:    time.Now(),
					Action:  "deploy",
					Target:  target.Name,
					Detail:  path,
					Success: false,
					Error:   err.Error(),
				})
				return deployResultMsg{err: fmt.Errorf("unzip failed: %w", err)}
			}
			srcDir = tmpDir
		}

		if err := client.CopyToContainer(target.Container, target.HtmlPath, srcDir); err != nil {
			config.RecordHistory(m.historyPath, config.HistoryEntry{
				Time:    time.Now(),
				Action:  "deploy",
				Target:  target.Name,
				Detail:  path,
				Success: false,
				Error:   err.Error(),
			})
			return deployResultMsg{err: fmt.Errorf("copy to container failed: %w", err)}
		}

		output := fmt.Sprintf("Deployed %s to %s:%s (backup: %s)", path, target.Container, target.HtmlPath, backupPath)
		config.RecordHistory(m.historyPath, config.HistoryEntry{
			Time:    time.Now(),
			Action:  "deploy",
			Target:  target.Name,
			Detail:  path,
			Success: true,
		})
		config.CleanOldBackups(target.BackupDir, 5)
		return deployResultMsg{output: output}
	}
}

func (m *Module) updateTableRows() {
	rows := make([]components.Row, len(m.targets))
	for i, t := range m.targets {
		rows[i] = components.Row{t.Name, t.Container, theme.Truncate(t.HtmlPath, 28), t.BackupDir}
	}
	m.table.SetRows(rows)
	cursor := m.table.Cursor()
	if cursor >= len(m.targets) {
		m.table.SetCursor(max(0, len(m.targets)-1))
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

func (m *Module) loadTargets() tea.Cmd {
	return func() tea.Msg { return targetsLoadedMsg{targets: m.cfg.DeployTargets} }
}

// --- Messages ---

type targetsLoadedMsg struct {
	targets []config.DeployTarget
}

type deployResultMsg struct {
	output string
	err    error
}
