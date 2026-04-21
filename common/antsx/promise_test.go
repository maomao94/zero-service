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

func TestPromise_ResolveAwait(t *testing.T) {
	p := antsx.NewPromise[int]()
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.Resolve(42)
	}()

	val, err := p.Await(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}
}

func TestPromise_RejectAwait(t *testing.T) {
	p := antsx.NewPromise[string]()
	go func() {
		p.Reject(errors.New("intentional failure"))
	}()

	_, err := p.Await(context.Background())
	if err == nil || err.Error() != "intentional failure" {
		t.Fatalf("expected 'intentional failure', got %v", err)
	}
}

func TestPromise_AwaitContextCancel(t *testing.T) {
	p := antsx.NewPromise[int]()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.Await(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestPromise_Get_Pending(t *testing.T) {
	p := antsx.NewPromise[int]()
	_, _, ok := p.Get()
	if ok {
		t.Fatal("expected ok=false for pending promise")
	}
}

func TestPromise_Get_Resolved(t *testing.T) {
	p := antsx.NewPromise[int]()
	p.Resolve(99)

	val, err, ok := p.Get()
	if !ok {
		t.Fatal("expected ok=true for resolved promise")
	}
	if err != nil || val != 99 {
		t.Fatalf("expected (99, nil), got (%d, %v)", val, err)
	}
}

func TestPromise_Done(t *testing.T) {
	p := antsx.NewPromise[int]()
	select {
	case <-p.Done():
		t.Fatal("Done should not fire before resolve")
	default:
	}

	p.Resolve(1)

	select {
	case <-p.Done():
	default:
		t.Fatal("Done should fire after resolve")
	}
}

func TestPromise_Catch_ErrorCallback(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	caughtCh := make(chan error, 1)
	p.Catch(ctx, func(err error) {
		caughtCh <- err
	})

	p.Reject(errors.New("catch me"))

	select {
	case err := <-caughtCh:
		if err == nil || err.Error() != "catch me" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Catch callback not called within timeout")
	}
}

func TestPromise_Catch_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := antsx.NewPromise[int]()
	called := make(chan struct{}, 1)
	p.Catch(ctx, func(err error) {
		called <- struct{}{}
	})

	cancel()
	time.Sleep(50 * time.Millisecond)

	p.Reject(errors.New("late error"))

	select {
	case <-called:
		t.Fatal("Catch should not be called after ctx cancel")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestPromise_Catch_ResolvedNoCall(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	called := make(chan struct{}, 1)
	p.Catch(ctx, func(err error) {
		called <- struct{}{}
	})

	p.Resolve(42)

	select {
	case <-called:
		t.Fatal("Catch should not be called on successful promise")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestAwaitWithTimeout_Success(t *testing.T) {
	p := antsx.NewPromise[int]()
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.Resolve(99)
	}()

	result, err := p.AwaitWithTimeout(1 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result != 99 {
		t.Fatalf("expected 99 but got %d", result)
	}
}

func TestAwaitWithTimeout_Expired(t *testing.T) {
	p := antsx.NewPromise[int]()

	_, err := p.AwaitWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestFireAndForget_PanicRecovery(t *testing.T) {
	p := antsx.NewPromise[int]()
	p.Resolve(42)
	p.FireAndForget(context.Background())
	time.Sleep(50 * time.Millisecond)
}

func TestThen_Success(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(42) }()

	p2 := antsx.Then(ctx, p, func(v int) (string, error) {
		return fmt.Sprintf("value is %d", v), nil
	})

	res, err := p2.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res != "value is 42" {
		t.Fatalf("unexpected: %s", res)
	}
}

func TestThen_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(42) }()

	p2 := antsx.Then(ctx, p, func(v int) (string, error) {
		panic("then boom")
	})

	_, err := p2.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic")
	}
}

func TestMap_Success(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(10) }()

	mapped := antsx.Map(ctx, p, func(v int) string {
		return fmt.Sprintf("val=%d", v)
	})

	result, err := mapped.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "val=10" {
		t.Fatalf("expected 'val=10' but got '%s'", result)
	}
}

func TestMap_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(10) }()

	mapped := antsx.Map(ctx, p, func(v int) string {
		panic("map boom")
	})

	_, err := mapped.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic")
	}
}

