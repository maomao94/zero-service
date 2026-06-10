package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// ModalButton represents a button in a modal dialog.
type ModalButton struct {
	Label string
	Key   string // key to press (e.g., "enter", "esc")
}

// Modal is a centered overlay dialog with a title, message, and buttons.
type Modal struct {
	Title   string
	Message string
	Buttons []ModalButton
	Active  int // index of selected button
	Width   int
}

// NewModal creates a new Modal.
func NewModal(title, message string, buttons []ModalButton, width int) Modal {
	return Modal{
		Title:   title,
		Message: message,
		Buttons: buttons,
		Width:   width,
	}
}

// NextButton moves to the next button.
func (m *Modal) NextButton() {
	if len(m.Buttons) > 0 {
		m.Active = (m.Active + 1) % len(m.Buttons)
	}
}

// PrevButton moves to the previous button.
func (m *Modal) PrevButton() {
	if len(m.Buttons) > 0 {
		m.Active = (m.Active - 1 + len(m.Buttons)) % len(m.Buttons)
	}
}

// SelectedButton returns the currently selected button.
func (m Modal) SelectedButton() *ModalButton {
	if len(m.Buttons) == 0 || m.Active < 0 || m.Active >= len(m.Buttons) {
		return nil
	}
	return &m.Buttons[m.Active]
}

// View renders the modal as a centered overlay.
func (m Modal) View() string {
	width := m.Width
	if width <= 0 {
		width = 60
	}
	// Cap width
	if width > 100 {
		width = 100
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorAccent))
	b.WriteString(titleStyle.Render(m.Title))
	b.WriteString("\n\n")

	// Message
	b.WriteString(m.Message)
	b.WriteString("\n\n")

	// Buttons
	if len(m.Buttons) > 0 {
		var buttonParts []string
		for i, btn := range m.Buttons {
			style := lipgloss.NewStyle()
			if i == m.Active {
				style = style.Bold(true).Foreground(lipgloss.Color(theme.ColorAccent))
				buttonParts = append(buttonParts, style.Render("[ "+btn.Label+" ]"))
			} else {
				style = style.Foreground(lipgloss.Color(theme.ColorDim))
				buttonParts = append(buttonParts, style.Render("  "+btn.Label+"  "))
			}
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, buttonParts...))
	}

	content := theme.Border(m.Title).Width(width - 4).Render(b.String())
	return lipgloss.Place(m.Width, lipgloss.Height(content),
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
