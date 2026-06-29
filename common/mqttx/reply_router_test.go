package mqttx

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"zero-service/common/antsx"

	"github.com/zeromicro/go-zero/core/stat"
)

func TestReplyDecoderFuncDecode(t *testing.T) {
	decoder := ReplyDecoderFunc[string](func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: string(payload), Value: topic + ":" + topicTemplate}, nil
	})

	msg, err := decoder.Decode(context.Background(), []byte("tid-1"), "reply/1", "reply/+")
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if msg.Tid != "tid-1" || msg.Value != "reply/1:reply/+" {
		t.Fatalf("unexpected decoded message: %+v", msg)
	}
}

func TestReplyRouterResolvesPendingTid(t *testing.T) {
	router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	})
	defer router.Close()

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := router.requestReply(context.Background(), "tid-1", func() error { return nil })
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	waitUntil(t, func() bool { return router.has("tid-1") })
	resolved, err := router.handleReply(context.Background(), []byte("{}"), "reply/1", "reply/+")
	if err != nil {
		t.Fatalf("handleReply returned error: %v", err)
	}
	if !resolved {
		t.Fatalf("expected reply to resolve pending tid")
	}

	select {
	case result := <-resultCh:
		if result != "ok" {
			t.Fatalf("expected ok result, got %s", result)
		}
	case err := <-errCh:
		t.Fatalf("requestReply returned error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reply")
	}
}

func TestReplyRouterConsumeReturnsNotMatched(t *testing.T) {
	router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "missing", Value: "late"}, nil
	})
	defer router.Close()

	err := router.Consume(context.Background(), nil, "reply/1", "reply/+")
	if !errors.Is(err, ErrReplyNotMatched) {
		t.Fatalf("expected ErrReplyNotMatched, got %v", err)
	}
}

func TestReplyRouterHandleReplyErrors(t *testing.T) {
	t.Run("nil decoder", func(t *testing.T) {
		router := NewReplyRouter[string](nil)
		defer router.Close()
		_, err := router.handleReply(context.Background(), nil, "reply/1", "reply/+")
		if !errors.Is(err, ErrNilDecoder) {
			t.Fatalf("expected ErrNilDecoder, got %v", err)
		}
	})

	t.Run("decoder error", func(t *testing.T) {
		decodeErr := errors.New("decode failed")
		router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
			return ReplyMessage[string]{}, decodeErr
		})
		defer router.Close()
		_, err := router.handleReply(context.Background(), nil, "reply/1", "reply/+")
		if !errors.Is(err, decodeErr) {
			t.Fatalf("expected decode error, got %v", err)
		}
	})

	t.Run("empty tid", func(t *testing.T) {
		router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
			return ReplyMessage[string]{Value: "missing tid"}, nil
		})
		defer router.Close()
		_, err := router.handleReply(context.Background(), nil, "reply/1", "reply/+")
		if !errors.Is(err, ErrEmptyReplyTid) {
			t.Fatalf("expected ErrEmptyReplyTid, got %v", err)
		}
	})
}

func TestReplyRouterRejectAndClosePending(t *testing.T) {
	t.Run("reject", func(t *testing.T) {
		router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
			return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
		})
		defer router.Close()
		rejectErr := errors.New("rejected")
		errCh := make(chan error, 1)
		go func() {
			_, err := router.requestReply(context.Background(), "tid-1", func() error { return nil })
			errCh <- err
		}()
		waitUntil(t, func() bool { return router.has("tid-1") })
		if !router.reject("tid-1", rejectErr) {
			t.Fatalf("expected Reject to return true")
		}
		select {
		case err := <-errCh:
			if !errors.Is(err, rejectErr) {
				t.Fatalf("expected reject error, got %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for rejected RequestReply")
		}
	})

	t.Run("close", func(t *testing.T) {
		router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
			return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
		})
		errCh := make(chan error, 1)
		go func() {
			_, err := router.requestReply(context.Background(), "tid-1", func() error { return nil })
			errCh <- err
		}()
		waitUntil(t, func() bool { return router.has("tid-1") })
		router.Close()
		select {
		case err := <-errCh:
			if !errors.Is(err, antsx.ErrReplyClosed) {
				t.Fatalf("expected ErrReplyClosed, got %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for closed RequestReply")
		}
	})
}

func TestWithReplyRouterRegistersReplyTopic(t *testing.T) {
	router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	})
	defer router.Close()
	c := &mqttClient{handlerMgr: newHandlerManager()}

	o := &ClientOptions{}
	WithReplyRouter("reply/+", router)(o)
	for _, reg := range o.replyRouters {
		c.handlerMgr.addReplyHandler(reg.topicTemplate, reg.handler)
	}

	assertTopicTemplateSet(t, c.handlerMgr.getAllTopicTemplates(), []string{"reply/+"})
}

