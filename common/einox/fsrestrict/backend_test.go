package fsrestrict

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnderRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "other")
	cases := []struct {
		abs   string
		want  bool
		label string
	}{
		{filepath.Join(root, "a", "b"), true, "child"},
		{root, true, "equal"},
		{outside, false, "outside"},
	}
	for _, tc := range cases {
		if got := underRoot(root, tc.abs); got != tc.want {
			t.Fatalf("%s: underRoot(%q,%q)=%v want %v", tc.label, root, tc.abs, got, tc.want)
		}
	}
}

func TestWrap_nilInner(t *testing.T) {
	if Wrap(nil, []string{"/tmp"}) != nil {
		t.Fatal("expected nil")
	}
}

func TestPolicyBackend_largeToolResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-style /large_tool_result prefix")
	}
	root := t.TempDir()
	b := &policyBackend{
		inner: nil,
		cfg: Config{
			UserRoots: []string{filepath.Clean(root)},
			Policy:    PermissivePolicy(),
		},
	}
	ctx := context.Background()
	got, _, err := b.resolveWithCtx(ctx, "/large_tool_result/call-1")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "large_tool_result", "call-1")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
