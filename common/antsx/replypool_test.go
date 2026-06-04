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

// ======================== ReplyPool ========================

func TestReplyPool_RegisterResolve(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
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

func TestReplyPool_RegisterReject(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
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

func TestReplyPool_AutoExpiry(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
	defer reg.Close()

	p, err := reg.Register("expire-1", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = p.Await(ctx)
	if !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("expected ErrReplyExpired, got %v", err)
	}

	if reg.Len() != 0 {
		t.Fatalf("expected Len=0 after expiry, got %d", reg.Len())
	}
}

func TestReplyPool_ResolveBeforeExpiry(t *testing.T) {
	reg := antsx.NewReplyPool[int]()
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

func TestReplyPool_DuplicateID(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
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

func TestReplyPool_Close(t *testing.T) {
	reg := antsx.NewReplyPool[string]()

	p1, _ := reg.Register("close-1")
	p2, _ := reg.Register("close-2")
	p3, _ := reg.Register("close-3")

	reg.Close()

	ctx := context.Background()
	for i, p := range []*antsx.Promise[string]{p1, p2, p3} {
		_, err := p.Await(ctx)
		if !errors.Is(err, antsx.ErrReplyClosed) {
			t.Fatalf("promise %d: expected ErrReplyClosed, got %v", i, err)
		}
	}
}

func TestReplyPool_CloseIdempotent(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
	reg.Close()
	reg.Close() // 不应 panic
}

func TestReplyPool_RegisterAfterClose(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
	reg.Close()

	_, err := reg.Register("after-close")
	if !errors.Is(err, antsx.ErrReplyClosed) {
		t.Fatalf("expected ErrReplyClosed, got %v", err)
	}
}

func TestReplyPool_Has(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
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

func TestReplyPool_ConcurrentResolve(t *testing.T) {
	reg := antsx.NewReplyPool[int]()
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

func TestReplyPool_ResolveNonExistent(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
	defer reg.Close()

	ok := reg.Resolve("ghost", "value")
	if ok {
		t.Fatal("expected false for non-existent ID")
	}
}

// ======================== RequestReply ========================

func TestRequestReply_Success(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
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
	reg := antsx.NewReplyPool[string]()
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
	reg := antsx.NewReplyPool[string]()
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

func TestReplyPool_WithDefaultTTL(t *testing.T) {
	reg := antsx.NewReplyPool[string](antsx.WithDefaultTTL(200 * time.Millisecond))
	defer reg.Close()

	p, err := reg.Register("ttl-default")
	if err != nil {
		t.Fatal(err)
	}

	// 不 Resolve，等待默认 TTL 过期
	ctx := context.Background()
	_, err = p.Await(ctx)
	if !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("expected ErrReplyExpired, got %v", err)
	}
}

func TestReplyPool_OverrideTTL(t *testing.T) {
	// defaultTTL=30s 但 Register 传 100ms
	reg := antsx.NewReplyPool[string](antsx.WithDefaultTTL(30 * time.Second))
	defer reg.Close()

	p, err := reg.Register("ttl-override", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = p.Await(ctx)
	if !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("expected ErrReplyExpired, got %v", err)
	}
}

func TestReplyPool_ConcurrentRegisterClose(t *testing.T) {
	reg := antsx.NewReplyPool[int]()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("race-%d", idx)
			p, err := reg.Register(id, time.Second)
			if err != nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			p.Await(ctx)
		}(i)
	}

	time.Sleep(20 * time.Millisecond)
	reg.Close()
	wg.Wait()
}

func TestRequestReply_TTL(t *testing.T) {
	reg := antsx.NewReplyPool[string](antsx.WithDefaultTTL(100 * time.Millisecond))
	defer reg.Close()

	ctx := context.Background()
	_, err := antsx.RequestReply(ctx, reg, "rr-ttl", func() error {
		return nil
	})
	if !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("expected ErrReplyExpired, got %v", err)
	}
}

func TestReplyPool_RejectNonExistent(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
	defer reg.Close()

	ok := reg.Reject("ghost", errors.New("nope"))
	if ok {
		t.Fatal("expected false for non-existent ID")
	}
}

func TestReplyPool_MassiveRegisterResolve(t *testing.T) {
	reg := antsx.NewReplyPool[int]()
	defer reg.Close()

	const n = 500
	promises := make([]*antsx.Promise[int], n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("mass-%d", i)
		p, err := reg.Register(id, 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		promises[i] = p

		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			time.Sleep(time.Duration(idx%10) * time.Millisecond)
			reg.Resolve(id, idx)
		}(i, id)
	}

	wg.Wait()

	ctx := context.Background()
	for i := 0; i < n; i++ {
		val, err := promises[i].Await(ctx)
		if err != nil {
			t.Fatalf("promise %d: %v", i, err)
		}
		if val != i {
			t.Fatalf("promise %d: expected %d, got %d", i, i, val)
		}
	}
}

func TestReplyPool_WithTimingWheel(t *testing.T) {
	reg := antsx.NewReplyPool[string](
		antsx.WithDefaultTTL(200 * time.Millisecond),
	)
	defer reg.Close()

	p, err := reg.Register("tw-1")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)

	_, err = p.Await(context.Background())
	if !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("expected ErrReplyExpired with auto TimingWheel, got: %v", err)
	}
}

func TestReplyPool_RegisterResolveBeforeTimeout(t *testing.T) {
	reg := antsx.NewReplyPool[string](
		antsx.WithDefaultTTL(500 * time.Millisecond),
	)
	defer reg.Close()

	p, err := reg.Register("fast-1")
	if err != nil {
		t.Fatal(err)
	}

	reg.Resolve("fast-1", "hello")

	val, err := p.Await(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %q", val)
	}
}

func TestRequestReply_CtxTimeout(t *testing.T) {
	reg := antsx.NewReplyPool[string](
		antsx.WithDefaultTTL(5 * time.Second),
	)
	defer reg.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := antsx.RequestReply(ctx, reg, "timeout-1", func() error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error from ctx timeout")
	}
}

func TestRequestReply_CustomTTL(t *testing.T) {
	reg := antsx.NewReplyPool[string](
		antsx.WithDefaultTTL(30 * time.Second),
	)
	defer reg.Close()

	ctx := context.Background()

	_, err := antsx.RequestReply(ctx, reg, "custom-ttl", func() error {
		return nil
	}, 100*time.Millisecond)
	if !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("expected ErrReplyExpired with custom TTL, got %v", err)
	}
}

