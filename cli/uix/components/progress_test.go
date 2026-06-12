package components

import (
	"testing"
)

func TestProgressPercent(t *testing.T) {
	p := NewProgress(80)
	if p.Percent() != 0 {
		t.Error("initial percent should be 0")
	}

	p.SetPercent(0.5)
	if p.Percent() != 0.5 {
		t.Errorf("expected 0.5, got %f", p.Percent())
	}

	// Test bounds
	p.SetPercent(-0.1)
	if p.Percent() != 0 {
		t.Error("negative percent should clamp to 0")
	}

	p.SetPercent(1.5)
	if p.Percent() != 1 {
		t.Error("percent > 1 should clamp to 1")
	}
}

func TestProgressView(t *testing.T) {
	p := NewProgress(80)
	p.SetPercent(0.5)

	view := p.View()
	if view == "" {
		t.Error("progress View() should not be empty")
	}
}

func TestProgressSetSize(t *testing.T) {
	p := NewProgress(80)
	p.SetSize(100, 10)
	if p.width != 100 {
		t.Errorf("expected width 100, got %d", p.width)
	}

	// Test safe default
	p.SetSize(0, 0)
	if p.width != 80 {
		t.Errorf("expected width 80 for zero input, got %d", p.width)
	}
}
