package tool

import "testing"

func TestHexBytes(t *testing.T) {
	raw := []byte{0x68, 0x0e, 0x00, 0xff}

	tests := []struct {
		name   string
		format HexBytesFormat
		want   string
	}{
		{name: "lower compact", format: HexLowerCompact, want: "680e00ff"},
		{name: "upper compact", format: HexUpperCompact, want: "680E00FF"},
		{name: "upper space", format: HexUpperSpace, want: "68 0E 00 FF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HexBytes(raw, tt.format); got != tt.want {
				t.Fatalf("HexBytes = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHexBytesDefaultsToLowerCompact(t *testing.T) {
	raw := []byte{0x68, 0x0e, 0x00, 0xff}
	got := HexBytes(raw, HexBytesFormat(99))
	want := "680e00ff"
	if got != want {
		t.Fatalf("HexBytes default = %q, want %q", got, want)
	}
}