func TestReplyPool_Lifecycle(t *testing.T) {
	reg := antsx.NewReplyPool[string](antsx.WithDefaultTTL(200 * time.Millisecond))
	defer reg.Close()

	// 初始状态为空
	if reg.Len() != 0 {
		t.Fatalf("expected empty pool, got Len=%d", reg.Len())
	}

	// Register
	p1, _ := reg.Register("s1")
	p2, _ := reg.Register("s2")
	p3, _ := reg.Register("s3", 100*time.Millisecond)
	if reg.Len() != 3 {
		t.Fatalf("expected Len=3, got %d", reg.Len())
	}
	if !reg.Has("s1") || !reg.Has("s2") || !reg.Has("s3") {
		t.Fatal("expected Has=true for all IDs")
	}

	// Resolve
	reg.Resolve("s1", "ok")
	v, err := p1.Await(context.Background())
	if err != nil || v != "ok" {
		t.Fatalf("expected resolved, got val=%v err=%v", v, err)
	}
	if reg.Len() != 2 || reg.Has("s1") {
		t.Fatalf("expected Len=2 Has(s1)=false after resolve, got Len=%d Has=%v", reg.Len(), reg.Has("s1"))
	}

	// Reject
	reg.Reject("s2", errors.New("fail"))
	_, err = p2.Await(context.Background())
	if err == nil {
		t.Fatal("expected reject error, got nil")
	}
	if reg.Len() != 1 || reg.Has("s2") {
		t.Fatalf("expected Len=1 Has(s2)=false after reject, got Len=%d Has=%v", reg.Len(), reg.Has("s2"))
	}

	// Expire
	time.Sleep(300 * time.Millisecond)
	_, err = p3.Await(context.Background())
	if !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("expected ErrReplyExpired, got %v", err)
	}
	if reg.Len() != 0 || reg.Has("s3") {
		t.Fatalf("expected Len=0 Has(s3)=false after expire, got Len=%d Has=%v", reg.Len(), reg.Has("s3"))
	}
}

