package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewCmdBarIsFocused(t *testing.T) {
	cmdbar := NewCmdBar("test > ")
	updated, _ := cmdbar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if updated.Value() != "a" {
		t.Fatalf("expected focused cmdbar to receive key, got %q", updated.Value())
	}
}

func TestDropdownShowsEmptyState(t *testing.T) {
	dropdown := NewDropdown(40, 5)
	dropdown.SetEntries([]DropdownEntry{{Label: "help", Description: "show help", Prefix: "/"}})
	dropdown.Filter("missing")

	if dropdown.Height() != 3 {
		t.Fatalf("expected empty dropdown height 3, got %d", dropdown.Height())
	}
	if !strings.Contains(dropdown.View(), "No matches") {
		t.Fatalf("expected empty dropdown copy, got %q", dropdown.View())
	}
}
