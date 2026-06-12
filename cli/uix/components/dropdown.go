package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

type DropdownEntry struct {
	Label       string
	Description string
	Prefix      string
}

type Dropdown struct {
	entries   []DropdownEntry
	filtered  []DropdownEntry
	cursor    int
	width     int
	maxHeight int
	visible   bool
}

func NewDropdown(width, maxHeight int) Dropdown {
	if width <= 0 {
		width = 80
	}
	if maxHeight <= 0 {
		maxHeight = 8
	}
	return Dropdown{
		width:     width,
		maxHeight: maxHeight,
		visible:   false,
	}
}

func (d *Dropdown) SetEntries(entries []DropdownEntry) {
	d.entries = entries
	d.cursor = 0
}

func (d *Dropdown) Filter(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		d.filtered = d.entries
		d.cursor = 0
		d.visible = true
		return
	}

	q := strings.ToLower(query)
	d.filtered = nil
	for _, e := range d.entries {
		if strings.Contains(strings.ToLower(e.Label), q) ||
			strings.Contains(strings.ToLower(e.Description), q) ||
			strings.Contains(strings.ToLower(e.Prefix), q) {
			d.filtered = append(d.filtered, e)
		}
	}
	if d.cursor >= len(d.filtered) {
		d.cursor = 0
	}
	d.visible = true
}

func (d *Dropdown) MoveUp() {
	if d.cursor > 0 {
		d.cursor--
	}
}

func (d *Dropdown) MoveDown() {
	if d.cursor < len(d.filtered)-1 {
		d.cursor++
	}
}

func (d *Dropdown) Selected() *DropdownEntry {
	if len(d.filtered) == 0 || d.cursor < 0 || d.cursor >= len(d.filtered) {
		return nil
	}
	return &d.filtered[d.cursor]
}

func (d *Dropdown) SetWidth(w int) {
	if w <= 0 {
		w = 80
	}
	d.width = w
}
func (d *Dropdown) Show() { d.visible = true }
func (d *Dropdown) Hide() { d.visible = false; d.cursor = 0 }

func (d Dropdown) Height() int {
	if !d.visible {
		return 0
	}
	if len(d.filtered) == 0 {
		return 3
	}
	return min(len(d.filtered), d.maxHeight) + 2
}

func (d Dropdown) visibleStartIdx() int {
	if d.cursor >= d.maxHeight {
		return d.cursor - d.maxHeight + 1
	}
	return 0
}

func (d Dropdown) View() string {
	if !d.visible {
		return ""
	}

	visibleCount := min(len(d.filtered), d.maxHeight)
	startIdx := d.visibleStartIdx()

	var b strings.Builder

	sep := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorBorder)).
		Render(strings.Repeat("─", d.width))

	b.WriteString(sep)
	b.WriteString("\n")

	if len(d.filtered) == 0 {
		empty := theme.PaletteHintStyle.Render("No matches")
		b.WriteString(fillLine(empty, d.width))
		b.WriteString("\n")
		hints := theme.PaletteHintStyle.Render("esc 取消")
		b.WriteString(fillLine(hints, d.width))
		return b.String()
	}

	for i := 0; i < visibleCount; i++ {
		idx := startIdx + i
		if idx >= len(d.filtered) {
			break
		}
		e := d.filtered[idx]

		isSel := idx == d.cursor
		prefix := "  "
		if isSel {
			prefix = theme.PalettePrefixStyle.Render("▶") + " "
		}

		label := e.Prefix + e.Label
		if isSel {
			label = theme.PaletteItemSelectedStyle.Render(label)
		} else {
			label = theme.PaletteItemStyle.Render(label)
		}

		desc := ""
		if e.Description != "" {
			desc = theme.PaletteDescStyle.Render(e.Description)
		}

		line := prefix + label + desc
		b.WriteString(fillLine(line, d.width))
		b.WriteString("\n")
	}

	hints := theme.PaletteHintStyle.Render("↑↓ 选择  ↵ 确认  esc 取消")
	b.WriteString(fillLine(hints, d.width))

	return b.String()
}

func fillLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}
