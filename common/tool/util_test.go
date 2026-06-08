package tool

import "testing"

func TestCountSignificantDigits(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"51.88", 4},
		{"0.001234", 4},
		{"100", 3},
		{"0", 0},
		{"0.0", 0},
		{"-51.88", 4},
		{"+100.0", 4},
		{"51.879791", 8},
		{"1.23e5", 3},
		{"0.0001", 1},
		{"1234567", 7},
		{"12345678", 8},
		{" 51.88 ", 4},
	}
	for _, tt := range tests {
		got := CountSignificantDigits(tt.input)
		if got != tt.want {
			t.Errorf("CountSignificantDigits(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
