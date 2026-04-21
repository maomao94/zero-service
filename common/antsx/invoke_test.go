package antsx_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"zero-service/common/antsx"
)

// ======================== Invoke ========================

func TestInvoke_AllSuccess(t *testing.T) {
	ctx := context.Background()

	results, err := antsx.Invoke(ctx,
		antsx.Task[int]{Name: "t1", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 10, nil
		}},
		antsx.Task[int]{Name: "t2", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(30 * time.Millisecond)
			return 20, nil
		}},
		antsx.Task[int]{Name: "t3", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 30, nil
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[0] != 10 || results[1] != 20 || results[2] != 30 {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestInvoke_OneFail_FastFail(t *testing.T) {
	ctx := context.Background()

	start := time.Now()
	_, err := antsx.Invoke(ctx,
		antsx.Task[string]{Name: "slow", Fn: func(ctx context.Context) (string, error) {
			select {
			case <-time.After(2 * time.Second):
				return "slow-done", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}},
		antsx.Task[string]{Name: "fast-fail", Fn: func(ctx context.Context) (string, error) {
			time.Sleep(30 * time.Millisecond)
			return "", errors.New("task2 boom")
		}},
	)

	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "task2 boom" {
		t.Fatalf("expected 'task2 boom', got '%v'", err)
	}
	// fast-fail 应该在远少于 2s 内返回
	if elapsed > 500*time.Millisecond {
		t.Fatalf("fast-fail took too long: %v", elapsed)
	}
}

func TestInvoke_PerTaskTimeout(t *testing.T) {
	ctx := context.Background()

	_, err := antsx.Invoke(ctx,
		antsx.Task[string]{
			Name:    "slow-task",
			Timeout: 50 * time.Millisecond,
			Fn: func(ctx context.Context) (string, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return "done", nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
		},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestInvoke_OverallTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := antsx.Invoke(ctx,
		antsx.Task[string]{Name: "forever", Fn: func(ctx context.Context) (string, error) {
			select {
			case <-time.After(5 * time.Second):
				return "done", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestInvoke_Empty(t *testing.T) {
	ctx := context.Background()
	results, err := antsx.Invoke[int](ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %v", results)
	}
}

func TestInvoke_ResultOrder(t *testing.T) {
	ctx := context.Background()

	results, err := antsx.Invoke(ctx,
		antsx.Task[string]{Name: "slow", Fn: func(ctx context.Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "first", nil
		}},
		antsx.Task[string]{Name: "fast", Fn: func(ctx context.Context) (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "second", nil
		}},
		antsx.Task[string]{Name: "medium", Fn: func(ctx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "third", nil
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	// 结果按 index 排列，不是按完成顺序
	if results[0] != "first" || results[1] != "second" || results[2] != "third" {
		t.Fatalf("results not in index order: %v", results)
	}
}

// ======================== InvokeCallback ========================

func TestInvokeCallback_Transform(t *testing.T) {
	ctx := context.Background()

	sum, err := antsx.InvokeCallback(ctx,
		[]antsx.Task[int]{
			{Name: "a", Fn: func(ctx context.Context) (int, error) { return 10, nil }},
			{Name: "b", Fn: func(ctx context.Context) (int, error) { return 20, nil }},
			{Name: "c", Fn: func(ctx context.Context) (int, error) { return 30, nil }},
		},
		func(vals []int) (int, error) {
			total := 0
			for _, v := range vals {
				total += v
			}
			return total, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sum != 60 {
		t.Fatalf("expected 60, got %d", sum)
	}
}

func TestInvokeCallback_TaskFail(t *testing.T) {
	ctx := context.Background()
	callbackCalled := false

	_, err := antsx.InvokeCallback(ctx,
		[]antsx.Task[int]{
			{Name: "ok", Fn: func(ctx context.Context) (int, error) { return 1, nil }},
			{Name: "fail", Fn: func(ctx context.Context) (int, error) { return 0, errors.New("boom") }},
		},
		func(vals []int) (string, error) {
			callbackCalled = true
			return "should not reach", nil
		},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if callbackCalled {
		t.Fatal("callback should not be called when task fails")
	}
}

// ======================== InvokeWithReactor ========================

func TestInvokeWithReactor_Success(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	results, err := antsx.InvokeWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "r1", Fn: func(ctx context.Context) (int, error) {
			return 100, nil
		}},
		antsx.Task[int]{Name: "r2", Fn: func(ctx context.Context) (int, error) {
			return 200, nil
		}},
		antsx.Task[int]{Name: "r3", Fn: func(ctx context.Context) (int, error) {
			return 300, nil
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[0] != 100 || results[1] != 200 || results[2] != 300 {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestInvokeWithReactor_PoolStress(t *testing.T) {
	reactor, err := antsx.NewReactor(1) // 只有 1 个 worker
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	tasks := make([]antsx.Task[int], 10)
	for i := 0; i < 10; i++ {
		idx := i
		tasks[i] = antsx.Task[int]{
			Name: "stress",
			Fn: func(ctx context.Context) (int, error) {
				time.Sleep(10 * time.Millisecond)
				return idx, nil
			},
		}
	}

	results, err := antsx.InvokeWithReactor(ctx, reactor, tasks...)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}
	for i, v := range results {
		if v != i {
			t.Fatalf("results[%d]: expected %d, got %d", i, i, v)
		}
	}
}

// ======================== Panic Recovery ========================

func TestInvoke_PanicRecovery(t *testing.T) {
	ctx := context.Background()

	_, err := antsx.Invoke(ctx,
		antsx.Task[string]{Name: "normal", Fn: func(ctx context.Context) (string, error) {
			return "ok", nil
		}},
		antsx.Task[string]{Name: "panicker", Fn: func(ctx context.Context) (string, error) {
			panic("something went wrong")
		}},
	)
	if err == nil {
		t.Fatal("expected error from panic")
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("expected panic error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "panicker") {
		t.Fatalf("expected task name in error, got: %v", err)
	}
	t.Logf("Panic recovered: %v", err)
}

func TestInvokeWithReactor_PanicRecovery(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	_, err = antsx.InvokeWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "boom", Fn: func(ctx context.Context) (int, error) {
			panic("reactor panic")
		}},
	)
	if err == nil {
		t.Fatal("expected error from panic")
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("expected panic error, got: %v", err)
	}
}

func TestInvoke_SingleTask(t *testing.T) {
	ctx := context.Background()
	results, err := antsx.Invoke(ctx,
		antsx.Task[string]{Name: "only", Fn: func(ctx context.Context) (string, error) {
			return "single", nil
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0] != "single" {
		t.Fatalf("unexpected: %v", results)
	}
}

func TestInvoke_SingleTaskPanic(t *testing.T) {
	ctx := context.Background()
	_, err := antsx.Invoke(ctx,
		antsx.Task[string]{Name: "panic-single", Fn: func(ctx context.Context) (string, error) {
			panic("single boom")
		}},
	)
	if err == nil {
		t.Fatal("expected error from panic")
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("expected panic error, got: %v", err)
	}
}

func TestInvokeCallback_Empty(t *testing.T) {
	ctx := context.Background()
	result, err := antsx.InvokeCallback(ctx,
		[]antsx.Task[int]{},
		func(vals []int) (int, error) {
			return 0, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result != 0 {
		t.Fatalf("expected 0, got %d", result)
	}
}

func TestInvoke_AllPanic(t *testing.T) {
	ctx := context.Background()
	_, err := antsx.Invoke(ctx,
		antsx.Task[int]{Name: "p1", Fn: func(ctx context.Context) (int, error) {
			panic("boom1")
		}},
		antsx.Task[int]{Name: "p2", Fn: func(ctx context.Context) (int, error) {
			panic("boom2")
		}},
	)
	if err == nil {
		t.Fatal("expected error from panic")
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("expected panic error, got: %v", err)
	}
}

func TestInvokeCallback_CallbackPanic(t *testing.T) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from callback")
		}
	}()

	antsx.InvokeCallback(ctx,
		[]antsx.Task[int]{
			{Name: "ok", Fn: func(ctx context.Context) (int, error) { return 1, nil }},
		},
		func(vals []int) (string, error) {
			panic("callback panic")
		},
	)
}

func TestInvokeWithReactor_PoolExhausted(t *testing.T) {
	reactor, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	results, err := antsx.InvokeWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "a", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 1, nil
		}},
		antsx.Task[int]{Name: "b", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 2, nil
		}},
		antsx.Task[int]{Name: "c", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 3, nil
		}},
		antsx.Task[int]{Name: "d", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 4, nil
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	for i, v := range results {
		if v != i+1 {
			t.Fatalf("results[%d]: expected %d, got %d", i, i+1, v)
		}
	}
}

func TestInvokeWithReactor_PoolReleased(t *testing.T) {
	reactor, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	reactor.Release()

	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := antsx.InvokeWithReactor(ctx, reactor,
			antsx.Task[int]{Name: "a", Fn: func(ctx context.Context) (int, error) {
				return 1, nil
			}},
			antsx.Task[int]{Name: "b", Fn: func(ctx context.Context) (int, error) {
				return 2, nil
			}},
			antsx.Task[int]{Name: "c", Fn: func(ctx context.Context) (int, error) {
				return 3, nil
			}},
		)
		if err == nil {
			t.Error("expected error from released pool")
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("InvokeWithReactor deadlocked on released pool (wg bug)")
	}
}

func TestInvokeWithReactor_CtxCancel(t *testing.T) {
	reactor, err := antsx.NewReactor(4)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = antsx.InvokeWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "a", Fn: func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		}},
		antsx.Task[int]{Name: "b", Fn: func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		}},
	)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestInvokeWithReactor_SingleTask(t *testing.T) {
	reactor, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	results, err := antsx.InvokeWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "single", Fn: func(ctx context.Context) (int, error) {
			return 42, nil
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0] != 42 {
		t.Fatalf("expected [42], got %v", results)
	}
}

func TestInvokeWithReactor_Empty(t *testing.T) {
	reactor, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	results, err := antsx.InvokeWithReactor[int](ctx, reactor)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %v", results)
	}
}

// ======================== InvokeAllSettled ========================

func TestInvokeAllSettled_AllSuccess(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "a", Fn: func(ctx context.Context) (int, error) { return 1, nil }},
		antsx.Task[int]{Name: "b", Fn: func(ctx context.Context) (int, error) { return 2, nil }},
		antsx.Task[int]{Name: "c", Fn: func(ctx context.Context) (int, error) { return 3, nil }},
	)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Succeeded() {
			t.Fatalf("results[%d] expected success, got err: %v", i, r.Err)
		}
		if r.Val != i+1 {
			t.Fatalf("results[%d].Val expected %d, got %d", i, i+1, r.Val)
		}
	}
}

func TestInvokeAllSettled_PartialFailure(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[string]{Name: "ok1", Fn: func(ctx context.Context) (string, error) {
			return "hello", nil
		}},
		antsx.Task[string]{Name: "fail", Fn: func(ctx context.Context) (string, error) {
			return "", errors.New("task failed")
		}},
		antsx.Task[string]{Name: "ok2", Fn: func(ctx context.Context) (string, error) {
			return "world", nil
		}},
	)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].Succeeded() || results[0].Val != "hello" {
		t.Fatalf("results[0] unexpected: %+v", results[0])
	}
	if results[1].Succeeded() || results[1].Err.Error() != "task failed" {
		t.Fatalf("results[1] expected failure, got: %+v", results[1])
	}
	if !results[2].Succeeded() || results[2].Val != "world" {
		t.Fatalf("results[2] unexpected: %+v", results[2])
	}
}

func TestInvokeAllSettled_NoFastFail(t *testing.T) {
	ctx := context.Background()
	start := time.Now()

	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "slow", Fn: func(ctx context.Context) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return 42, nil
		}},
		antsx.Task[int]{Name: "fast-fail", Fn: func(ctx context.Context) (int, error) {
			return 0, errors.New("boom")
		}},
	)

	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond {
		t.Fatalf("should wait for all tasks, but returned in %v", elapsed)
	}
	if !results[0].Succeeded() || results[0].Val != 42 {
		t.Fatalf("slow task should succeed: %+v", results[0])
	}
	if results[1].Succeeded() {
		t.Fatal("fast-fail task should have failed")
	}
}

func TestInvokeAllSettled_TaskNotCancelledByOther(t *testing.T) {
	ctx := context.Background()

	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[string]{Name: "fail-first", Fn: func(ctx context.Context) (string, error) {
			return "", errors.New("i failed")
		}},
		antsx.Task[string]{Name: "check-ctx", Fn: func(ctx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "still-alive", nil
		}},
	)

	if results[1].Err != nil {
		t.Fatalf("other task's ctx should not be cancelled, but got: %v", results[1].Err)
	}
	if results[1].Val != "still-alive" {
		t.Fatalf("expected 'still-alive', got %q", results[1].Val)
	}
}

func TestInvokeAllSettled_Panic(t *testing.T) {
	ctx := context.Background()

	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "normal", Fn: func(ctx context.Context) (int, error) {
			return 1, nil
		}},
		antsx.Task[int]{Name: "panicker", Fn: func(ctx context.Context) (int, error) {
			panic("boom")
		}},
		antsx.Task[int]{Name: "normal2", Fn: func(ctx context.Context) (int, error) {
			return 3, nil
		}},
	)

	if !results[0].Succeeded() || results[0].Val != 1 {
		t.Fatalf("results[0] unexpected: %+v", results[0])
	}
	if results[1].Succeeded() {
		t.Fatal("panicker should fail")
	}
	if !strings.Contains(results[1].Err.Error(), "panicked") {
		t.Fatalf("expected panic error, got: %v", results[1].Err)
	}
	if !results[2].Succeeded() || results[2].Val != 3 {
		t.Fatalf("results[2] unexpected: %+v", results[2])
	}
}

