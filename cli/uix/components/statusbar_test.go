package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusBarViewDoesNotExceedConfiguredWidth(t *testing.T) {
	bar := NewStatusBar()
	bar.SetWidth(24)
	bar.SetLeft("very-long-module-name")
	bar.SetRight("x inspect selected resource | s start | r restart | delete remove with confirmation | / commands")

	for _, line := range strings.Split(bar.View(), "\n") {
		if width := lipgloss.Width(line); width > 24 {
			t.Fatalf("expected line width <= 24, got %d for %q", width, line)
		}
	}
}

func TestStatusBarViewUsesSafeDefaultWidth(t *testing.T) {
	bar := NewStatusBar()
	bar.SetWidth(0)
	bar.SetLeft("chat")
	bar.SetRight("enter send")

	lines := strings.Split(bar.View(), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two status lines, got %d", len(lines))
	}
	if width := lipgloss.Width(lines[0]); width != 80 {
		t.Fatalf("expected default border width 80, got %d", width)
	}
}
