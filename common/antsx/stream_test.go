package antsx_test

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"
	"zero-service/common/antsx"
)

func drainReader[T any](t *testing.T, sr *antsx.StreamReader[T]) []T {
	t.Helper()
	var results []T
	for {
		val, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	return results
}

func TestStream_BasicPipe(t *testing.T) {
	sr, sw := antsx.Pipe[int](5)

	go func() {
		defer sw.Close()
		for i := 0; i < 5; i++ {
			sw.Send(i, nil)
		}
	}()

	defer sr.Close()
	results := drainReader(t, sr)

	if len(results) != 5 {
		t.Fatalf("expected 5 items, got %d", len(results))
	}
	for i, v := range results {
		if v != i {
			t.Fatalf("results[%d] = %d, want %d", i, v, i)
		}
	}
}

func TestStream_EarlyClose(t *testing.T) {
	sr, sw := antsx.Pipe[int](1)
	sr.Close()

	closed := sw.Send(42, nil)
	if !closed {
		t.Fatal("expected Send to return closed=true after reader close")
	}
	sw.Close()
}

func TestStream_CloseRecvIdempotent(t *testing.T) {
	sr, sw := antsx.Pipe[int](1)
	defer sw.Close()

	sr.Close()
	sr.Close()
	sr.Close()
}

func TestStream_ConcurrentClose(t *testing.T) {
	sr, sw := antsx.Pipe[int](1)
	defer sw.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sr.Close()
		}()
	}
	wg.Wait()
}

func TestStream_Copy(t *testing.T) {
	sr, sw := antsx.Pipe[int](5)

	go func() {
		defer sw.Close()
		for i := 0; i < 3; i++ {
			sw.Send(i, nil)
		}
	}()

	copies := sr.Copy(2)
	sr1, sr2 := copies[0], copies[1]
	defer sr1.Close()
	defer sr2.Close()

	r1 := drainReader(t, sr1)
	r2 := drainReader(t, sr2)

	if len(r1) != 3 || len(r2) != 3 {
		t.Fatalf("expected 3 items each, got %d and %d", len(r1), len(r2))
	}
	for i := 0; i < 3; i++ {
		if r1[i] != i || r2[i] != i {
			t.Fatalf("mismatch at %d: r1=%d, r2=%d", i, r1[i], r2[i])
		}
	}
}

func TestStream_CopyConcurrent(t *testing.T) {
	sr, sw := antsx.Pipe[int](10)

	go func() {
		defer sw.Close()
		for i := 0; i < 100; i++ {
			sw.Send(i, nil)
		}
	}()

	copies := sr.Copy(3)
	var wg sync.WaitGroup
	results := make([][]int, 3)

	for i, c := range copies {
		wg.Add(1)
		go func(idx int, r *antsx.StreamReader[int]) {
			defer wg.Done()
			defer r.Close()
			for {
				v, err := r.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Errorf("reader %d: %v", idx, err)
					return
				}
				results[idx] = append(results[idx], v)
			}
		}(i, c)
	}
	wg.Wait()

	for i, r := range results {
		if len(r) != 100 {
			t.Fatalf("reader %d: expected 100 items, got %d", i, len(r))
		}
	}
}

func TestStream_CopySingle(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	copies := sr.Copy(1)
	if copies[0] != sr {
		t.Fatal("Copy(1) should return same reader")
	}
}

func TestStream_CopyArray(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	copies := sr.Copy(2)
	r1 := drainReader(t, copies[0])
	r2 := drainReader(t, copies[1])

	if len(r1) != 3 || len(r2) != 3 {
		t.Fatalf("expected 3 items each, got %d and %d", len(r1), len(r2))
	}
}

func TestStream_FromArray(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]string{"a", "b", "c"})
	defer sr.Close()

	results := drainReader(t, sr)
	if len(results) != 3 || results[0] != "a" || results[1] != "b" || results[2] != "c" {
		t.Fatalf("unexpected: %v", results)
	}
}

