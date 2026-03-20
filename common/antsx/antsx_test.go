package antsx_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
	"zero-service/common/antsx"
)

func TestReactorAndPromise(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()

	// 提交成功任务，返回 int
	p1, err := antsx.Submit(ctx, reactor, "task1", func(ctx context.Context) (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// 链式转换 int -> string
	p2 := antsx.Then(ctx, p1, func(val int) (string, error) {
		return fmt.Sprintf("value is %d", val), nil
	})

	// 捕获错误（正常不会触发）
	p2.Catch(func(err error) {
		t.Errorf("unexpected error: %v", err)
	})

	// 等待结果
	res, err := p2.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if res != "value is 42" {
		t.Fatalf("unexpected result: %s", res)
	}

	t.Logf("Success chain result: %s", res)

	// 测试失败任务
	pFail, err := antsx.Submit(ctx, reactor, "failTask", func(ctx context.Context) (string, error) {
		return "", errors.New("intentional failure")
	})
	if err != nil {
		t.Fatal(err)
	}

	// 捕获失败错误
	caughtCh := make(chan error, 1)
	pFail.Catch(func(err error) {
		caughtCh <- err
	})

	_, err = pFail.Await(ctx)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	select {
	case caughtErr := <-caughtCh:
		if caughtErr == nil || caughtErr.Error() != "intentional failure" {
			t.Errorf("unexpected error in Catch: %v", caughtErr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Catch callback not called within timeout")
	}

	t.Log("Error handling test passed")
}

func TestSubmit_PanicRecovery(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()
	p, err := antsx.Submit(ctx, reactor, "panic-task", func(ctx context.Context) (string, error) {
		panic("submit panic")
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic")
	}
	t.Logf("Submit panic recovered: %v", err)
}

func TestPost(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatalf("Failed to create reactor: %v", err)
	}
	defer r.Release()

	ctx := context.Background()
	called := make(chan struct{}, 1)

	err = antsx.Post(ctx, r, func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		called <- struct{}{}
		return "done", nil
	})

	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	select {
	case <-called:
		t.Logf("Post task executed successfully")
	case <-time.After(1 * time.Second):
		t.Fatalf("Post task did not complete in time")
	}
}

// ======================== PromiseAll / PromiseRace ========================

func TestPromiseAll_AllSuccess(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()

	p1, _ := antsx.Submit(ctx, reactor, "all-1", func(ctx context.Context) (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 1, nil
	})
	p2, _ := antsx.Submit(ctx, reactor, "all-2", func(ctx context.Context) (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 2, nil
	})
	p3, _ := antsx.Submit(ctx, reactor, "all-3", func(ctx context.Context) (int, error) {
		time.Sleep(30 * time.Millisecond)
		return 3, nil
	})

	results, err := antsx.PromiseAll(ctx, p1, p2, p3)
	if err != nil {
		t.Fatalf("PromiseAll failed: %v", err)
	}

	if len(results) != 3 || results[0] != 1 || results[1] != 2 || results[2] != 3 {
		t.Fatalf("unexpected results: %v", results)
	}
	t.Logf("PromiseAll results: %v", results)
}

func TestPromiseAll_OneFail(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()

	p1, _ := antsx.Submit(ctx, reactor, "allf-1", func(ctx context.Context) (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 1, nil
	})
	p2, _ := antsx.Submit(ctx, reactor, "allf-2", func(ctx context.Context) (int, error) {
		time.Sleep(30 * time.Millisecond)
		return 0, errors.New("task2 failed")
	})
	p3, _ := antsx.Submit(ctx, reactor, "allf-3", func(ctx context.Context) (int, error) {
		time.Sleep(200 * time.Millisecond)
		return 3, nil
	})

	_, err = antsx.PromiseAll(ctx, p1, p2, p3)
	if err == nil {
		t.Fatal("expected error from PromiseAll")
	}
	t.Logf("PromiseAll correctly failed: %v", err)
}

func TestPromiseRace(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()

	p1, _ := antsx.Submit(ctx, reactor, "race-1", func(ctx context.Context) (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "slow", nil
	})
	p2, _ := antsx.Submit(ctx, reactor, "race-2", func(ctx context.Context) (string, error) {
		time.Sleep(30 * time.Millisecond)
		return "fast", nil
	})

	result, err := antsx.PromiseRace(ctx, p1, p2)
	if err != nil {
		t.Fatalf("PromiseRace failed: %v", err)
	}
	if result != "fast" {
		t.Fatalf("expected 'fast' but got '%s'", result)
	}
	t.Logf("PromiseRace winner: %s", result)
}

// ======================== Map / FlatMap ========================

func TestMap(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]("map-1")
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.Resolve(10)
	}()

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
	t.Logf("Map result: %s", result)
}

func TestFlatMap(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]("flatmap-1")
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.Resolve(5)
	}()

	flatMapped := antsx.FlatMap(ctx, p, func(v int) *antsx.Promise[string] {
		inner := antsx.NewPromise[string]("flatmap-inner")
		go func() {
			time.Sleep(30 * time.Millisecond)
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
	t.Logf("FlatMap result: %s", result)
}

// ======================== AwaitWithTimeout ========================

func TestAwaitWithTimeout_Success(t *testing.T) {
	p := antsx.NewPromise[int]("timeout-1")
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
	p := antsx.NewPromise[int]("timeout-2")
	// 不 Resolve，让它超时

	_, err := p.AwaitWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	t.Logf("Timeout correctly triggered: %v", err)
}

// ======================== EventEmitter ========================

func TestEventEmitter_SubscribeEmit(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	defer emitter.Close()

	ch, cancel := emitter.Subscribe("topic1")
	defer cancel()

	emitter.Emit("topic1", "hello")
	emitter.Emit("topic1", "world")

	msg1 := <-ch
	msg2 := <-ch

	if msg1 != "hello" || msg2 != "world" {
		t.Fatalf("unexpected messages: %s, %s", msg1, msg2)
	}
	t.Logf("EventEmitter received: %s, %s", msg1, msg2)
}

func TestEventEmitter_MultipleSubscribers(t *testing.T) {
	emitter := antsx.NewEventEmitter[int]()
	defer emitter.Close()

	ch1, cancel1 := emitter.Subscribe("multi")
	defer cancel1()
	ch2, cancel2 := emitter.Subscribe("multi")
	defer cancel2()

	emitter.Emit("multi", 42)

	v1 := <-ch1
	v2 := <-ch2

	if v1 != 42 || v2 != 42 {
		t.Fatalf("expected 42 for both, got %d and %d", v1, v2)
	}

	if emitter.SubscriberCount("multi") != 2 {
		t.Fatalf("expected 2 subscribers, got %d", emitter.SubscriberCount("multi"))
	}
}

func TestEventEmitter_Cancel(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	defer emitter.Close()

	_, cancel := emitter.Subscribe("cancel-topic")

	if emitter.SubscriberCount("cancel-topic") != 1 {
		t.Fatal("expected 1 subscriber")
	}

	cancel()

	if emitter.SubscriberCount("cancel-topic") != 0 {
		t.Fatal("expected 0 subscribers after cancel")
	}

	if emitter.TopicCount() != 0 {
		t.Fatal("expected 0 topics after all subscribers cancelled")
	}
}

// ======================== Integration: EventEmitter ========================

func TestEmitterToStream(t *testing.T) {
	emitter := antsx.NewEventEmitter[string]()
	defer emitter.Close()

	ch, cancel := emitter.Subscribe("events")
	defer cancel()

	var count atomic.Int32

	// 模拟 SSE 消费循环
	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range ch {
			count.Add(1)
			_ = msg
			if count.Load() >= 3 {
				return
			}
		}
	}()

	emitter.Emit("events", "msg1")
	emitter.Emit("events", "msg2")
	emitter.Emit("events", "msg3")

	select {
	case <-done:
		if count.Load() != 3 {
			t.Fatalf("expected 3 messages, got %d", count.Load())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for messages")
	}
	t.Logf("Emitter->Consumer integration: %d messages received", count.Load())
}

// ======================== Then/Map/FlatMap Panic Recovery ========================

func TestThen_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]("then-panic")
	go func() { p.Resolve(42) }()

	p2 := antsx.Then(ctx, p, func(v int) (string, error) {
		panic("then boom")
	})

	_, err := p2.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic in Then")
	}
	t.Logf("Then panic recovered: %v", err)
}

func TestMap_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]("map-panic")
	go func() { p.Resolve(10) }()

	mapped := antsx.Map(ctx, p, func(v int) string {
		panic("map boom")
	})

	_, err := mapped.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic in Map")
	}
	t.Logf("Map panic recovered: %v", err)
}

func TestFlatMap_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	p := antsx.NewPromise[int]("flatmap-panic")
	go func() { p.Resolve(5) }()

	flatMapped := antsx.FlatMap(ctx, p, func(v int) *antsx.Promise[string] {
		panic("flatmap boom")
	})

	_, err := flatMapped.Await(ctx)
	if err == nil {
		t.Fatal("expected error from panic in FlatMap")
	}
	t.Logf("FlatMap panic recovered: %v", err)
}
