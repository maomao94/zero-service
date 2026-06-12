package components

import (
	"strings"
	"testing"
)

func TestNewBarChart(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		wantW  int
		wantH  int
	}{
		{"normal", 50, 15, 50, 15},
		{"zero width", 0, 10, 40, 10},
		{"zero height", 40, 0, 40, 10},
		{"negative", -1, -1, 40, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := NewBarChart(tt.width, tt.height, nil, nil)
			if bc.width != tt.wantW {
				t.Errorf("width = %d, want %d", bc.width, tt.wantW)
			}
			if bc.height != tt.wantH {
				t.Errorf("height = %d, want %d", bc.height, tt.wantH)
			}
		})
	}
}

func TestBarChartWithData(t *testing.T) {
	labels := []string{"A", "B", "C"}
	values := []float64{10, 20, 30}
	bc := NewBarChart(40, 10, labels, values)

	view := bc.View()
	if view == "" {
		t.Error("View() returned empty string after construction with data")
	}
}

func TestBarChartSetData(t *testing.T) {
	bc := NewBarChart(40, 10, nil, nil)

	labels := []string{"X", "Y", "Z"}
	values := []float64{5, 15, 25}
	bc.SetData(labels, values)

	view := bc.View()
	if view == "" {
		t.Error("View() returned empty string after SetData")
	}
}

func TestBarChartClear(t *testing.T) {
	labels := []string{"A", "B"}
	values := []float64{10, 20}
	bc := NewBarChart(40, 10, labels, values)

	bc.Clear()

	view := bc.View()
	if view == "" {
		t.Error("View() returned empty string after Clear")
	}
}

func TestBarChartSetSize(t *testing.T) {
	labels := []string{"A", "B", "C"}
	values := []float64{10, 20, 30}
	bc := NewBarChart(40, 10, labels, values)

	bc.SetSize(60, 20)
	if bc.width != 60 {
		t.Errorf("width = %d, want 60", bc.width)
	}
	if bc.height != 20 {
		t.Errorf("height = %d, want 20", bc.height)
	}

	view := bc.View()
	if view == "" {
		t.Error("View() returned empty string after SetSize")
	}
}

func TestBarChartSetSizeSafeDefaults(t *testing.T) {
	bc := NewBarChart(40, 10, nil, nil)
	bc.SetSize(-1, -1)

	if bc.width != 40 {
		t.Errorf("width = %d, want 40", bc.width)
	}
	if bc.height != 10 {
		t.Errorf("height = %d, want 10", bc.height)
	}
}

func TestBarChartSetHorizontal(t *testing.T) {
	labels := []string{"A", "B", "C"}
	values := []float64{10, 20, 30}
	bc := NewBarChart(40, 10, labels, values)

	bc.SetHorizontal(true)
	view := bc.View()
	if view == "" {
		t.Error("View() returned empty string after SetHorizontal")
	}
}

func TestBarChartSetShowAxis(t *testing.T) {
	labels := []string{"A", "B"}
	values := []float64{10, 20}
	bc := NewBarChart(40, 10, labels, values)

	bc.SetShowAxis(false)
	view := bc.View()
	if view == "" {
		t.Error("View() returned empty string after SetShowAxis(false)")
	}
}

func TestBarChartViewNotEmpty(t *testing.T) {
	labels := []string{"CPU", "Memory", "Disk"}
	values := []float64{45.5, 72.3, 30.1}
	bc := NewBarChart(40, 10, labels, values)

	view := bc.View()
	if strings.TrimSpace(view) == "" {
		t.Error("View() returned blank string")
	}
}
