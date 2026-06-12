package test

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

type Module struct {
	width        int
	height       int
	status       string
	state        components.StateKind
	selectedFile string
	log          components.LogViewer
	logMode      bool
	chartMode    bool
	sparkline    *components.Sparkline
}

func New() *Module {
	log := components.NewLogViewer(80, 10)
	log.AppendLine("test module ready")
	log.AppendLine("press m for modal, # for file picker, l for log view, c for chart")
	sl := components.NewSparkline(30, 5)
	sl.SetData([]float64{10, 25, 15, 30, 20, 35, 25, 40, 30, 45})
	return &Module{
		width:     80,
		height:    20,
		status:    "ready",
		state:     components.StateEmpty,
		log:       log,
		sparkline: sl,
	}
}

func (m *Module) Name() string        { return "test" }
func (m *Module) Description() string { return "Exercise uix shell features without Docker" }
func (m *Module) Aliases() []string   { return []string{"t", "demo"} }
func (m *Module) Init() tea.Cmd       { return nil }
func (m *Module) IsRoot() bool        { return !m.logMode && !m.chartMode }

func (m *Module) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"m"}, Desc: "modal"},
		{Keys: []string{"#"}, Desc: "file"},
		{Keys: []string{"l"}, Desc: "logs"},
		{Keys: []string{"c"}, Desc: "chart"},
		{Keys: []string{"a"}, Desc: "append"},
		{Keys: []string{"e"}, Desc: "error"},
		{Keys: []string{"r"}, Desc: "reset"},
	}
}

func (m *Module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case uix.ConfirmMsg:
		return m.handleConfirm(msg.Button)
	case uix.FileSelectedMsg:
		m.selectedFile = msg.Path
		m.state = components.StateSuccess
		m.status = "file selected"
		m.log.AppendLine("file selected: " + msg.Path)
		return m, nil
	case tea.KeyMsg:
		if m.logMode {
			return m.handleLogKey(msg)
		}
		if m.chartMode {
			return m.handleChartKey(msg)
		}
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.log, cmd = m.log.Update(msg)
	return m, cmd
}

func (m *Module) View() string {
	if m.logMode {
		return components.LogHeader("test output", m.log.IsFollowing()) + "\n" + m.log.View()
	}

	if m.chartMode {
		return m.chartView()
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true).Render(" uix test module "))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("Exercises shared shell routing, modal, file picker, log viewer, and state views."))
	b.WriteString("\n\n")
	b.WriteString(renderField("Status", m.status))
	b.WriteString("\n")
	if m.selectedFile == "" {
		b.WriteString(renderField("File", "none selected; type # in the prompt"))
	} else {
		b.WriteString(renderField("File", m.selectedFile))
	}
	b.WriteString("\n\n")
	b.WriteString(components.RenderState(m.state, "module state", stateMessage(m.state), m.width))
	b.WriteString("\n\n")
	b.WriteString(components.LogHeader("recent output", m.log.IsFollowing()))
	b.WriteString("\n")
	b.WriteString(m.log.View())
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("m modal | # file picker | l log view | c chart | a append log | e error state | r reset | esc back"))

	panel := components.NewPanel("test", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
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
	logWidth := width - 8
	if logWidth < 20 {
		logWidth = 20
	}
	logHeight := height - 15
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

func (m *Module) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "m":
		return m, func() tea.Msg {
			return uix.ShowModalMsg{
				Title:   "Test Modal",
				Message: "Enter confirms the action button. Esc closes the modal without running anything.",
				Buttons: []components.ModalButton{
					{Label: "Cancel", Key: "esc"},
					{Label: "Run Demo", Key: "enter"},
				},
			}
		}
	case "c":
		m.chartMode = true
	case "a":
		line := "log appended at " + time.Now().Format("15:04:05")
		m.log.AppendLine(line)
		m.status = "appended output"
		m.state = components.StateSuccess
	case "e":
		m.status = "simulated error"
		m.state = components.StateError
		m.log.AppendLine("error: simulated module failure")
	case "l":
		m.logMode = true
		m.SetSize(m.width, m.height)
	case "r":
		m.status = "ready"
		m.state = components.StateEmpty
		m.selectedFile = ""
		m.log.SetLines([]string{"test module reset", "type /help for shell help"})
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

func (m *Module) handleChartKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.chartMode = false
	case "a":
		m.sparkline.AddData(float64(20 + time.Now().UnixMilli()%30))
		m.status = "added data point"
	}
	return m, nil
}

func (m *Module) chartView() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true).Render(" sparkline chart "))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("Sample sparkline using ntcharts library."))
	b.WriteString("\n\n")
	b.WriteString(m.sparkline.View())
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("a add data | esc/q back"))

	panel := components.NewPanel("chart demo", m.width, m.height)
	panel.Body = b.String()
	return panel.View()
}

func (m *Module) handleConfirm(button string) (tea.Model, tea.Cmd) {
	if button != "Run Demo" {
		m.status = "modal cancelled"
		m.state = components.StateWarning
		m.log.AppendLine("modal cancelled with " + button)
		return m, nil
	}
	m.status = "modal confirmed"
	m.state = components.StateSuccess
	m.log.AppendLine("modal confirmed at " + time.Now().Format("15:04:05"))
	return m, func() tea.Msg {
		return uix.AppendMessageMsg{Role: uix.RoleModule, Content: "test module modal action confirmed"}
	}
}

func renderField(label, value string) string {
	return fmt.Sprintf("%s  %s",
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(label),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg)).Render(value),
	)
}

func stateMessage(kind components.StateKind) string {
	switch kind {
	case components.StateSuccess:
		return "Last action completed and wrote to the shared log output."
	case components.StateWarning:
		return "The action was cancelled or needs attention."
	case components.StateError:
		return "The module is showing a simulated error state."
	default:
		return "No action has been run yet."
	}
}