func TestStream_FromArrayEmpty(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{})
	defer sr.Close()

	_, err := sr.Recv()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestStream_Convert(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{0, 1, 2, 3})
	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		if i == 0 {
			return "", antsx.ErrNoValue
		}
		return fmt.Sprintf("v%d", i), nil
	})
	defer converted.Close()

	results := drainReader(t, converted)
	if len(results) != 3 || results[0] != "v1" || results[1] != "v2" || results[2] != "v3" {
		t.Fatalf("unexpected: %v", results)
	}
}

func TestStream_ConvertError(t *testing.T) {
	testErr := errors.New("convert error")
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		if i == 2 {
			return "", testErr
		}
		return fmt.Sprintf("v%d", i), nil
	})
	defer converted.Close()

	v, err := converted.Recv()
	if err != nil || v != "v1" {
		t.Fatalf("expected v1, got %q, err=%v", v, err)
	}

	_, err = converted.Recv()
	if !errors.Is(err, testErr) {
		t.Fatalf("expected testErr, got %v", err)
	}
}

func TestStream_ConvertErrWrapper(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		return fmt.Sprintf("v%d", i), nil
	}, antsx.WithErrWrapper(func(err error) error {
		return nil
	}))
	defer converted.Close()

	results := drainReader(t, converted)
	if len(results) != 3 {
		t.Fatalf("expected 3 items, got %d", len(results))
	}
}

func TestMergeStreamReaders_Basic(t *testing.T) {
	sr1, sw1 := antsx.Pipe[int](5)
	sr2, sw2 := antsx.Pipe[int](5)

	go func() {
		defer sw1.Close()
		sw1.Send(1, nil)
		sw1.Send(2, nil)
	}()
	go func() {
		defer sw2.Close()
		sw2.Send(3, nil)
		sw2.Send(4, nil)
	}()

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2})
	defer merged.Close()

	results := drainReader(t, merged)
	if len(results) != 4 {
		t.Fatalf("expected 4 items, got %d", len(results))
	}
}

func TestMergeStreamReaders_ArrayOptimization(t *testing.T) {
	sr1 := antsx.StreamReaderFromArray([]int{1, 2, 3})
	sr2 := antsx.StreamReaderFromArray([]int{4, 5, 6})

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2})
	defer merged.Close()

	results := drainReader(t, merged)
	if len(results) != 6 {
		t.Fatalf("expected 6 items, got %d: %v", len(results), results)
	}
}

func TestMergeStreamReaders_SingleAndEmpty(t *testing.T) {
	sr1 := antsx.StreamReaderFromArray([]int{42})
	same := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1})
	if same != sr1 {
		t.Fatal("single merge should return same reader")
	}

	nilResult := antsx.MergeStreamReaders[int](nil)
	if nilResult != nil {
		t.Fatal("empty merge should return nil")
	}
}

func TestMergeStreamReaders_ManyReflectPath(t *testing.T) {
	const n = 7
	srs := make([]*antsx.StreamReader[int], n)
	for i := 0; i < n; i++ {
		sr, sw := antsx.Pipe[int](1)
		go func(idx int) {
			defer sw.Close()
			sw.Send(idx, nil)
		}(i)
		srs[i] = sr
	}

	merged := antsx.MergeStreamReaders(srs)
	defer merged.Close()

	results := drainReader(t, merged)
	if len(results) != n {
		t.Fatalf("expected %d items, got %d", n, len(results))
	}
}

func TestMergeStreamReaders_ReflectToStaticTransition(t *testing.T) {
	const n = 8
	srs := make([]*antsx.StreamReader[int], n)
	sws := make([]*antsx.StreamWriter[int], n)
	for i := 0; i < n; i++ {
		sr, sw := antsx.Pipe[int](10)
		srs[i] = sr
		sws[i] = sw
	}

	go func() {
		for i := 0; i < n; i++ {
			sws[i].Send(i*10, nil)
			sws[i].Send(i*10+1, nil)
			sws[i].Close()
		}
	}()

	merged := antsx.MergeStreamReaders(srs)
	defer merged.Close()

	results := drainReader(t, merged)
	if len(results) != n*2 {
		t.Fatalf("expected %d items, got %d", n*2, len(results))
	}
}