func TestInvokeAllSettled_Empty(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled[int](ctx)
	if len(results) != 0 {
		t.Fatalf("expected empty, got %v", results)
	}
}

func TestInvokeAllSettled_Single(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "only", Fn: func(ctx context.Context) (int, error) {
			return 99, nil
		}},
	)
	if len(results) != 1 || !results[0].Succeeded() || results[0].Val != 99 {
		t.Fatalf("unexpected: %+v", results)
	}
}

func TestInvokeAllSettled_SingleFail(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "fail", Fn: func(ctx context.Context) (int, error) {
			return 0, errors.New("single fail")
		}},
	)
	if len(results) != 1 || results[0].Succeeded() {
		t.Fatalf("unexpected: %+v", results)
	}
}

func TestInvokeAllSettled_SinglePanic(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "panic-single", Fn: func(ctx context.Context) (int, error) {
			panic("single panic")
		}},
	)
	if len(results) != 1 || results[0].Succeeded() {
		t.Fatalf("unexpected: %+v", results)
	}
	if !strings.Contains(results[0].Err.Error(), "panicked") {
		t.Fatalf("expected panic error, got: %v", results[0].Err)
	}
}

func TestInvokeAllSettled_PerTaskTimeout(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[string]{
			Name:    "timeout-task",
			Timeout: 30 * time.Millisecond,
			Fn: func(ctx context.Context) (string, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return "done", nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
		},
		antsx.Task[string]{
			Name: "normal-task",
			Fn: func(ctx context.Context) (string, error) {
				return "ok", nil
			},
		},
	)

	if results[0].Succeeded() {
		t.Fatal("timeout task should fail")
	}
	if !errors.Is(results[0].Err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", results[0].Err)
	}
	if !results[1].Succeeded() || results[1].Val != "ok" {
		t.Fatalf("normal task should succeed: %+v", results[1])
	}
}

