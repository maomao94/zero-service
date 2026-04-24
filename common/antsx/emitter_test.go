package antsx_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"
	"zero-service/common/antsx"
)

func TestEventEmitter_SubscribeEmit(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	defer emitter.Close()

	sr, cancel := emitter.Subscribe(context.Background(), "test-topic")
	defer cancel()

	emitter.Emit("test-topic", "hello")
	emitter.Emit("test-topic", "world")

	val, err := sr.Recv()
	if err != nil || val != "hello" {
		t.Fatalf("expected hello, got %q, err=%v", val, err)
	}

	val, err = sr.Recv()
	if err != nil || val != "world" {
		t.Fatalf("expected world, got %q, err=%v", val, err)
	}
}

func TestEventEmitter_MultipleSubscribers(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	defer emitter.Close()

	sr1, cancel1 := emitter.Subscribe(context.Background(), "topic")
	sr2, cancel2 := emitter.Subscribe(context.Background(), "topic")
	defer cancel1()
	defer cancel2()

	emitter.Emit("topic", 42)

	v1, _ := sr1.Recv()
	v2, _ := sr2.Recv()
	if v1 != 42 || v2 != 42 {
		t.Fatalf("expected 42/42, got %d/%d", v1, v2)
	}

	if emitter.SubscriberCount("topic") != 2 {
		t.Fatalf("expected 2 subscribers, got %d", emitter.SubscriberCount("topic"))
	}
	if emitter.TopicCount() != 1 {
		t.Fatalf("expected 1 topic, got %d", emitter.TopicCount())
	}
}

func TestEventEmitter_Cancel(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	defer emitter.Close()

	sr, cancel := emitter.Subscribe(context.Background(), "topic")
	cancel()

	_, err := sr.Recv()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF after cancel, got %v", err)
	}

	if emitter.SubscriberCount("topic") != 0 {
		t.Fatalf("expected 0 subscribers after cancel, got %d", emitter.SubscriberCount("topic"))
	}
}

func TestEventEmitter_CancelIdempotent(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	defer emitter.Close()

	_, cancel := emitter.Subscribe(context.Background(), "topic")
	cancel()
	cancel()
	cancel()
}

func TestEventEmitter_CloseStopsAll(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	sr, _ := emitter.Subscribe(context.Background(), "topic")
	emitter.Close()

	_, err := sr.Recv()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF after close, got %v", err)
	}
}

func TestEventEmitter_CloseIdempotent(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	emitter.Close()
	emitter.Close()
	emitter.Close()
}

func TestEventEmitter_SubscribeAfterClose(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	emitter.Close()

	sr, cancel := emitter.Subscribe(context.Background(), "topic")
	defer cancel()

	_, err := sr.Recv()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF for subscribe after close, got %v", err)
	}
}

func TestEventEmitter_CtxCancel(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	defer emitter.Close()

	ctx, ctxCancel := context.WithCancel(context.Background())
	sr, _ := emitter.Subscribe(ctx, "topic")

	ctxCancel()
	time.Sleep(50 * time.Millisecond)

	_, err := sr.Recv()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF after ctx cancel, got %v", err)
	}

	if emitter.SubscriberCount("topic") != 0 {
		t.Fatalf("expected 0 subscribers after ctx cancel, got %d", emitter.SubscriberCount("topic"))
	}
}

func TestEventEmitter_MultiTopicConcurrent(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	defer emitter.Close()

	const topics = 5
	const msgs = 20

	var wg sync.WaitGroup
	for i := 0; i < topics; i++ {
		topic := string(rune('A' + i))
		sr, cancel := emitter.Subscribe(context.Background(), topic)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			count := 0
			for {
				_, err := sr.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Errorf("topic %s: %v", topic, err)
					return
				}
				count++
				if count == msgs {
					return
				}
			}
		}()
	}

	for i := 0; i < topics; i++ {
		topic := string(rune('A' + i))
		for j := 0; j < msgs; j++ {
			emitter.Emit(topic, j)
		}
	}

	wg.Wait()
}

func TestEventEmitter_ConcurrentEmitSubscribeCancel(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	defer emitter.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sr, cancel := emitter.Subscribe(context.Background(), "race")
			go func() {
				for {
					_, err := sr.Recv()
					if errors.Is(err, io.EOF) {
						return
					}
				}
			}()
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				emitter.Emit("race", j)
			}
		}()
	}
	wg.Wait()
}

func TestEventEmitter_EmitNoSubscribers(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	defer emitter.Close()
	emitter.Emit("nonexistent", 42)
}

func TestEventEmitter_CustomBufSize(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	defer emitter.Close()

	sr, cancel := emitter.Subscribe(context.Background(), "topic", 2)
	defer cancel()

	emitter.Emit("topic", 1)
	emitter.Emit("topic", 2)

	v, _ := sr.Recv()
	if v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
}

func TestEmitter_ConcurrentSubscribeEmitClose(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sr, cancel := emitter.Subscribe(ctx, fmt.Sprintf("t%d", id%3))
			defer cancel()
			for j := 0; j < 5; j++ {
				_, err := sr.Recv()
				if err != nil {
					return
				}
			}
		}(i)
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				emitter.Emit(fmt.Sprintf("t%d", id), j)
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	time.Sleep(100 * time.Millisecond)
	emitter.Close()
	wg.Wait()
}

func TestEmitter_GoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	emitter := antsx.NewEventEmitter[int]()
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sr, unsub := emitter.Subscribe(ctx, "test")
			defer unsub()
			for {
				_, err := sr.Recv()
				if err != nil {
					return
				}
			}
		}()
	}

	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
	emitter.Close()
	wg.Wait()

	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+3 {
		t.Fatalf("potential goroutine leak: before=%d, after=%d", before, after)
	}
}

func TestEmitter_EmitAfterClose(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	emitter.Close()
	emitter.Emit("topic", 1)
	emitter.Close()
}
