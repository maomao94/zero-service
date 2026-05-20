package knowledge

import (
	"errors"
	"testing"
)

func TestNewServiceDisabledReturnsSentinel(t *testing.T) {
	svc, err := NewService(Config{Enabled: false}, "key")
	if !errors.Is(err, ErrKnowledgeDisabled) {
		t.Fatalf("NewService disabled: got err=%v, want ErrKnowledgeDisabled", err)
	}
	if svc != nil {
		t.Fatalf("NewService disabled: got svc=%v, want nil", svc)
	}
}

func TestNewServiceDisabledWithEmptyConfig(t *testing.T) {
	svc, err := NewService(Config{}, "")
	if !errors.Is(err, ErrKnowledgeDisabled) {
		t.Fatalf("NewService zero config: got err=%v, want ErrKnowledgeDisabled", err)
	}
	if svc != nil {
		t.Fatalf("NewService zero config: got svc=%v, want nil", svc)
	}
}

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
