package types

import "testing"

func TestIoaHexAddress(t *testing.T) {
	tests := []struct {
		name string
		ioa  uint
		want string
	}{
		{name: "zero", ioa: 0, want: "0x0000"},
		{name: "one", ioa: 1, want: "0x0001"},
		{name: "three bytes", ioa: 0x123456, want: "0x123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IoaHexAddress(tt.ioa); got != tt.want {
				t.Fatalf("IoaHexAddress(%d) = %q, want %q", tt.ioa, got, tt.want)
			}
		})
	}
}