func TestFlatMap_Success(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(5) }()

	flatMapped := antsx.FlatMap(ctx, p, func(v int) *antsx.Promise[string] {
		inner := antsx.NewPromise[string]()
		go func() {
			inner.Resolve(fmt.Sprintf("doubled=%d", v*2))
		}()
		return inner
	})

	result, err := flatMapped.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "doubled=10" {
		t.Fatalf("expected 'doubled=10' but got '%s'", result)
	}
}

func TestFlatMap_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(5) }()

	flatMapped := antsx.FlatMap(ctx, p, func(v int) *antsx.Promise[string] {
		panic("flatmap boom")
	})

	_, err := flatMapped.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic")
	}
}

func TestPromiseAll_AllSuccess(t *testing.T) {
	ctx := context.Background()
	p1 := antsx.NewPromise[int]()
	p2 := antsx.NewPromise[int]()
	p3 := antsx.NewPromise[int]()

	go func() { p1.Resolve(1) }()
	go func() { p2.Resolve(2) }()
	go func() { p3.Resolve(3) }()

	results, err := antsx.PromiseAll(ctx, p1, p2, p3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[0] != 1 || results[1] != 2 || results[2] != 3 {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestPromiseAll_OneFail(t *testing.T) {
	ctx := context.Background()
	p1 := antsx.NewPromise[int]()
	p2 := antsx.NewPromise[int]()

	go func() { p1.Resolve(1) }()
	go func() { p2.Reject(errors.New("fail")) }()

	_, err := antsx.PromiseAll(ctx, p1, p2)
	if err == nil {
		t.Fatal("expected error from PromiseAll")
	}
}

func TestPromiseAll_Empty(t *testing.T) {
	results, err := antsx.PromiseAll[int](context.Background())
	if err != nil || len(results) != 0 {
		t.Fatalf("expected empty, got %v, %v", results, err)
	}
}

func TestPromiseRace_Success(t *testing.T) {
	ctx := context.Background()
	p1 := antsx.NewPromise[string]()
	p2 := antsx.NewPromise[string]()

	go func() {
		time.Sleep(200 * time.Millisecond)
		p1.Resolve("slow")
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		p2.Resolve("fast")
	}()

	result, err := antsx.PromiseRace(ctx, p1, p2)
	if err != nil {
		t.Fatal(err)
	}
	if result != "fast" {
		t.Fatalf("expected 'fast' but got '%s'", result)
	}
}

func TestPromiseRace_Empty(t *testing.T) {
	_, err := antsx.PromiseRace[int](context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestGo_Success(t *testing.T) {
	ctx := context.Background()
	p := antsx.Go(ctx, func(ctx context.Context) (int, error) {
		return 100, nil
	})

	val, err := p.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val != 100 {
		t.Fatalf("expected 100, got %d", val)
	}
}

func TestGo_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	p := antsx.Go(ctx, func(ctx context.Context) (int, error) {
		panic("go boom")
	})

	_, err := p.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic")
	}
}

func TestPromiseAll_CtxCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	p1 := antsx.NewPromise[int]()

	_, err := antsx.PromiseAll(ctx, p1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestPromiseRace_AllFail(t *testing.T) {
	ctx := context.Background()
	p1 := antsx.NewPromise[string]()
	p2 := antsx.NewPromise[string]()

	go func() {
		p1.Reject(errors.New("err1"))
	}()
	go func() {
		p2.Reject(errors.New("err2"))
	}()

	_, err := antsx.PromiseRace(ctx, p1, p2)
	if err == nil {
		t.Fatal("expected error when all promises reject")
	}
}

func TestThen_UpstreamFail(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Reject(errors.New("upstream err")) }()

	p2 := antsx.Then(ctx, p, func(v int) (string, error) {
		return "should not reach", nil
	})

	_, err := p2.Await(ctx)
	if err == nil || err.Error() != "upstream err" {
		t.Fatalf("expected upstream err, got %v", err)
	}
}

func TestMap_UpstreamFail(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Reject(errors.New("map upstream")) }()

	mapped := antsx.Map(ctx, p, func(v int) string { return "nope" })

	_, err := mapped.Await(ctx)
	if err == nil || err.Error() != "map upstream" {
		t.Fatalf("expected 'map upstream', got %v", err)
	}
}

func TestFlatMap_UpstreamFail(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Reject(errors.New("flatmap upstream")) }()

	flatMapped := antsx.FlatMap(ctx, p, func(v int) *antsx.Promise[string] {
		return antsx.NewPromise[string]()
	})

	_, err := flatMapped.Await(ctx)
	if err == nil || err.Error() != "flatmap upstream" {
		t.Fatalf("expected 'flatmap upstream', got %v", err)
	}
}

func TestPromise_ResolveIdempotent(t *testing.T) {
	p := antsx.NewPromise[int]()
	p.Resolve(1)
	p.Resolve(2)
	p.Reject(errors.New("should not override"))

	val, err := p.Await(context.Background())
	if err != nil || val != 1 {
		t.Fatalf("expected (1, nil), got (%d, %v)", val, err)
	}
}

func TestPromise_ConcurrentResolveReject(t *testing.T) {
	p := antsx.NewPromise[int]()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			if v%2 == 0 {
				p.Resolve(v)
			} else {
				p.Reject(fmt.Errorf("err-%d", v))
			}
		}(i)
	}
	wg.Wait()

	_, _, ok := p.Get()
	if !ok {
		t.Fatal("expected promise to be settled")
	}
}