func TestReplyPool_HandleTimeoutVsRejectRace(t *testing.T) {
	reg := antsx.NewReplyPool[string](
		antsx.WithDefaultTTL(50 * time.Millisecond),
	)
	defer reg.Close()

	const n = 100
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("race-%d", i)
		reg.Register(id)
	}

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("race-%d", idx)
			time.Sleep(time.Duration(idx%5) * time.Millisecond)
			reg.Reject(id, errors.New("reject"))
		}(i)
	}

	time.Sleep(100 * time.Millisecond)
	wg.Wait()

	// 所有条目都应被处理（reject 或 expire），pending 必须归零
	if reg.Len() != 0 {
		t.Fatalf("expected Len=0 after all done, got %d", reg.Len())
	}
	t.Logf("race test passed: pending=%d", reg.Len())
}

func TestReplyPool_CloseStatsAccuracy(t *testing.T) {
	reg := antsx.NewReplyPool[string]()
	defer reg.Close()

	promises := make([]*antsx.Promise[string], 10)
	for i := 0; i < 10; i++ {
		p, _ := reg.Register(fmt.Sprintf("close-%d", i))
		promises[i] = p
	}

	reg.Resolve("close-0", "ok")
	reg.Reject("close-1", errors.New("fail"))

	if reg.Len() != 8 {
		t.Fatalf("before close: expected Len=8, got %d", reg.Len())
	}

	reg.Close()

	if reg.Len() != 0 {
		t.Fatalf("after close: expected Len=0, got %d", reg.Len())
	}

	// 剩余的 8 个应收到 ErrReplyClosed
	closedCount := 0
	for _, p := range promises {
		_, err := p.Await(context.Background())
		if errors.Is(err, antsx.ErrReplyClosed) {
			closedCount++
		}
	}
	if closedCount != 8 {
		t.Fatalf("expected 8 ErrReplyClosed, got %d", closedCount)
	}
}

func TestReplyPool_StatLoopLifecycle(t *testing.T) {
	// 验证默认内置统计循环在 Close 时正确停止，不发生死锁
	reg := antsx.NewReplyPool[string]()

	// 做几笔操作让循环有数据可输出
	reg.Register("lifecycle-1")
	reg.Resolve("lifecycle-1", "done")

	// Close 应停止 statLoop goroutine
	reg.Close()
}

func TestReplyPool_ConcurrentRegisterResolveStats(t *testing.T) {
	reg := antsx.NewReplyPool[int]()
	defer reg.Close()

	const n = 200
	var registerDone sync.WaitGroup
	registerDone.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			defer registerDone.Done()
			id := fmt.Sprintf("concurrent-%d", idx)
			reg.Register(id, 5*time.Second)
		}(i)
	}

	registerDone.Wait()

	var resolveDone sync.WaitGroup
	resolveDone.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer resolveDone.Done()
			id := fmt.Sprintf("concurrent-%d", idx)
			reg.Resolve(id, idx)
		}(i)
	}

	resolveDone.Wait()

	if reg.Len() != 0 {
		t.Fatalf("expected Len=0 after all resolved, got %d", reg.Len())
	}
}
