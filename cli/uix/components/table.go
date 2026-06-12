package components

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// Column represents a table column definition.
type Column struct {
	Title string
	Width int
}

// Row represents a table row as a slice of string values.
type Row = table.Row

// Table wraps bubbles/table with project theme styling.
type Table struct {
	model  table.Model
	width  int
	height int
}

// NewTable creates a new Table with the given columns, rows, and width.
// Columns MUST be initialized before rows are set (Bubble Tea gotcha).
func NewTable(columns []Column, rows []Row, width int) Table {
	if width <= 0 {
		width = 80
	}
	cols := make([]table.Column, len(columns))
	for i, col := range columns {
		cols[i] = table.Column{
			Title: col.Title,
			Width: max(10, col.Width),
		}
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithStyles(tableStyles()),
	)
	return Table{
		model: t,
		width: width,
	}
}

// SetRows replaces all rows in the table.
func (t *Table) SetRows(rows []Row) {
	t.model.SetRows(rows)
}

// SetColumns replaces all columns in the table.
func (t *Table) SetColumns(columns []Column) {
	cols := make([]table.Column, len(columns))
	for i, col := range columns {
		cols[i] = table.Column{
			Title: col.Title,
			Width: max(10, col.Width),
		}
	}
	t.model.SetColumns(cols)
}

// Selected returns the currently selected row and whether a row is selected.
func (t Table) Selected() (Row, bool) {
	row := t.model.SelectedRow()
	if row == nil {
		return nil, false
	}
	return row, true
}

// Cursor returns the current cursor position.
func (t Table) Cursor() int {
	return t.model.Cursor()
}

// SetCursor sets the cursor position.
func (t *Table) SetCursor(pos int) {
	t.model.SetCursor(pos)
}

// Len returns the number of rows.
func (t Table) Len() int {
	return len(t.model.Rows())
}

// Update processes table messages.
func (t Table) Update(msg tea.Msg) (Table, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return t, cmd
}

// View renders the table.
func (t Table) View() string {
	return t.model.View()
}

// SetSize updates the table dimensions and recalculates column widths.
func (t *Table) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	t.width = width
	t.height = height
	t.model.SetWidth(width)
	t.model.SetHeight(height)
}

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.ColorBorder)).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color(theme.ColorAccent))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(theme.ColorFg)).
		Background(lipgloss.Color(theme.ColorSelected)).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(lipgloss.Color(theme.ColorFg))
	return s
}
