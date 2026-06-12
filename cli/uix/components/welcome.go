package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

type WelcomeScreen struct {
	Logo   string
	width  int
	height int
}

func NewWelcomeScreen(logo string) WelcomeScreen {
	return WelcomeScreen{
		Logo:   logo,
		width:  80,
		height: 24,
	}
}

func (w *WelcomeScreen) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	w.width = width
	w.height = height
}

func (w WelcomeScreen) View() string {
	var b strings.Builder

	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorAccent)).
		Bold(true).
		Align(lipgloss.Center)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorDim)).
		Align(lipgloss.Center)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorYellow)).
		Align(lipgloss.Center).
		Padding(1, 2)

	slashStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.ColorSelected)).
		Foreground(lipgloss.Color(theme.ColorAccent)).
		Bold(true).
		Padding(0, 1)

	b.WriteString(logoStyle.Width(w.width).Render(w.Logo))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Width(w.width).Render("终端管理控制台"))
	b.WriteString("\n\n")
	slashHint := slashStyle.Render("/") + "  打开指令面板"
	b.WriteString(hintStyle.Width(w.width).Render(slashHint))

	content := b.String()
	return lipgloss.Place(
		w.width, w.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
