package checkpoint

import (
	"context"
	"testing"
)

func TestMemoryStoreSetGet(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if err := s.Set(ctx, "k1", []byte("value1")); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	v, ok, err := s.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get(): not found")
	}
	if string(v) != "value1" {
		t.Fatalf("Get(): got %q, want %q", string(v), "value1")
	}
}

func TestMemoryStoreGetMissing(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	v, ok, err := s.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Fatal("Get(): expected not found")
	}
	if v != nil {
		t.Fatalf("Get(): expected nil, got %v", v)
	}
}

func TestMemoryStoreOverwrite(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if err := s.Set(ctx, "k1", []byte("original")); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(ctx, "k1", []byte("updated")); err != nil {
		t.Fatal(err)
	}

	v, ok, _ := s.Get(ctx, "k1")
	if !ok || string(v) != "updated" {
		t.Fatalf("after overwrite: got %q, want %q", string(v), "updated")
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if err := s.Set(ctx, "k1", []byte("value1")); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, ok, _ := s.Get(ctx, "k1")
	if ok {
		t.Fatal("Get() after Delete: should not be found")
	}
}

func TestMemoryStoreDeleteMissing(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if err := s.Delete(ctx, "ghost"); err != nil {
		t.Fatalf("Delete() missing key error = %v", err)
	}
}

func TestMemoryStoreClose(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMemoryStoreIsolation(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	original := []byte("sensitive")
	if err := s.Set(ctx, "k1", original); err != nil {
		t.Fatal(err)
	}

	// Modify the original slice after Set
	original[0] = 'X'

	v, _, _ := s.Get(ctx, "k1")
	if string(v) == "Xensitive" {
		t.Fatal("Set() did not copy the input slice")
	}
	if string(v) != "sensitive" {
		t.Fatalf("Get(): got %q, want %q", string(v), "sensitive")
	}

	// Modify the returned slice
	v[0] = 'Y'
	v2, _, _ := s.Get(ctx, "k1")
	if string(v2) == "Yensitive" {
		t.Fatal("Get() did not copy the returned slice")
	}
}

func TestNewStoreMemoryType(t *testing.T) {
	s, err := NewStore(Config{Type: TypeMemory}, nil)
	if err != nil {
		t.Fatalf("NewStore(memory) error = %v", err)
	}
	if s == nil {
		t.Fatal("NewStore(memory) returned nil")
	}
	_ = s.Close()
}

func TestNewStoreEmptyType(t *testing.T) {
	s, err := NewStore(Config{}, nil)
	if err != nil {
		t.Fatalf("NewStore(empty) error = %v", err)
	}
	if s == nil {
		t.Fatal("NewStore(empty) returned nil")
	}
	_ = s.Close()
}

func TestNewStoreJSONLRequiresBaseDir(t *testing.T) {
	_, err := NewStore(Config{Type: TypeJSONL}, nil)
	if err == nil {
		t.Fatal("expected error for jsonl without BaseDir")
	}
}

func TestNewStoreGormxRequiresDB(t *testing.T) {
	_, err := NewStore(Config{Type: TypeGORMX}, nil)
	if err == nil {
		t.Fatal("expected error for gormx without db")
	}
}

func TestNewStoreUnknownType(t *testing.T) {
	_, err := NewStore(Config{Type: Type("unknown")}, nil)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestMemoryStoreConcurrentSafe(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_ = s.Set(ctx, "k", []byte("v"))
			_, _, _ = s.Get(ctx, "k")
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		_ = s.Set(ctx, "k", []byte("v"))
		_, _, _ = s.Get(ctx, "k")
	}
	<-done
}
