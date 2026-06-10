package theme

import "github.com/charmbracelet/lipgloss"

func WidthStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width).MaxWidth(width)
}

func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 2 {
		return s[:min(len(s), maxWidth)]
	}

	truncated := s
	for lipgloss.Width(truncated)+2 > maxWidth && len(truncated) > 0 {
		_, size := lastRune(truncated)
		truncated = truncated[:len(truncated)-size]
	}
	return truncated + ".."
}

func Border(title string) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBorder)).
		BorderTop(true).
		BorderRight(true).
		BorderBottom(true).
		BorderLeft(true).
		Padding(0, 1)
}

var (
	PaletteOverlayStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(ColorSurface)).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorAccent)).
				BorderTop(true).
				BorderRight(true).
				BorderBottom(true).
				BorderLeft(true)

	PaletteQueryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorFg)).
				Padding(0, 1)

	PaletteItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorFg)).
				Padding(0, 1)

	PaletteItemSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(ColorSelected)).
				Foreground(lipgloss.Color(ColorFg)).
				Bold(true).
				Padding(0, 1)

	PaletteDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDim)).
			Padding(0, 0, 0, 1)

	PaletteHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDim)).
			Padding(0, 1)

	PalettePrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorAccent)).
				Bold(true)
)

func lastRune(s string) (rune, int) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i]&0xc0 != 0x80 {
			return []rune(s[i:])[0], len(s) - i
		}
	}
	return 0, 0
}
