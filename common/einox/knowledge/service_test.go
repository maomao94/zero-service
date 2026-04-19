package knowledge

import "testing"

func TestFormatCitationsBlock(t *testing.T) {
	if s := formatCitationsBlock(nil); s != "" {
		t.Fatalf("empty: %q", s)
	}
	got := formatCitationsBlock([]Citation{
		{Filename: "a.txt", Text: "hello", Score: 0.5},
		{Text: "world", Score: 0.9},
	})
	if got == "" {
		t.Fatal("expected non-empty")
	}
}
