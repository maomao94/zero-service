package components

import (
	"testing"
)

func TestTextAreaFocus(t *testing.T) {
	ta := NewTextArea(80, 5)
	if ta.Focused() {
		t.Error("textarea should not be focused initially")
	}

	ta.Focus()
	if !ta.Focused() {
		t.Error("textarea should be focused after Focus()")
	}

	ta.Blur()
	if ta.Focused() {
		t.Error("textarea should not be focused after Blur()")
	}
}

func TestTextAreaValue(t *testing.T) {
	ta := NewTextArea(80, 5)

	ta.SetValue("hello world")
	if ta.Value() != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", ta.Value())
	}

	ta.SetValue("")
	if ta.Value() != "" {
		t.Errorf("expected empty string, got '%s'", ta.Value())
	}
}

func TestTextAreaLineCount(t *testing.T) {
	ta := NewTextArea(80, 5)
	if ta.LineCount() != 0 {
		t.Error("empty textarea should have 0 lines")
	}

	ta.SetValue("line 1")
	if ta.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", ta.LineCount())
	}

	ta.SetValue("line 1\nline 2\nline 3")
	if ta.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", ta.LineCount())
	}
}

func TestTextAreaSetSize(t *testing.T) {
	ta := NewTextArea(80, 5)
	ta.SetSize(100, 10)
	if ta.width != 100 {
		t.Errorf("expected width 100, got %d", ta.width)
	}
	if ta.height != 10 {
		t.Errorf("expected height 10, got %d", ta.height)
	}

	// Test safe defaults
	ta.SetSize(0, 0)
	if ta.width != 80 {
		t.Errorf("expected width 80 for zero input, got %d", ta.width)
	}
	if ta.height != 5 {
		t.Errorf("expected height 5 for zero input, got %d", ta.height)
	}
}
