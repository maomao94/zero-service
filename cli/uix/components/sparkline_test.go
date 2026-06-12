package components

import (
	"strings"
	"testing"
)

func TestNewSparkline(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		wantW  int
		wantH  int
	}{
		{"normal", 30, 8, 30, 8},
		{"zero width", 0, 5, 20, 5},
		{"zero height", 20, 0, 20, 5},
		{"negative", -1, -1, 20, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := NewSparkline(tt.width, tt.height)
			if sl.width != tt.wantW {
				t.Errorf("width = %d, want %d", sl.width, tt.wantW)
			}
			if sl.height != tt.wantH {
				t.Errorf("height = %d, want %d", sl.height, tt.wantH)
			}
		})
	}
}

func TestSparklineSetData(t *testing.T) {
	sl := NewSparkline(20, 5)
	sl.SetData([]float64{1, 2, 3, 4, 5})

	view := sl.View()
	if view == "" {
		t.Error("View() returned empty string after SetData")
	}
}

func TestSparklineAddData(t *testing.T) {
	sl := NewSparkline(20, 5)
	sl.AddData(1, 2, 3)
	sl.AddData(4, 5)

	view := sl.View()
	if view == "" {
		t.Error("View() returned empty string after AddData")
	}
}

func TestSparklineClear(t *testing.T) {
	sl := NewSparkline(20, 5)
	sl.SetData([]float64{1, 2, 3})
	sl.Clear()

	// After clear, view should still be renderable
	view := sl.View()
	if view == "" {
		t.Error("View() returned empty string after Clear")
	}
}

func TestSparklineSetSize(t *testing.T) {
	sl := NewSparkline(20, 5)
	sl.SetData([]float64{1, 2, 3, 4, 5})

	sl.SetSize(30, 10)
	if sl.width != 30 {
		t.Errorf("width = %d, want 30", sl.width)
	}
	if sl.height != 10 {
		t.Errorf("height = %d, want 10", sl.height)
	}

	view := sl.View()
	if view == "" {
		t.Error("View() returned empty string after SetSize")
	}
}

func TestSparklineSetSizeSafeDefaults(t *testing.T) {
	sl := NewSparkline(20, 5)
	sl.SetSize(-1, -1)

	if sl.width != 20 {
		t.Errorf("width = %d, want 20", sl.width)
	}
	if sl.height != 5 {
		t.Errorf("height = %d, want 5", sl.height)
	}
}

func TestSparklineViewNotEmpty(t *testing.T) {
	sl := NewSparkline(20, 5)
	sl.SetData([]float64{10, 20, 30, 40, 50})

	view := sl.View()
	if strings.TrimSpace(view) == "" {
		t.Error("View() returned blank string")
	}
}