func TestMergeNamedStreamReaders(t *testing.T) {
	sr1, sw1 := antsx.Pipe[string](5)
	sr2, sw2 := antsx.Pipe[string](5)

	go func() {
		defer sw1.Close()
		sw1.Send("from-a", nil)
	}()
	go func() {
		defer sw2.Close()
		sw2.Send("from-b", nil)
	}()

	merged := antsx.MergeNamedStreamReaders(map[string]*antsx.StreamReader[string]{
		"source-a": sr1,
		"source-b": sr2,
	})
	defer merged.Close()

	var (
		values     []string
		sourceEOFs []string
	)

	for {
		v, err := merged.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if name, ok := antsx.GetSourceName(err); ok {
				sourceEOFs = append(sourceEOFs, name)
				continue
			}
			t.Fatal(err)
		}
		values = append(values, v)
	}

	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d: %v", len(values), values)
	}
	if len(sourceEOFs) != 2 {
		t.Fatalf("expected 2 source EOFs, got %d: %v", len(sourceEOFs), sourceEOFs)
	}
}

func TestMergeNamedStreamReaders_Empty(t *testing.T) {
	result := antsx.MergeNamedStreamReaders[int](nil)
	if result != nil {
		t.Fatal("expected nil")
	}
}

func TestMergeStreamReaders_NestedFlatten(t *testing.T) {
	sr1, sw1 := antsx.Pipe[int](5)
	sr2, sw2 := antsx.Pipe[int](5)
	sr3, sw3 := antsx.Pipe[int](5)

	go func() {
		defer sw1.Close()
		sw1.Send(1, nil)
	}()
	go func() {
		defer sw2.Close()
		sw2.Send(2, nil)
	}()
	go func() {
		defer sw3.Close()
		sw3.Send(3, nil)
	}()

	inner := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2})
	outer := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{inner, sr3})
	defer outer.Close()

	results := drainReader(t, outer)
	if len(results) != 3 {
		t.Fatalf("expected 3 items, got %d", len(results))
	}
}

func TestStream_ErrRecvAfterClosed(t *testing.T) {
	sr, sw := antsx.Pipe[int](5)

	go func() {
		defer sw.Close()
		sw.Send(1, nil)
		sw.Send(2, nil)
	}()

	copies := sr.Copy(2)
	sr1, sr2 := copies[0], copies[1]

	v, err := sr1.Recv()
	if err != nil || v != 1 {
		t.Fatalf("expected 1, got %d, err=%v", v, err)
	}

	sr1.Close()

	_, err = sr1.Recv()
	if !errors.Is(err, antsx.ErrRecvAfterClosed) {
		t.Fatalf("expected ErrRecvAfterClosed, got %v", err)
	}

	v, err = sr2.Recv()
	if err != nil || v != 1 {
		t.Fatalf("sr2: expected 1, got %d, err=%v", v, err)
	}
	sr2.Close()
}

func TestStream_SetAutomaticClose(t *testing.T) {
	sr, sw := antsx.Pipe[int](1)
	sr.SetAutomaticClose()
	sw.Send(1, nil)
	sw.Close()

	v, err := sr.Recv()
	if err != nil || v != 1 {
		t.Fatalf("expected 1, got %d, err=%v", v, err)
	}
}

func TestStream_ConvertPanicPropagation(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	panicSR := antsx.StreamReaderWithConvert(sr, func(i int) (int, error) {
		if i == 2 {
			panic("boom")
		}
		return i * 10, nil
	})
	defer panicSR.Close()

	v, err := panicSR.Recv()
	if err != nil || v != 10 {
		t.Fatalf("expected 10, got %d, err=%v", v, err)
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic from convert function")
			}
		}()
		panicSR.Recv()
	}()
}

