package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

type StateKind string

const (
	StateEmpty   StateKind = "empty"
	StateLoading StateKind = "loading"
	StateSuccess StateKind = "success"
	StateWarning StateKind = "warning"
	StateError   StateKind = "error"
)

type StateView struct {
	Kind    StateKind
	Title   string
	Message string
	Width   int
}

func NewStateView(kind StateKind, title, message string, width int) StateView {
	if width <= 0 {
		width = 80
	}
	return StateView{Kind: kind, Title: title, Message: message, Width: width}
}

func RenderState(kind StateKind, title, message string, width int) string {
	return NewStateView(kind, title, message, width).View()
}

func (s StateView) View() string {
	width := s.Width
	if width <= 0 {
		width = 80
	}
	if width < 24 {
		width = 24
	}

	title := strings.TrimSpace(s.Title)
	if title == "" {
		title = string(s.Kind)
	}
	message := strings.TrimSpace(s.Message)
	if message == "" {
		message = defaultStateMessage(s.Kind)
	}

	icon, color := stateIcon(s.Kind)
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(color)).
		Render(icon + " " + title)
	body := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorDim)).
		Width(width - 6).
		Render(message)

	return theme.Border(title).Width(width - 4).Render(header + "\n" + body)
}

type Panel struct {
	Title  string
	Body   string
	Footer string
	Width  int
	Height int
}

func NewPanel(title string, width, height int) Panel {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	return Panel{Title: title, Width: width, Height: height}
}

func RenderPanel(title, body string, width, height int) string {
	panel := NewPanel(title, width, height)
	panel.Body = body
	return panel.View()
}

func (p Panel) View() string {
	width := p.Width
	if width <= 0 {
		width = 80
	}
	if width < 24 {
		width = 24
	}

	parts := []string{p.Body}
	if strings.TrimSpace(p.Footer) != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(p.Footer))
	}
	content := strings.Join(parts, "\n")

	style := theme.Border(p.Title).Width(width - 4)
	if p.Height > 2 {
		style = style.MaxHeight(p.Height - 2)
	}
	return style.Render(content)
}

func stateIcon(kind StateKind) (string, string) {
	switch kind {
	case StateLoading:
		return "…", theme.ColorYellow
	case StateSuccess:
		return "✓", theme.ColorGreen
	case StateWarning:
		return "!", theme.ColorYellow
	case StateError:
		return "✕", theme.ColorRed
	default:
		return "•", theme.ColorDim
	}
}

func defaultStateMessage(kind StateKind) string {
	switch kind {
	case StateLoading:
		return "Loading..."
	case StateSuccess:
		return "Done."
	case StateWarning:
		return "Check the status message."
	case StateError:
		return "Something went wrong."
	default:
		return "Nothing to show yet."
	}
}
