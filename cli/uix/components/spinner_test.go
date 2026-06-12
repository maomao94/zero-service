package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSpinnerLifecycle(t *testing.T) {
	s := NewSpinner()
	if s.IsActive() {
		t.Error("spinner should not be active initially")
	}

	s.Start()
	if !s.IsActive() {
		t.Error("spinner should be active after Start()")
	}

	view := s.View()
	if view == "" {
		t.Error("spinner View() should not be empty when active")
	}

	s.Stop()
	if s.IsActive() {
		t.Error("spinner should not be active after Stop()")
	}

	view = s.View()
	if view != "" {
		t.Error("spinner View() should be empty when inactive")
	}
}

func TestSpinnerUpdate(t *testing.T) {
	s := NewSpinner()
	s.Start()

	// Spinner tick messages should be handled
	s2, _ := s.Update(tea.KeyMsg{})
	if s2.IsActive() != s.IsActive() {
		t.Error("spinner state should not change on non-tick messages")
	}
}

func TestSpinnerSetSize(t *testing.T) {
	s := NewSpinner()
	s.SetSize(100, 10)
	if s.width != 100 {
		t.Errorf("expected width 100, got %d", s.width)
	}

	// Test safe default
	s.SetSize(0, 0)
	if s.width != 80 {
		t.Errorf("expected width 80 for zero input, got %d", s.width)
	}
}