func TestInvokeAllSettled_ResultName(t *testing.T) {
	ctx := context.Background()
	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "alpha", Fn: func(ctx context.Context) (int, error) { return 1, nil }},
		antsx.Task[int]{Name: "beta", Fn: func(ctx context.Context) (int, error) { return 0, errors.New("err") }},
	)
	if results[0].Name != "alpha" || results[1].Name != "beta" {
		t.Fatalf("names mismatch: %q, %q", results[0].Name, results[1].Name)
	}
}

func TestInvokeAllSettled_CtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := antsx.InvokeAllSettled(ctx,
		antsx.Task[int]{Name: "a", Fn: func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		}},
		antsx.Task[int]{Name: "b", Fn: func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		}},
	)
	for i, r := range results {
		if r.Succeeded() {
			t.Fatalf("results[%d] should fail on cancelled ctx", i)
		}
	}
}

// ======================== InvokeAllSettledWithReactor ========================

func TestInvokeAllSettledWithReactor_AllSuccess(t *testing.T) {
	reactor, err := antsx.NewReactor(4)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	results := antsx.InvokeAllSettledWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "a", Fn: func(ctx context.Context) (int, error) { return 10, nil }},
		antsx.Task[int]{Name: "b", Fn: func(ctx context.Context) (int, error) { return 20, nil }},
		antsx.Task[int]{Name: "c", Fn: func(ctx context.Context) (int, error) { return 30, nil }},
	)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Succeeded() {
			t.Fatalf("results[%d] expected success, got: %v", i, r.Err)
		}
	}
	if results[0].Val != 10 || results[1].Val != 20 || results[2].Val != 30 {
		t.Fatalf("values mismatch: %v", results)
	}
}

