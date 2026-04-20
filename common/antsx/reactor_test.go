package antsx_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
	"zero-service/common/antsx"
)

func TestReactor_Submit(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	p, err := antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	val, err := p.Await(context.Background())
	if err != nil || val != 42 {
		t.Fatalf("expected 42, got %d, err=%v", val, err)
	}
}

func TestReactor_SubmitPanic(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	p, err := antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
		panic("submit boom")
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Await(context.Background())
	if err == nil {
		t.Fatal("expected error from panic")
	}
}

func TestReactor_SubmitCtxCancel(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p, submitErr := antsx.Submit(ctx, r, func(ctx context.Context) (int, error) {
		return 0, ctx.Err()
	})
	if submitErr != nil {
		t.Fatal(submitErr)
	}

	_, err = p.Await(context.Background())
	if err == nil {
		t.Fatal("expected error from cancelled ctx")
	}
}

func TestReactor_Post(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	called := make(chan struct{}, 1)
	err = antsx.Post(context.Background(), r, func(ctx context.Context) error {
		called <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("Post callback not called")
	}
}

func TestReactor_PostPanic(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	err = antsx.Post(context.Background(), r, func(ctx context.Context) error {
		panic("post boom")
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestReactor_Go(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	var called int32
	err = r.Go(context.Background(), func(ctx context.Context) {
		atomic.StoreInt32(&called, 1)
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&called) != 1 {
		t.Fatal("Go callback not called")
	}
}

func TestReactor_ActiveCount(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	start := make(chan struct{})
	_ = r.Go(context.Background(), func(ctx context.Context) {
		<-start
	})
	time.Sleep(20 * time.Millisecond)
	if r.ActiveCount() < 1 {
		t.Fatal("expected at least 1 active")
	}
	close(start)
}

func TestReactor_SubmitError(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	testErr := errors.New("task failed")
	p, err := antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
		return 0, testErr
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Await(context.Background())
	if !errors.Is(err, testErr) {
		t.Fatalf("expected testErr, got %v", err)
	}
}

func TestReactor_Go_PanicRecovery(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	err = r.Go(context.Background(), func(ctx context.Context) {
		panic("go boom")
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestReactor_Go_CtxCancel(t *testing.T) {
	r, err := antsx.NewReactor(10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var ctxErr error
	done := make(chan struct{})
	err = r.Go(ctx, func(ctx context.Context) {
		ctxErr = ctx.Err()
		close(done)
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Go callback not called")
	}
	if ctxErr == nil {
		t.Fatal("expected ctx error")
	}
}

func TestReactor_SubmitConcurrent(t *testing.T) {
	r, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	const n = 50
	promises := make([]*antsx.Promise[int], n)
	for i := 0; i < n; i++ {
		idx := i
		p, err := antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
			time.Sleep(time.Millisecond)
			return idx * 2, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		promises[i] = p
	}

	for i, p := range promises {
		val, err := p.Await(context.Background())
		if err != nil {
			t.Fatalf("promise %d: %v", i, err)
		}
		if val != i*2 {
			t.Fatalf("promise %d: expected %d, got %d", i, i*2, val)
		}
	}
}

func TestReactor_SubmitAfterRelease(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	r.Release()

	_, err = antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
		return 1, nil
	})
	if err == nil {
		t.Fatal("expected error after release")
	}
}

func TestReactor_PostAfterRelease(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	r.Release()

	err = antsx.Post(context.Background(), r, func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error after release")
	}
}

func TestReactor_GoAfterRelease(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	r.Release()

	err = r.Go(context.Background(), func(ctx context.Context) {})
	if err == nil {
		t.Fatal("expected error after release")
	}
}
