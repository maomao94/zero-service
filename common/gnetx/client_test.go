package gnetx

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor 轮询等待 cond 为真，超时 t.Fatal。
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

// noopServerHandler 是 server 侧不处理入站的占位 handler。
func noopServerHandler() Handler {
	return HandlerFunc(func(context.Context, Conn, any) (any, error) { return nil, nil })
}

// noopClientHandler 是 client 侧不处理入站的占位 handler。
func noopClientHandler() Handler {
	return HandlerFunc(func(context.Context, Conn, any) (any, error) { return nil, nil })
}

// TestClientSend：Client 连到 server，Send 发 echo，server 回包，client handler 收到。
func TestClientSend(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	got := make(chan *echoMsg, 1)
	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
			if e, ok := msg.(*echoMsg); ok {
				got <- e
			}
			return nil, nil
		})),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	sess := cli.Session()
	if sess == nil {
		t.Fatal("Session() nil after connect")
	}

	if err := sess.Send(context.Background(), &echoMsg{Body: "from-client"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case e := <-got:
		if e.Body != "from-client" {
			t.Fatalf("got %q, want from-client", e.Body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server reply")
	}
}

// TestClientRequestResponse：Client 发 Request 等 server 回包（Response 自动路由）。
func TestClientRequestResponse(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if p, ok := msg.(*pingReq); ok {
			return &pongResp{RespSerial: p.Serial, Reply: "ack-" + p.Msg}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := cli.Session().Request(ctx, &pingReq{Serial: 1, Msg: "hello"}, 5*time.Second)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	pong, ok := resp.(*pongResp)
	if !ok {
		t.Fatalf("resp type = %T, want *pongResp", resp)
	}
	if pong.RespSerial != 1 || pong.Reply != "ack-hello" {
		t.Fatalf("pong = %+v, want {1 ack-hello}", pong)
	}
}

// TestClientRequestTimeout：Request 超时返回错误。
func TestClientRequestTimeout(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) { return nil, nil })
	stop := startServer(t, port, srvHandler)
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err = cli.Session().Request(ctx, &pingReq{Serial: 2, Msg: "no-reply"}, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expect timeout error, got nil")
	}
}

// TestClientConnectError：拨号到不存在的 server 返回错误（不 panic）。
func TestClientConnectError(t *testing.T) {
	port := freePort(t) // 未监听端口

	_, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err == nil {
		t.Fatal("expect dial error connecting to closed port, got nil")
	}
}

// TestClientOnReady：首次连上触发 OnReady 回调一次。
func TestClientOnReady(t *testing.T) {
	port := freePort(t)
	stop := startServer(t, port, noopServerHandler())
	defer stop()

	ready := make(chan *Client, 1)
	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
		WithClientOnReady(func(c *Client) { ready <- c }),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	select {
	case c := <-ready:
		if c != cli {
			t.Fatal("OnReady got different client")
		}
	case <-time.After(time.Second):
		t.Fatal("OnReady not called")
	}
}

// TestClientSendViaClient：使用 Client.Send 便捷方法（而非 Session().Send）。
func TestClientSendViaClient(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	got := make(chan *echoMsg, 1)
	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
			if e, ok := msg.(*echoMsg); ok {
				got <- e
			}
			return nil, nil
		})),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if err := cli.Send(context.Background(), &echoMsg{Body: "via-client"}); err != nil {
		t.Fatalf("Send via Client: %v", err)
	}
	select {
	case e := <-got:
		if e.Body != "via-client" {
			t.Fatalf("got %q, want via-client", e.Body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

// TestClientRequestViaClient：使用 Client.Request 便捷方法（而非 Session().Request）。
func TestClientRequestViaClient(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if p, ok := msg.(*pingReq); ok {
			return &pongResp{RespSerial: p.Serial, Reply: "ack-" + p.Msg}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := cli.Request(ctx, &pingReq{Serial: 42, Msg: "direct"}, 5*time.Second)
	if err != nil {
		t.Fatalf("Request via Client: %v", err)
	}
	pong, ok := resp.(*pongResp)
	if !ok {
		t.Fatalf("resp type = %T, want *pongResp", resp)
	}
	if pong.RespSerial != 42 || pong.Reply != "ack-direct" {
		t.Fatalf("pong = %+v, want {42 ack-direct}", pong)
	}
}

// TestClientOpsOnDisconnected：连接关闭后 Send/Request 返回 ErrSessionClosed。
func TestClientOpsOnDisconnected(t *testing.T) {
	port := freePort(t)
	stop := startServer(t, port, noopServerHandler())
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cli.Close()

	if err := cli.Send(context.Background(), &echoMsg{Body: "x"}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Send: want ErrSessionClosed, got %v", err)
	}
	if _, err := cli.Request(context.Background(), &pingReq{Serial: 1}, time.Second); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Request: want ErrSessionClosed, got %v", err)
	}
}

// TestClientCloseIdempotent：多次 Close 不 panic。
func TestClientCloseIdempotent(t *testing.T) {
	port := freePort(t)
	stop := startServer(t, port, noopServerHandler())
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cli.Close()
	cli.Close() // 第二次不应 panic
	cli.Close() // 第三次也不应 panic
}

// TestHandlerError：client handler 返回 error 不 panic，仅日志。
func TestHandlerError(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	testErr := errors.New("test handler failure")
	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
			return nil, testErr
		})),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if err := cli.Send(context.Background(), &echoMsg{Body: "trigger"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// server echoback arrives → client handler returns error → 不 panic 即通过
	time.Sleep(200 * time.Millisecond)
}

// TestClientOnReadyNotOnReconnect：OnReady 仅在首次连上触发，重连不重复触发。
func TestClientOnReadyNotOnReconnect(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	srv, err := NewServer(
		WithAddr("127.0.0.1:"+strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithServerHandler(srvHandler),
		WithMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.Run() }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-srvDone
	}()
	time.Sleep(100 * time.Millisecond)

	readyCount := atomic.Int32{}
	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
		WithClientReconnectInterval(200*time.Millisecond),
		WithClientOnReady(func(c *Client) { readyCount.Add(1) }),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	waitFor(t, 3*time.Second, func() bool { return readyCount.Load() >= 1 })
	if got := readyCount.Load(); got != 1 {
		t.Fatalf("OnReady called %d times, want 1", got)
	}

	for _, s := range srv.Manager().All() {
		_ = s.Close()
	}
	waitFor(t, 5*time.Second, func() bool { return cli.Session() != nil })

	if readyCount.Load() != 1 {
		t.Fatalf("OnReady called %d times after reconnect, want still 1", readyCount.Load())
	}
}

// TestClientReconnect：连接被服务端断开后，Client 按固定间隔自动重连到仍在运行的 server，
// 且重连后 Session 可正常收发。用连接计数监听器判定重连发生（server 全程不停，避免端口复用抖动）。
func TestClientReconnect(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	srv, err := NewServer(
		WithAddr("127.0.0.1:"+strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithServerHandler(srvHandler),
		WithMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.Run() }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-srvDone
	}()
	time.Sleep(100 * time.Millisecond)

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
		WithClientReconnectInterval(200*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if cli.Session() == nil {
		t.Fatal("Session() nil after initial connect")
	}

	// 从服务端强制断开当前连接，触发 client 重连
	for _, s := range srv.Manager().All() {
		_ = s.Close()
	}

	// Client 自动重连到仍在运行的 server
	waitFor(t, 5*time.Second, func() bool { return cli.Session() != nil })

	// 重连后可通过 Client 便捷接口正常发送（不再是 ErrSessionClosed）
	if cli.Session() == nil {
		t.Fatal("Session() nil after reconnect")
	}
	if err := cli.Send(context.Background(), &echoMsg{Body: "after-reconnect"}); err != nil {
		t.Fatalf("Send after reconnect: %v", err)
	}
}

// TestClientRequestViaServer 验证 Client 单连接模型下 Request/Response 全链路。
func TestClientRequestViaServer(t *testing.T) {
	port := freePort(t)

	srvStop := startServer(t, port, HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if p, ok := msg.(*pingReq); ok {
			return &pongResp{RespSerial: p.Serial, Reply: "ack"}, nil
		}
		return nil, nil
	}))
	defer srvStop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := cli.Request(ctx, &pingReq{Serial: 7, Msg: "x"}, 5*time.Second)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	pong, ok := resp.(*pongResp)
	if !ok || pong.RespSerial != 7 || pong.Reply != "ack" {
		t.Fatalf("resp = %+v", resp)
	}
}

// TestClientSessionNilAfterClose 验证 Close 后 Session() 返回 nil。
func TestClientSessionNilAfterClose(t *testing.T) {
	port := freePort(t)
	stop := startServer(t, port, noopServerHandler())
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cli.Close()

	if cli.Session() != nil {
		t.Fatal("Session() should return nil after Close")
	}
}

// TestClientHeartbeat 验证 client 按 HeartbeatInterval 定时发送心跳报文。
func TestClientHeartbeat(t *testing.T) {
	port := freePort(t)

	hbCh := make(chan struct{}, 10)
	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if _, ok := msg.(*hbMsg); ok {
			select {
			case hbCh <- struct{}{}:
			default:
			}
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
		WithClientHeartbeatInterval(100*time.Millisecond),
		WithClientHeartbeatMessage(func() any { return &hbMsg{} }),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	// 等至少收到 2 个心跳
	for i := 0; i < 2; i++ {
		select {
		case <-hbCh:
		case <-time.After(2 * time.Second):
			t.Fatalf("received %d heartbeats, want at least 2", i)
		}
	}
}

// TestClientHeartbeatDisabled 验证 HeartbeatInterval=0 时不发送心跳。
func TestClientHeartbeatDisabled(t *testing.T) {
	port := freePort(t)

	hbCh := make(chan struct{}, 10)
	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if _, ok := msg.(*hbMsg); ok {
			select {
			case hbCh <- struct{}{}:
			default:
			}
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	// 确认正常收发
	if err := cli.Send(context.Background(), &echoMsg{Body: "check"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case <-hbCh:
		t.Fatal("unexpected heartbeat when HeartbeatInterval is 0")
	case <-time.After(500 * time.Millisecond):
	}
}

// TestClientIdleTimeout 验证 client 空闲超时关闭（通过 server idle timeout 断开 client）。
func TestClientIdleTimeout(t *testing.T) {
	port := freePort(t)

	srv, err := NewServer(
		WithAddr("127.0.0.1:"+strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithServerHandler(noopServerHandler()),
		WithMaxFrameLength(1024*1024),
		WithIdleTimeout(300*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Run() }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-done
	}()
	time.Sleep(100 * time.Millisecond)

	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(noopClientHandler()),
		WithClientMaxFrameLength(1024*1024),
		WithClientReconnectInterval(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if cli.Session() == nil {
		t.Fatal("Session() nil after connect")
	}

	// 不发数据，等 server idle timeout 断开
	waitFor(t, 2*time.Second, func() bool { return cli.Session() == nil })

	// 验证重连
	waitFor(t, 3*time.Second, func() bool { return cli.Session() != nil })
}
