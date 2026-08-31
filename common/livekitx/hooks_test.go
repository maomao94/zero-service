package livekitx

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

func TestChatHooksRunInRegistrationOrderAndCanUnsubscribe(t *testing.T) {
	d := newHookSet()
	var calls []string
	one := d.onChatMessage(func(context.Context, ChatMessageEvent) error {
		calls = append(calls, "one")
		return nil
	})
	d.onChatMessage(func(context.Context, ChatMessageEvent) error {
		calls = append(calls, "two")
		return errors.New("second")
	})
	if err := d.dispatchChatMessage(context.Background(), ChatMessageEvent{Text: "hello"}); err == nil {
		t.Fatal("expected handler error")
	}
	one.Unsubscribe()
	if err := d.dispatchChatMessage(context.Background(), ChatMessageEvent{Text: "again"}); err == nil {
		t.Fatal("expected remaining handler error")
	}
	if !reflect.DeepEqual(calls, []string{"one", "two", "two"}) {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestDispatcherAggregatesErrorsAndContinues(t *testing.T) {
	d := newHookSet()
	first := errors.New("first")
	second := errors.New("second")
	d.onChatMessage(func(context.Context, ChatMessageEvent) error { return first })
	d.onChatMessage(func(context.Context, ChatMessageEvent) error { return second })
	err := d.dispatchChatMessage(context.Background(), ChatMessageEvent{Text: "x"})
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if !errors.Is(err, first) {
		t.Fatalf("aggregated error must wrap first error: %v", err)
	}
	if !errors.Is(err, second) {
		t.Fatalf("aggregated error must wrap second error: %v", err)
	}
}

func TestDispatcherRecoversPanicIntoHookPanicError(t *testing.T) {
	d := newHookSet()
	d.onChatMessage(func(context.Context, ChatMessageEvent) error { panic("boom") })
	err := d.dispatchChatMessage(context.Background(), ChatMessageEvent{Text: "x"})
	if err == nil {
		t.Fatal("expected panic error")
	}
	var panicErr *HookPanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected *HookPanicError, got %T: %v", err, err)
	}
	if panicErr.Event != "chat message" || panicErr.Value != "boom" {
		t.Fatalf("unexpected panic error: %+v", panicErr)
	}
}

func TestDispatcherConcurrentRegisterUnsubscribeDuringDispatch(t *testing.T) {
	d := newHookSet()
	d.onChatMessage(func(context.Context, ChatMessageEvent) error { return nil })
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				sub := d.onChatMessage(func(context.Context, ChatMessageEvent) error { return nil })
				sub.Unsubscribe()
				_ = d.dispatchChatMessage(context.Background(), ChatMessageEvent{Text: "x"})
			}
		}()
	}
	wg.Wait()
}

func TestDispatcherClosedReturnsErrClosed(t *testing.T) {
	d := newHookSet()
	sub := d.onChatMessage(func(context.Context, ChatMessageEvent) error { return nil })
	d.close()
	if err := d.dispatchChatMessage(context.Background(), ChatMessageEvent{}); !errors.Is(err, ErrClosed) {
		t.Fatalf("dispatch after close = %v, want ErrClosed", err)
	}
	// 关闭后注销与注册都不 panic；注册返回空订阅。
	sub.Unsubscribe()
	if closed := d.onChatMessage(func(context.Context, ChatMessageEvent) error { return nil }); closed == nil {
		t.Fatal("expected non-nil no-op subscription")
	}
}

func TestDispatcherAllEventTypesHaveIndependentHandlerSlots(t *testing.T) {
	d := newHookSet()
	var chatCalls, dataCalls int
	d.onChatMessage(func(context.Context, ChatMessageEvent) error { chatCalls++; return nil })
	d.data.add(func(context.Context, DataEvent) error { dataCalls++; return nil })
	_ = d.chat.dispatch(context.Background(), ChatMessageEvent{})
	_ = d.data.dispatch(context.Background(), DataEvent{})
	if chatCalls != 1 || dataCalls != 1 {
		t.Fatalf("chat calls = %d, data calls = %d", chatCalls, dataCalls)
	}
}

func TestClientOnChatMessageRegistersAndCloseStopsDispatch(t *testing.T) {
	client, err := New(WithURL("http://127.0.0.1:7880"), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	sub := client.OnChatMessage(func(context.Context, ChatMessageEvent) error { calls++; return nil })
	if sub == nil {
		t.Fatal("expected subscription")
	}
	sub.Unsubscribe()
	sub.Unsubscribe() // 幂等
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	// 关闭后注册返回空订阅且不 panic。
	_ = client.OnChatMessage(func(context.Context, ChatMessageEvent) error { return nil })
}