func TestRequestReplyResolvesPendingTid(t *testing.T) {
	router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	})
	defer router.Close()
	c := &mqttClient{handlerMgr: newHandlerManager()}
	c.handlerMgr.addReplyHandler("reply/+", router)

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := RequestReply[string](context.Background(), c, "reply/+", "tid-1", func() error { return nil })
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	waitUntil(t, func() bool { return router.has("tid-1") })
	resolved, err := router.handleReply(context.Background(), []byte("{}"), "reply/1", "reply/+")
	if err != nil {
		t.Fatalf("handleReply returned error: %v", err)
	}
	if !resolved {
		t.Fatalf("expected reply to resolve pending tid")
	}

	select {
	case result := <-resultCh:
		if result != "ok" {
			t.Fatalf("expected ok result, got %v", result)
		}
	case err := <-errCh:
		t.Fatalf("RequestReply returned error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for RequestReply result")
	}
}

func TestRequestReplySendFailureCleansPendingTid(t *testing.T) {
	router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	})
	defer router.Close()
	c := &mqttClient{handlerMgr: newHandlerManager()}
	c.handlerMgr.addReplyHandler("reply/+", router)
	sendErr := errors.New("send failed")

	_, err := RequestReply[string](context.Background(), c, "reply/+", "tid-1", func() error { return sendErr })
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected send error, got %v", err)
	}
	if router.has("tid-1") {
		t.Fatalf("expected failed send to clean pending tid")
	}
}

func TestDispatcherRunsReplyBeforeRegularAndKeepsRegular(t *testing.T) {
	var calls []string
	router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		calls = append(calls, "reply:"+topicTemplate)
		return ReplyMessage[string]{Tid: "unmatched", Value: "late"}, nil
	})
	defer router.Close()
	manager := newHandlerManager()
	manager.addReplyHandler("device/+/reply", router)
	manager.addHandler("device/+/reply", ConsumeHandlerFunc(func(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
		calls = append(calls, "regular:"+topicTemplate)
		return nil
	}))
	dispatcher := newTestDispatcher(manager)

	dispatcher.dispatch(context.Background(), []byte("{}"), "device/1/reply", "device/+/reply")

	expected := []string{"reply:device/+/reply", "regular:device/+/reply"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected calls %v, got %v", expected, calls)
	}
}

func TestDispatcherUsesCallbackTopicTemplate(t *testing.T) {
	manager := newHandlerManager()
	called := false
	manager.addHandler("device/+/reply", ConsumeHandlerFunc(func(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
		called = true
		if topic != "device/1/reply" || topicTemplate != "device/+/reply" {
			t.Fatalf("unexpected topic args: topic=%s template=%s", topic, topicTemplate)
		}
		return nil
	}))
	dispatcher := newTestDispatcher(manager)
	noHandlerCalled := false
	dispatcher.SetNoHandlerHandler(func(ctx context.Context, payload []byte, topic string, topicTemplate string) {
		noHandlerCalled = true
	})

	dispatcher.dispatch(context.Background(), []byte("{}"), "device/1/reply", "device/+/reply")
	if !called {
		t.Fatalf("expected handler to match callback topic template")
	}

	dispatcher.dispatch(context.Background(), []byte("{}"), "device/1/reply", "other/template")
	if !noHandlerCalled {
		t.Fatalf("expected no handler callback")
	}
}

func TestHandlerOrderForSameTopic(t *testing.T) {
	manager := newHandlerManager()
	var calls []string
	manager.addHandler("device/+", ConsumeHandlerFunc(func(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
		calls = append(calls, "first")
		return nil
	}))
	manager.addHandler("device/+", ConsumeHandlerFunc(func(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
		calls = append(calls, "second")
		return nil
	}))

	newTestDispatcher(manager).dispatch(context.Background(), []byte("{}"), "device/1", "device/+")

	expected := []string{"first", "second"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected calls %v, got %v", expected, calls)
	}
}

func TestHandlerManagerGetAllTopicTemplatesIncludesRegularAndReplyTemplates(t *testing.T) {
	manager := newHandlerManager()
	router := newTestReplyRouter(func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	})
	defer router.Close()
	manager.addHandler("device/+", ConsumeHandlerFunc(func(ctx context.Context, payload []byte, topic string, topicTemplate string) error { return nil }))
	manager.addHandler("device/+", ConsumeHandlerFunc(func(ctx context.Context, payload []byte, topic string, topicTemplate string) error { return nil }))
	manager.addReplyHandler("reply/+", router)

	assertTopicTemplateSet(t, manager.getAllTopicTemplates(), []string{"device/+", "reply/+"})
}

func newTestReplyRouter(decode func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[string], error)) *ReplyRouter[string] {
	return NewReplyRouter[string](ReplyDecoderFunc[string](decode), WithReplyRouterName("mqttx-test-reply-router"), WithReplyRouterTTL(time.Second))
}

func newTestDispatcher(manager *handlerManager) *messageDispatcher {
	return newMessageDispatcher(manager, stat.NewMetrics("mqttx-test"))
}

func waitUntil(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func assertTopicTemplateSet(t *testing.T, got []string, want []string) {
	t.Helper()
	gotSet := make(map[string]int, len(got))
	for _, topicTemplate := range got {
		gotSet[topicTemplate]++
	}
	if len(gotSet) != len(want) {
		t.Fatalf("expected %d unique topic templates, got %v", len(want), got)
	}
	for _, topicTemplate := range want {
		if gotSet[topicTemplate] != 1 {
			t.Fatalf("expected topic template %s exactly once, got topic templates %v", topicTemplate, got)
		}
	}
}