func TestGetSourceName(t *testing.T) {
	_, ok := antsx.GetSourceName(io.EOF)
	if ok {
		t.Fatal("expected false for io.EOF")
	}

	_, ok = antsx.GetSourceName(errors.New("random"))
	if ok {
		t.Fatal("expected false for random error")
	}
}

func TestStream_SendError(t *testing.T) {
	sr, sw := antsx.Pipe[int](5)
	testErr := errors.New("test error")
	go func() {
		defer sw.Close()
		sw.Send(1, nil)
		sw.Send(0, testErr)
		sw.Send(2, nil)
	}()
	defer sr.Close()

	v, err := sr.Recv()
	if err != nil || v != 1 {
		t.Fatalf("expected 1, got %d, err=%v", v, err)
	}

	_, err = sr.Recv()
	if !errors.Is(err, testErr) {
		t.Fatalf("expected testErr, got %v", err)
	}

	v, err = sr.Recv()
	if err != nil || v != 2 {
		t.Fatalf("expected 2, got %d, err=%v", v, err)
	}
}

func TestStream_CloseSendIdempotent(t *testing.T) {
	sr, sw := antsx.Pipe[int](1)
	defer sr.Close()
	sw.Close()
	sw.Close()
	sw.Close()
}

func TestStream_ConcurrentCloseSend(t *testing.T) {
	sr, sw := antsx.Pipe[int](1)
	defer sr.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sw.Close()
		}()
	}
	wg.Wait()
}

func TestStream_MergeMidClose(t *testing.T) {
	sr1, sw1 := antsx.Pipe[int](5)
	sr2, sw2 := antsx.Pipe[int](5)

	go func() {
		defer sw1.Close()
		for i := 0; i < 10; i++ {
			sw1.Send(i, nil)
		}
	}()
	go func() {
		defer sw2.Close()
		for i := 100; i < 110; i++ {
			sw2.Send(i, nil)
		}
	}()

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2})

	count := 0
	for {
		_, err := merged.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		if count == 5 {
			merged.Close()
			break
		}
	}
	if count != 5 {
		t.Fatalf("expected 5 items before close, got %d", count)
	}
}

func TestStream_ConvertChained(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3, 4, 5})
	doubled := antsx.StreamReaderWithConvert(sr, func(i int) (int, error) {
		return i * 2, nil
	})
	asStr := antsx.StreamReaderWithConvert(doubled, func(i int) (string, error) {
		return fmt.Sprintf("%d", i), nil
	})
	defer asStr.Close()

	results := drainReader(t, asStr)
	expected := []string{"2", "4", "6", "8", "10"}
	if len(results) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(results))
	}
	for i, v := range results {
		if v != expected[i] {
			t.Fatalf("results[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestStream_CopyClosePartial(t *testing.T) {
	sr, sw := antsx.Pipe[int](10)

	go func() {
		defer sw.Close()
		for i := 0; i < 5; i++ {
			sw.Send(i, nil)
		}
	}()

	copies := sr.Copy(3)
	copies[0].Close()

	r1 := drainReader(t, copies[1])
	r2 := drainReader(t, copies[2])

	if len(r1) != 5 || len(r2) != 5 {
		t.Fatalf("expected 5 items each, got %d and %d", len(r1), len(r2))
	}
}

func TestStream_SetAutomaticClose_MultiReader(t *testing.T) {
	sr1, sw1 := antsx.Pipe[int](5)
	sr2, sw2 := antsx.Pipe[int](5)

	go func() {
		defer sw1.Close()
		sw1.Send(1, nil)
	}()
	go func() {
		defer sw2.Close()
		sw2.Send(2, nil)
	}()

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2})
	merged.SetAutomaticClose()

	results := drainReader(t, merged)
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
}

func TestStream_SetAutomaticClose_ConvertReader(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		return fmt.Sprintf("%d", i), nil
	})
	converted.SetAutomaticClose()
	defer converted.Close()

	results := drainReader(t, converted)
	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}
}

