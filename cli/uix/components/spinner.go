package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// Spinner wraps bubbles/spinner with project theme styling.
type Spinner struct {
	spinner spinner.Model
	active  bool
	width   int
}

// NewSpinner creates a new Spinner with default project theme colors.
func NewSpinner() Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent))
	return Spinner{
		spinner: s,
		width:   80,
	}
}

// Start activates the spinner and returns the tick command.
func (s *Spinner) Start() tea.Cmd {
	s.active = true
	return s.spinner.Tick
}

// Stop deactivates the spinner.
func (s *Spinner) Stop() {
	s.active = false
}

// IsActive returns whether the spinner is active.
func (s Spinner) IsActive() bool {
	return s.active
}

// SetSpinner sets a custom spinner style.
func (s *Spinner) SetSpinner(sp spinner.Spinner) {
	s.spinner.Spinner = sp
}

// SetStyle sets a custom Lip Gloss style for the spinner.
func (s *Spinner) SetStyle(style lipgloss.Style) {
	s.spinner.Style = style
}

// Update processes spinner tick messages.
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the spinner. Returns empty string if inactive.
func (s Spinner) View() string {
	if !s.active {
		return ""
	}
	return s.spinner.View()
}

// SetSize updates the spinner width.
func (s *Spinner) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	s.width = width
}
