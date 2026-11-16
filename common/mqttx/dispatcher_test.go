package mqttx

import (
	"context"
	"errors"
	"testing"
)

func TestDispatcherCallsNoHandlerOnlyWhenNothingRegistered(t *testing.T) {
	manager := newHandlerManager()
	dispatcher := newTestDispatcher(manager)
	var calls int
	dispatcher.SetNoHandlerHandler(func(context.Context, []byte, string, string) {
		calls++
	})

	dispatcher.dispatch(context.Background(), []byte("{}"), "device/1", "device/+")
	if calls != 1 {
		t.Fatalf("no-handler calls = %d, want 1", calls)
	}

	manager.addHandler("device/+", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		return nil
	}))
	dispatcher.dispatch(context.Background(), []byte("{}"), "device/1", "device/+")
	if calls != 1 {
		t.Fatalf("no-handler calls = %d, want 1 after handler registration", calls)
	}
}

func TestDispatcherDefaultNoHandlerIsSafe(t *testing.T) {
	newTestDispatcher(newHandlerManager()).dispatch(context.Background(), []byte("{}"), "device/1", "device/+")
}

func TestDispatcherSetNoHandlerIgnoresNil(t *testing.T) {
	dispatcher := newTestDispatcher(newHandlerManager())
	dispatcher.SetNoHandlerHandler(nil)
	if dispatcher.onNoHandler == nil {
		t.Fatal("nil callback must not clear the default no-handler handler")
	}
}

func TestDispatcherRunsRegularHandlersWhenReplyFails(t *testing.T) {
	manager := newHandlerManager()
	decodeErr := errors.New("decode failed")
	router := newTestReplyRouter(func(context.Context, []byte, string, string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{}, decodeErr
	})
	defer router.Close()
	manager.addReplyHandler("device/+/reply", router)
	called := false
	manager.addHandler("device/+/reply", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		called = true
		return errors.New("regular handler failed")
	}))

	newTestDispatcher(manager).dispatch(context.Background(), []byte("{}"), "device/1/reply", "device/+/reply")
	if !called {
		t.Fatal("expected regular handler to run after reply handler error")
	}
}

func TestDispatcherToleratesUnmatchedReply(t *testing.T) {
	manager := newHandlerManager()
	router := newTestReplyRouter(func(context.Context, []byte, string, string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "late", Value: "late"}, nil
	})
	defer router.Close()
	manager.addReplyHandler("device/+/reply", router)
	called := false
	manager.addHandler("device/+/reply", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		called = true
		return nil
	}))

	newTestDispatcher(manager).dispatch(context.Background(), []byte("{}"), "device/1/reply", "device/+/reply")
	if !called {
		t.Fatal("expected regular handler to run for unmatched reply")
	}
}

func TestHandlerManagerGetHandlersReturnsCopy(t *testing.T) {
	manager := newHandlerManager()
	manager.addHandler("device/+", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		return nil
	}))

	handlers := manager.getHandlers("device/+")
	handlers[0] = nil

	if got := manager.getHandlers("device/+"); len(got) != 1 || got[0] == nil {
		t.Fatal("getHandlers must return a copy of the handler list")
	}
}

func TestCloseReplyHandlersClosesClosers(t *testing.T) {
	manager := newHandlerManager()
	first := &closeCountingHandler{}
	second := &closeCountingHandler{}
	manager.addReplyHandler("reply/a", first)
	manager.addReplyHandler("reply/b", second)
	manager.addHandler("device/+", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		return nil
	}))

	manager.closeReplyHandlers()

	if first.closed != 1 || second.closed != 1 {
		t.Fatalf("closed counts = [%d %d], want [1 1]", first.closed, second.closed)
	}
}
