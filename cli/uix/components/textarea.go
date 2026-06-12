package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// TextArea wraps bubbles/textarea with project theme styling.
type TextArea struct {
	textarea textarea.Model
	width    int
	height   int
}

// NewTextArea creates a new TextArea with the given dimensions.
func NewTextArea(width, height int) TextArea {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 5
	}
	ta := textarea.New()
	ta.Placeholder = "Enter text..."
	ta.CharLimit = 0
	ta.SetWidth(width)
	ta.SetHeight(height)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color(theme.ColorSelected))
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent))
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg))
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	ta.BlurredStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	return TextArea{
		textarea: ta,
		width:    width,
		height:   height,
	}
}

// Focus sets focus on the textarea.
func (ta *TextArea) Focus() tea.Cmd {
	return ta.textarea.Focus()
}

// Blur removes focus from the textarea.
func (ta *TextArea) Blur() {
	ta.textarea.Blur()
}

// Focused returns whether the textarea is focused.
func (ta TextArea) Focused() bool {
	return ta.textarea.Focused()
}

// Value returns the current text content.
func (ta TextArea) Value() string {
	return ta.textarea.Value()
}

// SetValue sets the text content.
func (ta *TextArea) SetValue(s string) {
	ta.textarea.SetValue(s)
}

// SetPlaceholder sets the placeholder text.
func (ta *TextArea) SetPlaceholder(s string) {
	ta.textarea.Placeholder = s
}

// SetCharLimit sets the maximum character limit. 0 means no limit.
func (ta *TextArea) SetCharLimit(limit int) {
	ta.textarea.CharLimit = limit
}

// LineCount returns the number of lines in the textarea.
func (ta TextArea) LineCount() int {
	val := ta.textarea.Value()
	if val == "" {
		return 0
	}
	count := 1
	for _, c := range val {
		if c == '\n' {
			count++
		}
	}
	return count
}

// Update processes textarea messages.
func (ta TextArea) Update(msg tea.Msg) (TextArea, tea.Cmd) {
	var cmd tea.Cmd
	ta.textarea, cmd = ta.textarea.Update(msg)
	return ta, cmd
}

// View renders the textarea with optional line count display.
func (ta TextArea) View() string {
	content := ta.textarea.View()
	width := ta.width
	if width <= 0 {
		width = 80
	}
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorDim)).
		Width(width).
		Align(lipgloss.Right).
		Render(fmt.Sprintf("lines: %d", ta.LineCount()))
	return content + "\n" + footer
}

// SetSize updates the textarea dimensions.
func (ta *TextArea) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 5
	}
	ta.width = width
	ta.height = height
	ta.textarea.SetWidth(width)
	ta.textarea.SetHeight(height)
}
