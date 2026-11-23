package bytex

import "testing"

func TestInt16SliceToUint16Slice(t *testing.T) {
	cases := []struct {
		name string
		in   []int16
		want []uint16
	}{
		{"空切片", nil, []uint16{}},
		{"正数", []int16{0, 1, 32767}, []uint16{0, 1, 32767}},
		{"负数按补码", []int16{-1, -32768}, []uint16{65535, 32768}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Int16SliceToUint16Slice(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("len = %d, want %d", len(got), len(c.want))
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("got[%d] = %d, want %d", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestInt32SliceToUint32Slice(t *testing.T) {
	got := Int32SliceToUint32Slice([]int32{-1, 0, 2147483647})
	want := []uint32{4294967295, 0, 2147483647}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestInt32SliceToUint16SliceValidate(t *testing.T) {
	t.Run("范围内按补码转换", func(t *testing.T) {
		got, errIdx, err := Int32SliceToUint16SliceValidate([]int32{-32768, -1, 0, 32767})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if errIdx != -1 {
			t.Fatalf("errIdx = %d, want -1", errIdx)
		}
		want := []uint16{32768, 65535, 0, 32767}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("got[%d] = %d, want %d", i, got[i], want[i])
			}
		}
	})

	t.Run("超上限", func(t *testing.T) {
		_, errIdx, err := Int32SliceToUint16SliceValidate([]int32{1, 32768, 2})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errIdx != 1 {
			t.Fatalf("errIdx = %d, want 1", errIdx)
		}
	})

	t.Run("超下限", func(t *testing.T) {
		_, errIdx, err := Int32SliceToUint16SliceValidate([]int32{-32769})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errIdx != 0 {
			t.Fatalf("errIdx = %d, want 0", errIdx)
		}
	})
}

func TestUint16SliceToBytes(t *testing.T) {
	got := Uint16SliceToBytes([]uint16{0x1234, 0x5678})
	want := []byte{0x12, 0x34, 0x56, 0x78}
	if string(got) != string(want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