func TestInvokeAllSettledWithReactor_PartialFailure(t *testing.T) {
	reactor, err := antsx.NewReactor(4)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	results := antsx.InvokeAllSettledWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "ok", Fn: func(ctx context.Context) (int, error) { return 1, nil }},
		antsx.Task[int]{Name: "fail", Fn: func(ctx context.Context) (int, error) { return 0, errors.New("boom") }},
		antsx.Task[int]{Name: "ok2", Fn: func(ctx context.Context) (int, error) { return 3, nil }},
	)

	if !results[0].Succeeded() || results[0].Val != 1 {
		t.Fatalf("results[0] unexpected: %+v", results[0])
	}
	if results[1].Succeeded() {
		t.Fatal("results[1] should fail")
	}
	if !results[2].Succeeded() || results[2].Val != 3 {
		t.Fatalf("results[2] unexpected: %+v", results[2])
	}
}

func TestInvokeAllSettledWithReactor_PoolReleased(t *testing.T) {
	reactor, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	reactor.Release()

	ctx := context.Background()
	results := antsx.InvokeAllSettledWithReactor(ctx, reactor,
		antsx.Task[int]{Name: "a", Fn: func(ctx context.Context) (int, error) { return 1, nil }},
		antsx.Task[int]{Name: "b", Fn: func(ctx context.Context) (int, error) { return 2, nil }},
		antsx.Task[int]{Name: "c", Fn: func(ctx context.Context) (int, error) { return 3, nil }},
	)

	hasErr := false
	for _, r := range results {
		if !r.Succeeded() {
			hasErr = true
			break
		}
	}
	if !hasErr {
		t.Fatal("expected at least one error from released pool")
	}
}

