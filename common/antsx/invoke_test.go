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
