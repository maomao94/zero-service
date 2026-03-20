package antsx_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
	"zero-service/common/antsx"
)

// ======================== PendingRegistry ========================

func TestPendingRegistry_RegisterResolve(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	p, err := reg.Register("req-1")
	if err != nil {
		t.Fatal(err)
	}

	if !reg.Has("req-1") {
		t.Fatal("expected Has to return true")
	}
	if reg.Len() != 1 {
		t.Fatalf("expected Len=1, got %d", reg.Len())
	}

	// 模拟异步响应
	go func() {
		time.Sleep(50 * time.Millisecond)
		reg.Resolve("req-1", "response-1")
	}()

	ctx := context.Background()
	val, err := p.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val != "response-1" {
		t.Fatalf("expected 'response-1', got '%s'", val)
	}

	if reg.Has("req-1") {
		t.Fatal("expected Has to return false after resolve")
	}
	if reg.Len() != 0 {
		t.Fatalf("expected Len=0, got %d", reg.Len())
	}
}

func TestPendingRegistry_RegisterReject(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	p, err := reg.Register("req-2")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		reg.Reject("req-2", errors.New("remote error"))
	}()

	ctx := context.Background()
	_, err = p.Await(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "remote error" {
		t.Fatalf("expected 'remote error', got '%v'", err)
	}
}

func TestPendingRegistry_AutoExpiry(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	p, err := reg.Register("expire-1", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = p.Await(ctx)
	if !errors.Is(err, antsx.ErrPendingExpired) {
		t.Fatalf("expected ErrPendingExpired, got %v", err)
	}

	if reg.Len() != 0 {
		t.Fatalf("expected Len=0 after expiry, got %d", reg.Len())
	}
}

func TestPendingRegistry_ResolveBeforeExpiry(t *testing.T) {
	reg := antsx.NewPendingRegistry[int]()
	defer reg.Close()

	p, err := reg.Register("fast-1", 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	// 50ms 后 Resolve，远早于 500ms 的 TTL
	go func() {
		time.Sleep(50 * time.Millisecond)
		reg.Resolve("fast-1", 42)
	}()

	ctx := context.Background()
	val, err := p.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}
}

func TestPendingRegistry_DuplicateID(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	_, err := reg.Register("dup-1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = reg.Register("dup-1")
	if !errors.Is(err, antsx.ErrDuplicateID) {
		t.Fatalf("expected ErrDuplicateID, got %v", err)
	}
}

func TestPendingRegistry_Close(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()

	p1, _ := reg.Register("close-1")
	p2, _ := reg.Register("close-2")
	p3, _ := reg.Register("close-3")

	reg.Close()

	ctx := context.Background()
	for i, p := range []*antsx.Promise[string]{p1, p2, p3} {
		_, err := p.Await(ctx)
		if !errors.Is(err, antsx.ErrRegistryClosed) {
			t.Fatalf("promise %d: expected ErrRegistryClosed, got %v", i, err)
		}
	}
}

func TestPendingRegistry_CloseIdempotent(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	reg.Close()
	reg.Close() // 不应 panic
}

func TestPendingRegistry_RegisterAfterClose(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	reg.Close()

	_, err := reg.Register("after-close")
	if !errors.Is(err, antsx.ErrRegistryClosed) {
		t.Fatalf("expected ErrRegistryClosed, got %v", err)
	}
}

func TestPendingRegistry_Has(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	if reg.Has("no-exist") {
		t.Fatal("expected false for non-existent ID")
	}

	reg.Register("has-1")

	if !reg.Has("has-1") {
		t.Fatal("expected true after register")
	}

	reg.Resolve("has-1", "done")

	if reg.Has("has-1") {
		t.Fatal("expected false after resolve")
	}
}

func TestPendingRegistry_ConcurrentResolve(t *testing.T) {
	reg := antsx.NewPendingRegistry[int]()
	defer reg.Close()

	const n = 100
	promises := make([]*antsx.Promise[int], n)

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("concurrent-%d", i)
		p, err := reg.Register(id, 5*time.Second)
		if err != nil {
			t.Fatalf("register %s failed: %v", id, err)
		}
		promises[i] = p
	}

	// 并发 Resolve
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("concurrent-%d", idx)
			reg.Resolve(id, idx*10)
		}(i)
	}
	wg.Wait()

	// 验证结果
	ctx := context.Background()
	for i := 0; i < n; i++ {
		val, err := promises[i].Await(ctx)
		if err != nil {
			t.Fatalf("promise %d error: %v", i, err)
		}
		if val != i*10 {
			t.Fatalf("promise %d: expected %d, got %d", i, i*10, val)
		}
	}

	if reg.Len() != 0 {
		t.Fatalf("expected Len=0, got %d", reg.Len())
	}
}

func TestPendingRegistry_ResolveNonExistent(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	ok := reg.Resolve("ghost", "value")
	if ok {
		t.Fatal("expected false for non-existent ID")
	}
}

// ======================== RequestReply ========================

func TestRequestReply_Success(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	ctx := context.Background()

	// 模拟：sendFn 发送后，外部异步 Resolve
	go func() {
		// 等待 Register 完成
		time.Sleep(50 * time.Millisecond)
		reg.Resolve("rr-1", "reply-data")
	}()

	val, err := antsx.RequestReply(ctx, reg, "rr-1", func() error {
		return nil // 发送成功
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "reply-data" {
		t.Fatalf("expected 'reply-data', got '%s'", val)
	}
}

func TestRequestReply_SendFail(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	ctx := context.Background()
	sendErr := errors.New("send failed")

	_, err := antsx.RequestReply(ctx, reg, "rr-fail", func() error {
		return sendErr
	})
	if err == nil || err.Error() != "send failed" {
		t.Fatalf("expected 'send failed', got %v", err)
	}

	// 确认 entry 已被清理
	if reg.Has("rr-fail") {
		t.Fatal("expected entry to be cleaned up after send failure")
	}
}

func TestRequestReply_CtxCancel(t *testing.T) {
	reg := antsx.NewPendingRegistry[string]()
	defer reg.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// sendFn 成功但没有人 Resolve，ctx 超时
	_, err := antsx.RequestReply(ctx, reg, "rr-timeout", func() error {
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestPendingRegistry_WithDefaultTTL(t *testing.T) {
	reg := antsx.NewPendingRegistry[string](antsx.WithDefaultTTL(200 * time.Millisecond))
	defer reg.Close()

	p, err := reg.Register("ttl-default")
	if err != nil {
		t.Fatal(err)
	}

	// 不 Resolve，等待默认 TTL 过期
	ctx := context.Background()
	_, err = p.Await(ctx)
	if !errors.Is(err, antsx.ErrPendingExpired) {
		t.Fatalf("expected ErrPendingExpired, got %v", err)
	}
}

func TestPendingRegistry_OverrideTTL(t *testing.T) {
	// defaultTTL=30s 但 Register 传 100ms
	reg := antsx.NewPendingRegistry[string](antsx.WithDefaultTTL(30 * time.Second))
	defer reg.Close()

	p, err := reg.Register("ttl-override", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = p.Await(ctx)
	if !errors.Is(err, antsx.ErrPendingExpired) {
		t.Fatalf("expected ErrPendingExpired, got %v", err)
	}
}
