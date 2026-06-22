package model

import "testing"

func TestKmToH3RecallK(t *testing.T) {
	if h3RecallResolution != 9 {
		t.Fatalf("h3RecallResolution = %d, want 9", h3RecallResolution)
	}
	if h3RecallCellType != "h3_r9" {
		t.Fatalf("h3RecallCellType = %q, want h3_r9", h3RecallCellType)
	}

	k := kmToH3RecallK(1)
	if k != 5 {
		t.Fatalf("k = %d, want 5", k)
	}

	k = kmToH3RecallK(50)
	if k != 250 {
		t.Fatalf("k = %d, want 250", k)
	}

	k = kmToH3RecallK(0)
	if k != 1 {
		t.Fatalf("k = %d, want 1", k)
	}
}