func TestStream_SetAutomaticClose_ArrayReader(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	sr.SetAutomaticClose()
	defer sr.Close()

	results := drainReader(t, sr)
	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}
}

func TestStream_SetAutomaticClose_GC(t *testing.T) {
	_, sw := antsx.Pipe[int](1)

	sendDone := make(chan struct{})
	go func() {
		defer close(sendDone)
		for i := 0; i < 100; i++ {
			if sw.Send(i, nil) {
				return
			}
		}
		sw.Close()
	}()

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	runtime.GC()

	select {
	case <-sendDone:
	case <-time.After(2 * time.Second):
	}
}

func TestStream_ConvertWithUpstreamError(t *testing.T) {
	sr, sw := antsx.Pipe[int](5)
	testErr := errors.New("upstream error")

	go func() {
		defer sw.Close()
		sw.Send(1, nil)
		sw.Send(0, testErr)
		sw.Send(2, nil)
	}()

	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		return fmt.Sprintf("v%d", i), nil
	})
	defer converted.Close()

	v, err := converted.Recv()
	if err != nil || v != "v1" {
		t.Fatalf("expected v1, got %q, err=%v", v, err)
	}

	_, err = converted.Recv()
	if !errors.Is(err, testErr) {
		t.Fatalf("expected testErr, got %v", err)
	}
}

func TestStream_ConvertWithUpstreamErrorWrapper_Skip(t *testing.T) {
	sr, sw := antsx.Pipe[int](5)
	testErr := errors.New("upstream error")

	go func() {
		defer sw.Close()
		sw.Send(1, nil)
		sw.Send(0, testErr)
		sw.Send(2, nil)
	}()

	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		return fmt.Sprintf("v%d", i), nil
	}, antsx.WithErrWrapper(func(err error) error {
		return nil
	}))
	defer converted.Close()

	results := drainReader(t, converted)
	if len(results) != 2 || results[0] != "v1" || results[1] != "v2" {
		t.Fatalf("unexpected: %v", results)
	}
}

func TestStream_ConvertWithUpstreamErrorWrapper_Wrap(t *testing.T) {
	sr, sw := antsx.Pipe[int](5)
	testErr := errors.New("upstream error")
	wrappedErr := errors.New("wrapped")

	go func() {
		defer sw.Close()
		sw.Send(1, nil)
		sw.Send(0, testErr)
		sw.Send(2, nil)
	}()

	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		return fmt.Sprintf("v%d", i), nil
	}, antsx.WithErrWrapper(func(err error) error {
		return wrappedErr
	}))
	defer converted.Close()

	v, err := converted.Recv()
	if err != nil || v != "v1" {
		t.Fatalf("expected v1, got %q, err=%v", v, err)
	}

	_, err = converted.Recv()
	if !errors.Is(err, wrappedErr) {
		t.Fatalf("expected wrappedErr, got %v", err)
	}
}

func TestStream_ConvertWrappedErrNoValue(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{0, 1, 2, 3})
	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		if i == 0 {
			return "", fmt.Errorf("wrapped: %w", antsx.ErrNoValue)
		}
		return fmt.Sprintf("v%d", i), nil
	})
	defer converted.Close()

	results := drainReader(t, converted)
	if len(results) != 3 || results[0] != "v1" {
		t.Fatalf("wrapped ErrNoValue should be filtered, got: %v", results)
	}
}

func TestStream_MergeConvertAndStream(t *testing.T) {
	sr1 := antsx.StreamReaderFromArray([]int{1, 2, 3})
	converted := antsx.StreamReaderWithConvert(sr1, func(i int) (int, error) {
		return i * 10, nil
	})

	sr2, sw2 := antsx.Pipe[int](5)
	go func() {
		defer sw2.Close()
		sw2.Send(100, nil)
	}()

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{converted, sr2})
	defer merged.Close()

	results := drainReader(t, merged)
	if len(results) != 4 {
		t.Fatalf("expected 4 items, got %d: %v", len(results), results)
	}
}

