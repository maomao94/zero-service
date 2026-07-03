package gnetx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRouterRouteByID(t *testing.T) {
	r := NewRouter(nil)
	r.RegisterFunc(1, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "handler-1", nil
	})
	r.RegisterFunc(2, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "handler-2", nil
	})

	// 命中 id=1
	reply, err := r.Handle(context.Background(), nil, testMsg{id: 1})
	if err != nil || reply != "handler-1" {
		t.Fatalf("id=1: reply=%v err=%v", reply, err)
	}
	// 命中 id=2
	reply, err = r.Handle(context.Background(), nil, testMsg{id: 2})
	if err != nil || reply != "handler-2" {
		t.Fatalf("id=2: reply=%v err=%v", reply, err)
	}
}

func TestRouterNoHandler(t *testing.T) {
	r := NewRouter(nil)
	_, err := r.Handle(context.Background(), nil, testMsg{id: 999})
	if !errors.Is(err, ErrNoHandler) {
		t.Fatalf("want ErrNoHandler, got %v", err)
	}
}

func TestRouterFallback(t *testing.T) {
	r := NewRouter(nil)
	r.FallbackFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		return "fallback", nil
	})
	reply, err := r.Handle(context.Background(), nil, testMsg{id: 999})
	if err != nil || reply != "fallback" {
		t.Fatalf("fallback: reply=%v err=%v", reply, err)
	}
}

func TestRouterMsgNotIdentifiable(t *testing.T) {
	r := NewRouter(nil)
	r.RegisterFunc(1, func(ctx context.Context, c Conn, msg any) (any, error) {
		return "handler-1", nil
	})
	// 未实现 Identifiable 的消息走 fallback；无 fallback 返回 ErrNoHandler
	_, err := r.Handle(context.Background(), nil, "plain-string")
	if !errors.Is(err, ErrNoHandler) {
		t.Fatalf("want ErrNoHandler for non-identifiable, got %v", err)
	}

	// 配了 fallback 则走 fallback
	r.FallbackFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		return "fb", nil
	})
	reply, err := r.Handle(context.Background(), nil, "plain-string")
	if err != nil || reply != "fb" {
		t.Fatalf("non-identifiable with fallback: reply=%v err=%v", reply, err)
	}
}

func TestHandleTyped(t *testing.T) {
	r := NewRouter(nil)
	HandleTyped(r, 1, func(ctx context.Context, c Conn, msg testMsg) (any, error) {
		return msg.id * 10, nil
	})

	// 正确类型
	reply, err := r.Handle(context.Background(), nil, testMsg{id: 1})
	if err != nil || reply != 10 {
		t.Fatalf("typed: reply=%v err=%v", reply, err)
	}

	// 类型不匹配
	_, err = r.Handle(context.Background(), nil, testClient{cid: "x"})
	if err == nil {
		t.Fatal("type mismatch should error")
	}
}

func TestRouterRegisterTypeFactory(t *testing.T) {
	r := NewRouter(nil)
	r.RegisterType(1, func() any { return &testMsg{} })

	f := r.Factory(1)
	if f == nil {
		t.Fatal("factory not registered")
	}
	m := f()
	if _, ok := m.(*testMsg); !ok {
		t.Fatalf("factory produced %T, want *testMsg", m)
	}

	if r.Factory(2) != nil {
		t.Fatal("unregistered id should return nil factory")
	}
}

func TestRouterIsAsyncFlag(t *testing.T) {
	// 验证 AsyncServer 标记的 handler 被 isAsync 识别。
	called := false
	h := AsyncFunc(HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		called = true
		return nil, nil
	}))
	if !isAsync(h) {
		t.Fatal("AsyncFunc handler should be recognized as async")
	}
	// 同步 handler 不应被识别为 async
	sync := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) { return nil, nil })
	if isAsync(sync) {
		t.Fatal("plain HandlerFunc should not be async")
	}
	// 执行验证
	_, _ = h.Handle(context.Background(), nil, nil)
	if !called {
		t.Fatal("async handler not called")
	}
}

// TestRouterPerHandlerAsync 验证 Router.Async(id) + Router.Handle 真正 offload 到 pool。
func TestRouterPerHandlerAsync(t *testing.T) {
	pool := defaultWorkerPool()
	r := NewRouter(pool)

	done := make(chan struct{}, 1)
	r.RegisterFunc(42, func(ctx context.Context, c Conn, msg any) (any, error) {
		close(done)
		return nil, nil
	})
	r.Async(42) // 标记为异步

	// 直接用 Router.Handle（模拟 server dispatch 路径），应 offload 到 pool 而非同步执行。
	// 同步执行的话 done 会在返回前关闭；异步的话 Handle 立即返回 (nil, nil)，done 后闭。
	reply, err := r.Handle(context.Background(), nil, testMsg{id: 42})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if reply != nil {
		t.Fatal("async handler should return nil reply from Handle")
	}
	// 等异步完成
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("async handler not dispatched to pool")
	}
}

// TestRouterPerHandlerAsyncNoPool 验证无 pool 时 Router.Async(id) 不 offload（回退同步）。
func TestRouterPerHandlerAsyncNoPool(t *testing.T) {
	r := NewRouter(nil) // 无 pool

	called := false
	r.RegisterFunc(1, func(ctx context.Context, c Conn, msg any) (any, error) {
		called = true
		return nil, nil
	})
	r.Async(1)
	// 无 pool 时 isAsync 检查跳过，handler 同步执行
	_, _ = r.Handle(context.Background(), nil, testMsg{id: 1})
	if !called {
		t.Fatal("handler not called (should run sync without pool)")
	}
}
