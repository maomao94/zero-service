package svc

import "testing"

func TestSessionAgentModeStore(t *testing.T) {
	s := &ServiceContext{}
	s.SetSessionAgentMode("u1", "s1", "deep")

	got := s.GetSessionAgentMode("u1", "s1")
	if got != "deep" {
		t.Fatalf("expected deep, got %s", got)
	}
}