func TestStream_MergeChildReader(t *testing.T) {
	sr, sw := antsx.Pipe[int](10)
	go func() {
		defer sw.Close()
		for i := 0; i < 5; i++ {
			sw.Send(i, nil)
		}
	}()

	copies := sr.Copy(2)
	sr3 := antsx.StreamReaderFromArray([]int{100})

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{copies[0], sr3})
	defer merged.Close()
	defer copies[1].Close()

	results := drainReader(t, merged)
	if len(results) != 6 {
		t.Fatalf("expected 6 items, got %d", len(results))
	}
}

func TestStream_MergeOnlyArrays_Empty(t *testing.T) {
	sr1 := antsx.StreamReaderFromArray([]int{})
	sr2 := antsx.StreamReaderFromArray([]int{})

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2})
	if merged != nil {
		t.Fatal("merge of empty arrays should return nil")
	}
}

func TestStream_CopyCloseChild_Idempotent(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	copies := sr.Copy(2)

	copies[0].Close()
	copies[0].Close()
	copies[0].Close()

	results := drainReader(t, copies[1])
	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}
	copies[1].Close()
}

func TestStream_CopyConvertReader(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
	converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
		return fmt.Sprintf("v%d", i), nil
	})

	copies := converted.Copy(2)
	defer copies[0].Close()
	defer copies[1].Close()

	r1 := drainReader(t, copies[0])
	r2 := drainReader(t, copies[1])

	if len(r1) != 3 || len(r2) != 3 {
		t.Fatalf("expected 3 items each, got %d and %d", len(r1), len(r2))
	}
	for i := 0; i < 3; i++ {
		expected := fmt.Sprintf("v%d", i+1)
		if r1[i] != expected || r2[i] != expected {
			t.Fatalf("mismatch at %d: r1=%s, r2=%s, want %s", i, r1[i], r2[i], expected)
		}
	}
}

func TestStream_ArrayReaderToStream(t *testing.T) {
	sr1 := antsx.StreamReaderFromArray([]int{1, 2, 3})
	sr2, sw2 := antsx.Pipe[int](5)
	go func() {
		defer sw2.Close()
		sw2.Send(4, nil)
		sw2.Send(5, nil)
	}()

	merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2})
	defer merged.Close()

	results := drainReader(t, merged)
	if len(results) != 5 {
		t.Fatalf("expected 5 items, got %d: %v", len(results), results)
	}
}

func TestStream_MergeNamedWithArrayReader(t *testing.T) {
	sr := antsx.StreamReaderFromArray([]int{1, 2, 3})

	merged := antsx.MergeNamedStreamReaders(map[string]*antsx.StreamReader[int]{
		"arr": sr,
	})
	defer merged.Close()

	var values []int
	var eofs []string
	for {
		v, err := merged.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if name, ok := antsx.GetSourceName(err); ok {
				eofs = append(eofs, name)
				continue
			}
			t.Fatal(err)
		}
		values = append(values, v)
	}
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}
	if len(eofs) != 1 || eofs[0] != "arr" {
		t.Fatalf("expected 1 eof for 'arr', got %v", eofs)
	}
}

func TestStream_MergeNamedSingle(t *testing.T) {
	sr, sw := antsx.Pipe[string](5)
	go func() {
		defer sw.Close()
		sw.Send("hello", nil)
	}()

	merged := antsx.MergeNamedStreamReaders(map[string]*antsx.StreamReader[string]{
		"only": sr,
	})
	defer merged.Close()

	var values []string
	var eofs []string
	for {
		v, err := merged.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if name, ok := antsx.GetSourceName(err); ok {
				eofs = append(eofs, name)
				continue
			}
			t.Fatal(err)
		}
		values = append(values, v)
	}
	if len(values) != 1 || values[0] != "hello" {
		t.Fatalf("unexpected values: %v", values)
	}
	if len(eofs) != 1 || eofs[0] != "only" {
		t.Fatalf("unexpected eofs: %v", eofs)
	}
}
