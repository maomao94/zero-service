package components

import (
	"testing"
)

func TestTableInitialization(t *testing.T) {
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
	}
	rows := []Row{
		{"item1", "active"},
		{"item2", "inactive"},
	}

	table := NewTable(cols, rows, 80)
	if table.Len() != 2 {
		t.Errorf("expected 2 rows, got %d", table.Len())
	}
}

func TestTableSelection(t *testing.T) {
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []Row{
		{"item1"},
		{"item2"},
		{"item3"},
	}

	table := NewTable(cols, rows, 80)
	if table.Cursor() != 0 {
		t.Error("initial cursor should be 0")
	}

	row, ok := table.Selected()
	if !ok || row[0] != "item1" {
		t.Error("should select first row")
	}

	table.SetCursor(2)
	if table.Cursor() != 2 {
		t.Error("cursor should be 2 after SetCursor(2)")
	}
}

func TestTableSetRows(t *testing.T) {
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []Row{{"item1"}}

	table := NewTable(cols, rows, 80)
	if table.Len() != 1 {
		t.Error("should have 1 row")
	}

	table.SetRows([]Row{{"new1"}, {"new2"}})
	if table.Len() != 2 {
		t.Errorf("expected 2 rows after SetRows, got %d", table.Len())
	}
}

func TestTableSetSize(t *testing.T) {
	cols := []Column{{Title: "Name", Width: 20}}
	table := NewTable(cols, nil, 80)

	table.SetSize(100, 25)
	if table.width != 100 {
		t.Errorf("expected width 100, got %d", table.width)
	}

	// Test safe defaults
	table.SetSize(0, 0)
	if table.width != 80 {
		t.Errorf("expected width 80 for zero input, got %d", table.width)
	}
}
