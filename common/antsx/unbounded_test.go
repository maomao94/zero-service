package antsx_test

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
	"zero-service/common/antsx"
)

func TestUnboundedChan_Basic(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()

	go func() {
		for i := 0; i < 10; i++ {
			ch.Send(i)
		}
		ch.Close()
	}()

	var results []int
	for {
		v, ok := ch.Receive()
		if !ok {
			break
		}
		results = append(results, v)
	}
	if len(results) != 10 {
		t.Fatalf("expected 10 items, got %d", len(results))
	}
}

func TestUnboundedChan_CloseReceive(t *testing.T) {
	ch := antsx.NewUnboundedChan[string]()
	ch.Close()

	_, ok := ch.Receive()
	if ok {
		t.Fatal("expected ok=false after close")
	}
}

func TestUnboundedChan_CloseSendPanic(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	ch.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on send after close")
		}
	}()
	ch.Send(42)
}

func TestUnboundedChan_TrySend_Success(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	ok := ch.TrySend(42)
	if !ok {
		t.Fatal("expected TrySend to succeed")
	}

	v, ok := ch.Receive()
	if !ok || v != 42 {
		t.Fatalf("expected (42, true), got (%d, %v)", v, ok)
	}
}

func TestUnboundedChan_TrySend_Closed(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	ch.Close()

	ok := ch.TrySend(42)
	if ok {
		t.Fatal("expected TrySend to return false on closed channel")
	}
}

func TestUnboundedChan_ReceiveContext_Success(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	ch.Send(99)

	v, ok := ch.ReceiveContext(context.Background())
	if !ok || v != 99 {
		t.Fatalf("expected (99, true), got (%d, %v)", v, ok)
	}
}

func TestUnboundedChan_ReceiveContext_Cancel(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, ok := ch.ReceiveContext(ctx)
	if ok {
		t.Fatal("expected ok=false after context cancel")
	}
}

func TestUnboundedChan_ReceiveContext_DataBeforeCancel(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		ch.Send(77)
	}()

	v, ok := ch.ReceiveContext(ctx)
	if !ok || v != 77 {
		t.Fatalf("expected (77, true), got (%d, %v)", v, ok)
	}
}

func TestUnboundedChan_Len(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	if ch.Len() != 0 {
		t.Fatalf("expected 0, got %d", ch.Len())
	}

	ch.Send(1)
	ch.Send(2)
	ch.Send(3)

	if ch.Len() != 3 {
		t.Fatalf("expected 3, got %d", ch.Len())
	}

	ch.Receive()
	if ch.Len() != 2 {
		t.Fatalf("expected 2, got %d", ch.Len())
	}
}

func TestUnboundedChan_Concurrent(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	const producers = 5
	const perProducer = 100

	var wg sync.WaitGroup
	wg.Add(producers)
	for p := 0; p < producers; p++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perProducer; i++ {
				ch.Send(i)
			}
		}()
	}

	go func() {
		wg.Wait()
		ch.Close()
	}()

	count := 0
	for {
		_, ok := ch.Receive()
		if !ok {
			break
		}
		count++
	}

	expected := producers * perProducer
	if count != expected {
		t.Fatalf("expected %d items, got %d", expected, count)
	}
}

func TestUnboundedChan_CloseIdempotent(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	ch.Close()
	ch.Close()
	ch.Close()
}

func TestUnboundedChan_MemoryRelease(t *testing.T) {
	ch := antsx.NewUnboundedChan[*[1024]byte]()

	for i := 0; i < 1000; i++ {
		data := new([1024]byte)
		ch.Send(data)
	}

	for i := 0; i < 1000; i++ {
		_, ok := ch.Receive()
		if !ok {
			t.Fatal("expected data")
		}
	}

	ch.Send(new([1024]byte))
	v, ok := ch.Receive()
	if !ok || v == nil {
		t.Fatal("expected non-nil value")
	}
	ch.Close()
}

func TestUnboundedChan_ReceiveContext_MultiGoroutine(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	const n = 10
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			ch.ReceiveContext(ctx)
		}()
	}

	time.Sleep(50 * time.Millisecond)
	for i := 0; i < n; i++ {
		ch.Send(i)
	}
	wg.Wait()
}

func TestUnboundedChan_TrySend_Concurrent(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			ch.TrySend(val)
		}(i)
	}
	wg.Wait()

	count := 0
	for {
		if ch.Len() == 0 {
			break
		}
		_, ok := ch.Receive()
		if !ok {
			break
		}
		count++
	}
	if count != 50 {
		t.Fatalf("expected 50 items, got %d", count)
	}
}

func TestUnboundedChan_BufferCompact(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	for i := 0; i < 1000; i++ {
		ch.Send(i)
	}

	for i := 0; i < 1000; i++ {
		v, ok := ch.Receive()
		if !ok {
			t.Fatalf("expected value at %d", i)
		}
		if v != i {
			t.Fatalf("expected %d, got %d", i, v)
		}
	}

	ch.Send(9999)
	v, ok := ch.Receive()
	if !ok || v != 9999 {
		t.Fatalf("expected (9999, true), got (%d, %v)", v, ok)
	}

	if ch.Len() != 0 {
		t.Fatalf("expected empty after drain, got %d", ch.Len())
	}
}

func TestUnboundedChan_ReceiveContext_NoGoroutineLeak(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	defer ch.Close()

	before := runtime.NumGoroutine()

	ch.Send(1)
	ch.Send(2)
	ch.Send(3)

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		v, ok := ch.ReceiveContext(ctx)
		cancel()
		if !ok {
			t.Fatalf("expected value at %d", i)
		}
		_ = v
	}

	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Fatalf("potential goroutine leak: before=%d, after=%d", before, after)
	}
}

func TestUnboundedChan_MPMC(t *testing.T) {
	ch := antsx.NewUnboundedChan[int]()
	const producers = 5
	const consumers = 3
	const perProducer = 200

	var sendWg sync.WaitGroup
	sendWg.Add(producers)
	for p := 0; p < producers; p++ {
		go func(id int) {
			defer sendWg.Done()
			for i := 0; i < perProducer; i++ {
				ch.Send(id*perProducer + i)
			}
		}(p)
	}

	go func() {
		sendWg.Wait()
		ch.Close()
	}()

	var mu sync.Mutex
	total := 0
	var recvWg sync.WaitGroup
	recvWg.Add(consumers)
	for c := 0; c < consumers; c++ {
		go func() {
			defer recvWg.Done()
			localCount := 0
			for {
				_, ok := ch.Receive()
				if !ok {
					break
				}
				localCount++
			}
			mu.Lock()
			total += localCount
			mu.Unlock()
		}()
	}
	recvWg.Wait()

	expected := producers * perProducer
	if total != expected {
		t.Fatalf("expected %d items, got %d", expected, total)
	}
}