func TestInvokeAllSettledWithReactor_Empty(t *testing.T) {
	reactor, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	results := antsx.InvokeAllSettledWithReactor[int](context.Background(), reactor)
	if len(results) != 0 {
		t.Fatalf("expected empty, got %v", results)
	}
}

func TestInvokeAllSettledWithReactor_Single(t *testing.T) {
	reactor, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	results := antsx.InvokeAllSettledWithReactor(context.Background(), reactor,
		antsx.Task[int]{Name: "only", Fn: func(ctx context.Context) (int, error) { return 7, nil }},
	)
	if len(results) != 1 || !results[0].Succeeded() || results[0].Val != 7 {
		t.Fatalf("unexpected: %+v", results)
	}
}

func TestInvokeAllSettledWithReactor_Panic(t *testing.T) {
	reactor, err := antsx.NewReactor(4)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	results := antsx.InvokeAllSettledWithReactor(context.Background(), reactor,
		antsx.Task[int]{Name: "normal", Fn: func(ctx context.Context) (int, error) { return 1, nil }},
		antsx.Task[int]{Name: "panicker", Fn: func(ctx context.Context) (int, error) { panic("reactor panic") }},
	)

	if !results[0].Succeeded() || results[0].Val != 1 {
		t.Fatalf("results[0] unexpected: %+v", results[0])
	}
	if results[1].Succeeded() {
		t.Fatal("panicker should fail")
	}
	if !strings.Contains(results[1].Err.Error(), "panicked") {
		t.Fatalf("expected panic error, got: %v", results[1].Err)
	}
}
