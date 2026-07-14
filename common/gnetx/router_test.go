package gnetx

import (
	"context"
	"errors"
	"testing"
)

func TestRouterRouteByID(t *testing.T) {
	r := NewRouter()
	r.RegisterFunc(1, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "handler-1", nil
	})
	r.RegisterFunc(2, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "handler-2", nil
	})

	reply, err := r.Handle(context.Background(), nil, testMsg{id: 1})
	if err != nil || reply != "handler-1" {
		t.Fatalf("id=1: reply=%v err=%v", reply, err)
	}
	reply, err = r.Handle(context.Background(), nil, testMsg{id: 2})
	if err != nil || reply != "handler-2" {
		t.Fatalf("id=2: reply=%v err=%v", reply, err)
	}
}

func TestRouterNoHandler(t *testing.T) {
	r := NewRouter()
	_, err := r.Handle(context.Background(), nil, testMsg{id: 999})
	if !errors.Is(err, ErrNoHandler) {
		t.Fatalf("want ErrNoHandler, got %v", err)
	}
}

func TestRouterFallback(t *testing.T) {
	r := NewRouter()
	r.FallbackFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		return "fallback", nil
	})
	reply, err := r.Handle(context.Background(), nil, testMsg{id: 999})
	if err != nil || reply != "fallback" {
		t.Fatalf("fallback: reply=%v err=%v", reply, err)
	}
}

func TestRouterMsgNotIdentifiable(t *testing.T) {
	r := NewRouter()
	r.RegisterFunc(1, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "handler-1", nil
	})
	_, err := r.Handle(context.Background(), nil, "plain-string")
	if !errors.Is(err, ErrNoHandler) {
		t.Fatalf("want ErrNoHandler for non-identifiable, got %v", err)
	}

	r.FallbackFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		return "fb", nil
	})
	reply, err := r.Handle(context.Background(), nil, "plain-string")
	if err != nil || reply != "fb" {
		t.Fatalf("non-identifiable with fallback: reply=%v err=%v", reply, err)
	}
}

func TestHandleTyped(t *testing.T) {
	r := NewRouter()
	HandleTyped(r, 1, func(ctx context.Context, c Conn, msg testMsg) (any, error) {
		return msg.id * 10, nil
	})

	reply, err := r.Handle(context.Background(), nil, testMsg{id: 1})
	if err != nil || reply != 10 {
		t.Fatalf("typed: reply=%v err=%v", reply, err)
	}

	_, err = r.Handle(context.Background(), nil, testClient{cid: "x"})
	if err == nil {
		t.Fatal("type mismatch should error")
	}
}

func TestRouterResolve(t *testing.T) {
	r := NewRouter()
	r.RegisterFunc(42, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "resolved", nil
	})

	h, err := r.Resolve(testMsg{id: 42})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	reply, err := h.Handle(context.Background(), nil, nil)
	if err != nil || reply != "resolved" {
		t.Fatalf("resolved handler: reply=%v err=%v", reply, err)
	}

	_, err = r.Resolve(testMsg{id: 999})
	if !errors.Is(err, ErrNoHandler) {
		t.Fatalf("unregistered id: want ErrNoHandler, got %v", err)
	}
}

func TestRouterAsyncEntryIsAsyncHandler(t *testing.T) {
	r := NewRouter()
	r.RegisterFunc(1, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "sync", nil
	})
	r.Async(1)

	h, err := r.Resolve(testMsg{id: 1})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !isAsync(h) {
		t.Fatal("resolved handler after Async() should be recognized as async")
	}
}

func TestRouterSyncEntryNotAsync(t *testing.T) {
	r := NewRouter()
	r.RegisterFunc(1, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "sync", nil
	})

	h, err := r.Resolve(testMsg{id: 1})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if isAsync(h) {
		t.Fatal("plain handler should not be async")
	}
}

func TestHandleTypedAsyncResolvesToAsyncHandler(t *testing.T) {
	r := NewRouter()
	HandleTypedAsync(r, 1, func(ctx context.Context, c Conn, msg testMsg) (any, error) {
		return "async-typed", nil
	})

	h, err := r.Resolve(testMsg{id: 1})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !isAsync(h) {
		t.Fatal("HandleTypedAsync should register an async handler")
	}
}

func TestRouterFallbackFuncAsyncIsAsync(t *testing.T) {
	r := NewRouter()
	r.FallbackFuncAsync(func(ctx context.Context, c Conn, msg any) (any, error) {
		return "async-fallback", nil
	})

	h, err := r.Resolve("anything")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !isAsync(h) {
		t.Fatal("FallbackFuncAsync should register an async handler")
	}
}
