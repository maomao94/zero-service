package logic

import (
	"math"
	"testing"
)

func TestGeohashCellSize(t *testing.T) {
	tests := []struct {
		name          string
		precision     int
		wantWidthDeg  float64
		wantHeightDeg float64
	}{
		{name: "precision 1", precision: 1, wantWidthDeg: 45, wantHeightDeg: 45},
		{name: "precision 2", precision: 2, wantWidthDeg: 11.25, wantHeightDeg: 5.625},
		{name: "precision 6", precision: 6, wantWidthDeg: 0.010986328125, wantHeightDeg: 0.0054931640625},
		{name: "precision 7", precision: 7, wantWidthDeg: 0.001373291015625, wantHeightDeg: 0.001373291015625},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widthDeg, heightDeg := geohashCellSize(tt.precision, 39.9)
			if math.Abs(widthDeg-tt.wantWidthDeg) > 1e-12 {
				t.Fatalf("widthDeg = %.15f, want %.15f", widthDeg, tt.wantWidthDeg)
			}
			if math.Abs(heightDeg-tt.wantHeightDeg) > 1e-12 {
				t.Fatalf("heightDeg = %.15f, want %.15f", heightDeg, tt.wantHeightDeg)
			}
		})
	}
}
