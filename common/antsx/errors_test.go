package antsx_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"zero-service/common/antsx"
)

func TestSourceEOF_Error(t *testing.T) {
	sr1, sw1 := antsx.Pipe[int](1)
	go func() {
		defer sw1.Close()
		sw1.Send(1, nil)
	}()

	merged := antsx.MergeNamedStreamReaders(map[string]*antsx.StreamReader[int]{
		"test-source": sr1,
	})
	defer merged.Close()

	_, _ = merged.Recv()
	_, err := merged.Recv()
	if err == nil {
		t.Fatal("expected SourceEOF error")
	}

	name, ok := antsx.GetSourceName(err)
	if !ok || name != "test-source" {
		t.Fatalf("expected source name 'test-source', got %q, ok=%v", name, ok)
	}

	if !strings.Contains(err.Error(), "test-source") {
		t.Fatalf("error message should contain source name, got: %s", err.Error())
	}
}

func TestPanicErr_FromReactorSubmit(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	innerErr := errors.New("inner panic error")
	p, err := antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
		panic(innerErr)
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Await(context.Background())
	if err == nil {
		t.Fatal("expected error from panic")
	}

	if !strings.Contains(err.Error(), "inner panic error") {
		t.Fatalf("error should contain panic info, got: %s", err.Error())
	}

	if !errors.Is(err, innerErr) {
		t.Fatal("errors.Is should unwrap to inner error")
	}
}

func TestPanicErr_Unwrap_NonError(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	p, err := antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
		panic("string panic")
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Await(context.Background())
	if err == nil {
		t.Fatal("expected error from panic")
	}

	if !strings.Contains(err.Error(), "string panic") {
		t.Fatalf("error should contain panic info, got: %s", err.Error())
	}
}

func TestPanicErr_ContainsStack(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	p, err := antsx.Submit(context.Background(), r, func(ctx context.Context) (int, error) {
		panic("stack trace test")
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Await(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "goroutine") || !strings.Contains(errMsg, "stack trace test") {
		t.Fatalf("error should contain stack trace, got: %s", errMsg)
	}
}
