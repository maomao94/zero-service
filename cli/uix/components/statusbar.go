package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

var (
	statusbarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorFg)).
			Background(lipgloss.Color(theme.ColorBg))

	statusbarLeftStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorAccent)).
				Bold(true).
				Padding(0, 1)

	statusbarRightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorDim)).
				Padding(0, 1)

	statusbarBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorBorder))
)

// StatusBar is the bottom status bar showing plugin name and help hints.
type StatusBar struct {
	left  string
	right string
	width int
}

// NewStatusBar creates a new StatusBar.
func NewStatusBar() StatusBar {
	return StatusBar{width: 80}
}

// SetLeft sets the left-side text (plugin name).
func (s *StatusBar) SetLeft(text string) {
	s.left = text
}

// SetRight sets the right-side text (help hints).
func (s *StatusBar) SetRight(text string) {
	s.right = text
}

// View renders the status bar with a top border.
func (s StatusBar) View() string {
	width := s.width
	if width <= 0 {
		width = 80
	}
	if width < 4 {
		width = 4
	}

	border := statusbarBorderStyle.Render(strings.Repeat("─", width))

	leftInnerWidth := min(lipgloss.Width(s.left), max(0, width/3))
	leftText := statusbarLeftStyle.Render(" " + theme.Truncate(s.left, leftInnerWidth) + " ")
	leftWidth := lipgloss.Width(leftText)

	rightInnerWidth := width - leftWidth - 1 - 2
	if rightInnerWidth < 0 {
		rightInnerWidth = 0
	}
	rightText := statusbarRightStyle.Render(theme.Truncate(s.right, rightInnerWidth) + " ")

	spacer := width - lipgloss.Width(leftText) - lipgloss.Width(rightText)
	if spacer < 0 {
		spacer = 0
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		leftText,
		statusbarStyle.Render(strings.Repeat(" ", spacer)),
		rightText,
	)

	return border + "\n" + content
}

// SetWidth updates the status bar width.
func (s *StatusBar) SetWidth(w int) {
	if w <= 0 {
		w = 80
	}
	s.width = w
}
