package components

import (
	"testing"
)

func TestHelpBindings(t *testing.T) {
	h := NewHelp()
	if h.BindingCount() != 0 {
		t.Error("new help should have 0 bindings")
	}

	h.AddBinding(HelpBinding{
		Keys: []string{"q"},
		Desc: "quit",
	})
	if h.BindingCount() != 1 {
		t.Errorf("expected 1 binding, got %d", h.BindingCount())
	}

	h.AddBinding(HelpBinding{
		Keys: []string{"ctrl+c"},
		Desc: "force quit",
	})
	if h.BindingCount() != 2 {
		t.Errorf("expected 2 bindings, got %d", h.BindingCount())
	}
}

func TestHelpSetBindings(t *testing.T) {
	h := NewHelp()
	bindings := []HelpBinding{
		{Keys: []string{"j"}, Desc: "down"},
		{Keys: []string{"k"}, Desc: "up"},
		{Keys: []string{"q"}, Desc: "quit"},
	}

	h.SetBindings(bindings)
	if h.BindingCount() != 3 {
		t.Errorf("expected 3 bindings, got %d", h.BindingCount())
	}
}

func TestHelpView(t *testing.T) {
	h := NewHelp()

	// Empty bindings should produce empty view
	view := h.View()
	if view != "" {
		t.Error("empty help should produce empty view")
	}

	h.AddBinding(HelpBinding{
		Keys: []string{"q"},
		Desc: "quit",
	})
	view = h.View()
	if view == "" {
		t.Error("help with bindings should produce non-empty view")
	}
}

func TestHelpToggle(t *testing.T) {
	h := NewHelp()
	if h.model.ShowAll {
		t.Error("help should start with short view")
	}

	h.ToggleHelp()
	if !h.model.ShowAll {
		t.Error("help should show full view after toggle")
	}

	h.ToggleHelp()
	if h.model.ShowAll {
		t.Error("help should show short view after second toggle")
	}
}

func TestHelpSetSize(t *testing.T) {
	h := NewHelp()
	h.SetSize(100, 10)
	if h.width != 100 {
		t.Errorf("expected width 100, got %d", h.width)
	}

	// Test safe default
	h.SetSize(0, 0)
	if h.width != 80 {
		t.Errorf("expected width 80 for zero input, got %d", h.width)
	}
}
