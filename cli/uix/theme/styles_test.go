package theme

import (
	"testing"
	"unicode/utf8"
)

func TestTruncatePreservesUTF8(t *testing.T) {
	got := Truncate("你好世界", 2)
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8, got %q", got)
	}
}