func TestPromise_ConcurrentAwait(t *testing.T) {
	p := antsx.NewPromise[int]()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := p.Await(context.Background())
			if err != nil || val != 42 {
				t.Errorf("expected (42, nil), got (%d, %v)", val, err)
			}
		}()
	}

	time.Sleep(10 * time.Millisecond)
	p.Resolve(42)
	wg.Wait()
}

func TestPromiseAll_ConcurrentCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	promises := make([]*antsx.Promise[int], 5)
	for i := range promises {
		promises[i] = antsx.NewPromise[int]()
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := antsx.PromiseAll(ctx, promises...)
	if err == nil {
		t.Fatal("expected error from PromiseAll cancel")
	}
}

func TestThen_Chained(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(1) }()

	p2 := antsx.Then(ctx, p, func(v int) (int, error) { return v + 1, nil })
	p3 := antsx.Then(ctx, p2, func(v int) (int, error) { return v * 10, nil })
	p4 := antsx.Then(ctx, p3, func(v int) (string, error) { return fmt.Sprintf("result=%d", v), nil })

	result, err := p4.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "result=20" {
		t.Fatalf("expected 'result=20', got '%s'", result)
	}
}

func TestFlatMap_InnerPanic(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]()
	go func() { p.Resolve(1) }()

	fm := antsx.FlatMap(ctx, p, func(v int) *antsx.Promise[string] {
		inner := antsx.NewPromise[string]()
		go func() {
			defer func() {
				if r := recover(); r != nil {
					inner.Reject(fmt.Errorf("inner panic: %v", r))
				}
			}()
			panic("inner boom")
		}()
		return inner
	})

	_, err := fm.Await(ctx)
	if err == nil {
		t.Fatal("expected error from inner panic")
	}
}

func TestPromiseRace_CtxTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	p1 := antsx.NewPromise[int]()
	p2 := antsx.NewPromise[int]()

	_, err := antsx.PromiseRace(ctx, p1, p2)
	if err == nil {
		t.Fatal("expected error from ctx timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestThen_CtxCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := antsx.NewPromise[int]()
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.Resolve(42)
	}()

	result := antsx.Then(ctx, p, func(v int) (string, error) {
		return fmt.Sprintf("v=%d", v), nil
	})

	_, err := result.Await(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled ctx")
	}
}

func TestGo_CtxCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	p := antsx.Go(ctx, func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})

	cancel()

	_, err := p.Await(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
