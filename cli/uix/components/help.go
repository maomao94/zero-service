package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// HelpBinding represents a key binding for the help component.
type HelpBinding struct {
	Keys []string
	Desc string
}

// Help wraps bubbles/help with project theme styling.
type Help struct {
	model    help.Model
	bindings []key.Binding
	width    int
}

// NewHelp creates a new Help component.
func NewHelp() Help {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent))
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent))
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg))
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	h.Styles.Ellipsis = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	return Help{
		model: h,
		width: 80,
	}
}

// AddBinding adds a key binding to the help component.
func (h *Help) AddBinding(b HelpBinding) {
	if len(b.Keys) == 0 {
		return
	}
	kb := key.NewBinding(
		key.WithKeys(b.Keys[0]),
		key.WithHelp(strings.Join(b.Keys, "/"), b.Desc),
	)
	h.bindings = append(h.bindings, kb)
}

// SetBindings replaces all bindings.
func (h *Help) SetBindings(bindings []HelpBinding) {
	h.bindings = make([]key.Binding, 0, len(bindings))
	for _, b := range bindings {
		h.AddBinding(b)
	}
}

// ShowFullHelp toggles between short and full help display.
func (h *Help) ShowFullHelp() {
	h.model.ShowAll = true
}

// ShowShortHelp toggles between short and full help display.
func (h *Help) ShowShortHelp() {
	h.model.ShowAll = false
}

// ToggleHelp toggles between short and full help display.
func (h *Help) ToggleHelp() {
	h.model.ShowAll = !h.model.ShowAll
}

// Update processes help messages.
func (h Help) Update(msg tea.Msg) (Help, tea.Cmd) {
	var cmd tea.Cmd
	h.model, cmd = h.model.Update(msg)
	return h, cmd
}

// View renders the help bar.
func (h Help) View() string {
	if len(h.bindings) == 0 {
		return ""
	}
	return h.model.ShortHelpView(h.bindings)
}

// ViewFull renders the full help view.
func (h Help) ViewFull() string {
	if len(h.bindings) == 0 {
		return ""
	}
	return h.model.FullHelpView([][]key.Binding{h.bindings})
}

// SetSize updates the help width.
func (h *Help) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	h.width = width
	h.model.Width = width
}

// BindingCount returns the number of registered bindings.
func (h Help) BindingCount() int {
	return len(h.bindings)
}
