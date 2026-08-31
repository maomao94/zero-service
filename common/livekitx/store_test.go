package livekitx

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreTracksSerializableConnectionState(t *testing.T) {
	store := NewMemoryStore()
	state := ConnectionState{NodeID: "node-a", RoomName: "room-a", Identity: "alice", SessionID: "session-a", Status: ConnectionStatusConnected, UpdatedAt: time.Now()}
	if err := store.Put(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Get(context.Background(), state.SessionID)
	if err != nil || !ok {
		t.Fatalf("Get() = %#v, %v, want state", got, err)
	}
	if got != state {
		t.Fatalf("Get() = %#v, want %#v", got, state)
	}
	if err := store.Delete(context.Background(), state.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Get(context.Background(), state.SessionID); err != nil || ok {
		t.Fatalf("deleted state = ok %v, err %v", ok, err)
	}
}

func TestMemoryStoreExpiresAndCloses(t *testing.T) {
	store := NewMemoryStore()
	state := ConnectionState{SessionID: "expired", ExpiresAt: time.Now().Add(-time.Second)}
	if err := store.Put(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Get(context.Background(), state.SessionID); err != nil || ok {
		t.Fatalf("expired state = ok %v, err %v", ok, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), state); err != ErrClosed {
		t.Fatalf("Put after Close() = %v, want ErrClosed", err)
	}
}
