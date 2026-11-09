package confirm

import "testing"

func TestNotifierRoomUsesDeptLevel(t *testing.T) {
	n := NewNotifier(nil, "", "", "")
	if got := n.Room("D001"); got != "dept:D001" {
		t.Fatalf("Room(D001) = %q, want dept:D001", got)
	}
	if got := n.Room(""); got != DefaultFallbackRoom {
		t.Fatalf("Room(empty) = %q, want %q", got, DefaultFallbackRoom)
	}

	custom := NewNotifier(nil, "org-", "fallback", "custom:event")
	if got := custom.Room("D002"); got != "org-D002" {
		t.Fatalf("custom Room(D002) = %q, want org-D002", got)
	}
	if got := custom.Room(""); got != "fallback" {
		t.Fatalf("custom Room(empty) = %q, want fallback", got)
	}
}
