package memory

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStorageSaveAndGet(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	msg := &ConversationMessage{
		UserID:    "u1",
		SessionID: "s1",
		Content:   "hello",
		Role:      "user",
	}
	if err := s.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage() error = %v", err)
	}
	if msg.ID == "" {
		t.Fatal("SaveMessage() did not assign ID")
	}
	if msg.CreatedAt.IsZero() {
		t.Fatal("SaveMessage() did not set CreatedAt")
	}

	msgs, err := s.GetMessages(ctx, "u1", "s1", 0)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("GetMessages() = %d, want 1", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("got content %q, want %q", msgs[0].Content, "hello")
	}
}

func TestMemoryStorageGetReturnsSliceCopy(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	if err := s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s1", Content: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s1", Content: "second"}); err != nil {
		t.Fatal(err)
	}

	msgs, _ := s.GetMessages(ctx, "u1", "s1", 0)
	// Append to returned slice should not affect original
	msgs = append(msgs, &ConversationMessage{Content: "injected"})

	msgs2, _ := s.GetMessages(ctx, "u1", "s1", 0)
	if len(msgs2) != 2 {
		t.Fatalf("GetMessages() after append = %d, want 2 (original unmodified)", len(msgs2))
	}
}

func TestMemoryStorageGetMessagesLimit(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s1", Content: "msg"}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Microsecond)
	}

	msgs, err := s.GetMessages(ctx, "u1", "s1", 3)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("GetMessages(limit=3) = %d, want 3", len(msgs))
	}
}

func TestMemoryStorageGetMessagesNoMatch(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	msgs, err := s.GetMessages(ctx, "nonexistent", "session", 0)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("GetMessages() = %d, want 0", len(msgs))
	}
}

func TestMemoryStorageDeleteSession(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	if err := s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s1", Content: "msg"}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSession(ctx, "u1", "s1"); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	msgs, _ := s.GetMessages(ctx, "u1", "s1", 0)
	if len(msgs) != 0 {
		t.Fatal("DeleteSession() did not remove messages")
	}
}

func TestMemoryStorageDeleteSessionNoop(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	if err := s.DeleteSession(ctx, "ghost", "session"); err != nil {
		t.Fatalf("DeleteSession() nonexistent error = %v", err)
	}
}

func TestMemoryStorageSaveNilMessage(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	if err := s.SaveMessage(ctx, nil); err != nil {
		t.Fatalf("SaveMessage(nil) error = %v", err)
	}
}

func TestMemoryStorageClose(t *testing.T) {
	s := NewMemoryStorage()
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMemoryStorageSessionIsolation(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	if err := s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s1", Content: "msg1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessage(ctx, &ConversationMessage{UserID: "u2", SessionID: "s1", Content: "msg2"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s2", Content: "msg3"}); err != nil {
		t.Fatal(err)
	}

	msgs, _ := s.GetMessages(ctx, "u1", "s1", 0)
	if len(msgs) != 1 || msgs[0].Content != "msg1" {
		t.Fatalf("u1/s1 = %d msgs, want 1 msg1", len(msgs))
	}
}

func TestNewStorageMemoryType(t *testing.T) {
	s, err := NewStorage(Config{Type: TypeMemory})
	if err != nil {
		t.Fatalf("NewStorage(memory) error = %v", err)
	}
	if s == nil {
		t.Fatal("NewStorage(memory) returned nil")
	}
	_ = s.Close()
}

func TestNewStorageEmptyType(t *testing.T) {
	s, err := NewStorage(Config{})
	if err != nil {
		t.Fatalf("NewStorage(empty) error = %v", err)
	}
	if s == nil {
		t.Fatal("NewStorage(empty) returned nil")
	}
	_ = s.Close()
}

func TestNewStorageJSONLRequiresBaseDir(t *testing.T) {
	_, err := NewStorage(Config{Type: TypeJSONL})
	if err == nil {
		t.Fatal("expected error for jsonl without BaseDir")
	}
}

func TestNewStorageGormxReturnsError(t *testing.T) {
	_, err := NewStorage(Config{Type: TypeGORMX})
	if err == nil {
		t.Fatal("expected error for gormx without db")
	}
}

func TestMemoryStorageConcurrentSafe(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			_ = s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s1"})
			_, _ = s.GetMessages(ctx, "u1", "s1", 0)
		}
		close(done)
	}()
	for i := 0; i < 50; i++ {
		_ = s.SaveMessage(ctx, &ConversationMessage{UserID: "u1", SessionID: "s1"})
		_, _ = s.GetMessages(ctx, "u1", "s1", 0)
	}
	<-done
}

func TestSanitizeFilename(t *testing.T) {
	if got := sanitizeFilename(""); got != "_" {
		t.Fatalf("sanitize empty: got %q, want _", got)
	}
	if got := sanitizeFilename("normal.txt"); got != "normal.txt" {
		t.Fatalf("sanitize normal: got %q", got)
	}
	if got := sanitizeFilename("a/b:c*d"); got != "a_b_c_d" {
		t.Fatalf("sanitize special: got %q", got)
	}
}
