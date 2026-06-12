package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// LogViewer is a reusable log/terminal output viewer component.
// Features: scroll, follow mode, search, loading state.
type LogViewer struct {
	viewport viewport.Model
	lines    []string
	follow   bool
	loading  bool
	width    int
	height   int
}

// NewLogViewer creates a new LogViewer.
func NewLogViewer(width, height int) LogViewer {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 12
	}
	vp := viewport.New(width, height)
	return LogViewer{
		viewport: vp,
		follow:   true,
		width:    width,
		height:   height,
	}
}

// SetLines replaces all lines and updates the viewport.
func (lv *LogViewer) SetLines(lines []string) {
	lv.lines = lines
	lv.updateContent()
	if lv.follow {
		lv.viewport.GotoBottom()
	}
}

// AppendLine adds a single line.
func (lv *LogViewer) AppendLine(line string) {
	lv.lines = append(lv.lines, line)
	if len(lv.lines) > 500 {
		lv.lines = lv.lines[len(lv.lines)-500:]
	}
	lv.updateContent()
	if lv.follow {
		lv.viewport.GotoBottom()
	}
}

// SetLoading sets the loading state.
func (lv *LogViewer) SetLoading(loading bool) {
	lv.loading = loading
}

// ToggleFollow toggles follow mode.
func (lv *LogViewer) ToggleFollow() {
	lv.follow = !lv.follow
}

// IsFollowing returns whether follow mode is on.
func (lv LogViewer) IsFollowing() bool {
	return lv.follow
}

// Update handles key events for scrolling.
func (lv LogViewer) Update(msg tea.Msg) (LogViewer, tea.Cmd) {
	var cmd tea.Cmd
	lv.viewport, cmd = lv.viewport.Update(msg)
	return lv, cmd
}

// View renders the log viewer.
func (lv LogViewer) View() string {
	if lv.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorYellow)).
			Width(lv.safeWidth()).
			Render("Loading...")
	}
	if len(lv.lines) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorDim)).
			Width(lv.safeWidth()).
			Render("No output yet")
	}
	return lv.viewport.View()
}

// SetSize updates dimensions.
func (lv *LogViewer) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 12
	}
	lv.width = width
	lv.height = height
	lv.viewport.Width = width
	lv.viewport.Height = height
	lv.updateContent()
}

// ScrollUp scrolls up one line (disables follow).
func (lv *LogViewer) ScrollUp() {
	lv.follow = false
	lv.viewport.LineUp(1)
}

// ScrollDown scrolls down. Re-enables follow if at bottom.
func (lv *LogViewer) ScrollDown() {
	lv.viewport.LineDown(1)
	if lv.viewport.AtBottom() {
		lv.follow = true
	}
}

// PageUp scrolls up one page.
func (lv *LogViewer) PageUp() {
	lv.follow = false
	lv.viewport.PageUp()
}

// PageDown scrolls down one page.
func (lv *LogViewer) PageDown() {
	lv.viewport.PageDown()
	if lv.viewport.AtBottom() {
		lv.follow = true
	}
}

// GotoTop scrolls to top.
func (lv *LogViewer) GotoTop() {
	lv.follow = false
	lv.viewport.GotoTop()
}

// GotoBottom scrolls to bottom.
func (lv *LogViewer) GotoBottom() {
	lv.follow = true
	lv.viewport.GotoBottom()
}

// LineCount returns the number of lines.
func (lv LogViewer) LineCount() int {
	return len(lv.lines)
}

func (lv *LogViewer) updateContent() {
	lv.viewport.SetContent(strings.Join(lv.lines, "\n"))
}

func (lv LogViewer) safeWidth() int {
	if lv.width <= 0 {
		return 80
	}
	return lv.width
}

// LogHeader renders a standard header for log panels.
func LogHeader(title string, follow bool) string {
	followStatus := "follow: on"
	if !follow {
		followStatus = "follow: off"
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true).Render(" "+title+" "),
		"  ",
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(followStatus),
		"  ",
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render("esc/q close"),
	)
}
