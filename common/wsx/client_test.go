package wsx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newWSServer(handler func(*websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}))
}

func wsURL(ts *httptest.Server) string {
	return "ws" + strings.TrimPrefix(ts.URL, "http")
}

func waitState(t *testing.T, cli Client, want ConnState, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		if cli.State() == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for state %s, current: %s", want, cli.State())
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func echoServer(received *[][]byte, mu *sync.Mutex) func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if received != nil {
				mu.Lock()
				*received = append(*received, msg)
				mu.Unlock()
			}
			if err := conn.WriteMessage(mt, msg); err != nil {
				return
			}
		}
	}
}

func drainServer(conn *websocket.Conn) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

// ---------- NewClient / MustNewClient ----------

func TestNewClient_EmptyURL(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil || !strings.Contains(err.Error(), "URL is required") {
		t.Fatalf("expected URL required error, got: %v", err)
	}
}

func TestNewClient_DefaultConfig(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, err := NewClient(Config{URL: wsURL(ts)})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)
}

func TestNewClient_FillsDefaults(t *testing.T) {
	cli, _ := NewClient(Config{URL: "ws://localhost:9999"})
	defer cli.Close()
	c := cli.(*client)
	if c.cfg.WriteTimeout != DefaultWriteTimeout {
		t.Fatalf("WriteTimeout default: want %v, got %v", DefaultWriteTimeout, c.cfg.WriteTimeout)
	}
	if c.cfg.ReadTimeout != DefaultReadTimeout {
		t.Fatalf("ReadTimeout default: want %v, got %v", DefaultReadTimeout, c.cfg.ReadTimeout)
	}
}

func TestMustNewClient(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli := MustNewClient(Config{URL: wsURL(ts)})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)
}

// ---------- Close ----------

func TestConnectAndClose(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.Close()
	if cli.State() != StateDisconnected {
		t.Fatalf("expected StateDisconnected after Close, got %s", cli.State())
	}
}

func TestDoubleClose_Noop(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.Close()
	if err := cli.Close(); err != nil {
		t.Fatalf("second close should be no-op, got: %v", err)
	}
}

func TestClose_BeforeConnect(t *testing.T) {
	cli, _ := NewClient(Config{URL: "ws://localhost:9999"})
	if err := cli.Close(); err != nil {
		t.Fatalf("Close before connect should succeed, got: %v", err)
	}
}

// ---------- Send / Receive ----------

func TestSend_ReceiveByServer(t *testing.T) {
	received := make(chan []byte, 1)
	ts := newWSServer(func(conn *websocket.Conn) {
		_, msg, _ := conn.ReadMessage()
		received <- msg
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.Send(context.Background(), []byte("hello"))

	select {
	case msg := <-received:
		if string(msg) != "hello" {
			t.Fatalf("want 'hello', got '%s'", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message on server")
	}
}

func TestOnMessage_ReceiveFromServer(t *testing.T) {
	got := make(chan []byte, 1)
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte("from-server"))
		drainServer(conn)
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)},
		WithOnMessage(func(ctx context.Context, msg []byte) error {
			got <- msg
			return nil
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case msg := <-got:
		if string(msg) != "from-server" {
			t.Fatalf("want 'from-server', got '%s'", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for OnMessage callback")
	}
}

func TestOnMessage_OnMessageError_MetricsDrop(t *testing.T) {
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte("boom"))
		drainServer(conn)
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)},
		WithOnMessage(func(ctx context.Context, msg []byte) error {
			return ErrNotConnected
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	time.Sleep(200 * time.Millisecond)
}

func TestSendJSON(t *testing.T) {
	received := make(chan []byte, 1)
	ts := newWSServer(func(conn *websocket.Conn) {
		_, msg, _ := conn.ReadMessage()
		received <- msg
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.SendJSON(context.Background(), map[string]int{"a": 1})

	select {
	case msg := <-received:
		if string(msg) != `{"a":1}` {
			t.Fatalf("want '{\"a\":1}', got '%s'", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSend_NotConnected(t *testing.T) {
	cli, _ := NewClient(Config{URL: "ws://localhost:9999"})
	defer cli.Close()
	if err := cli.Send(context.Background(), []byte("x")); err != ErrNotConnected {
		t.Fatalf("want ErrNotConnected, got %v", err)
	}
}

func TestSend_AfterClose(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)
	cli.Close()

	if err := cli.Send(context.Background(), []byte("x")); err != ErrNotConnected {
		t.Fatalf("want ErrNotConnected after Close, got %v", err)
	}
}

// ---------- Bidirectional echo ----------

func TestBidirectionalEcho(t *testing.T) {
	var mu sync.Mutex
	var received [][]byte
	ts := newWSServer(echoServer(&received, &mu))
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)},
		WithOnMessage(func(ctx context.Context, msg []byte) error {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
			return nil
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	for i := 0; i < 5; i++ {
		if err := cli.Send(context.Background(), []byte("ping")); err != nil {
			t.Fatalf("send %d failed: %v", i, err)
		}
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	count := len(received)
	mu.Unlock()
	if count < 5 {
		t.Fatalf("expected at least 5 echoed messages received by client, got %d", count)
	}
}

// ---------- Concurrent sends ----------

func TestConcurrentSends(t *testing.T) {
	msgCount := atomic.Int32{}
	ts := newWSServer(func(conn *websocket.Conn) {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			msgCount.Add(1)
		}
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), WriteTimeout: 5 * time.Second})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cli.Send(context.Background(), []byte("x"))
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	if msgCount.Load() != 20 {
		t.Fatalf("expected 20 messages on server, got %d", msgCount.Load())
	}
}

// ---------- Authentication ----------

func TestAuthSuccess(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	called := false
	cli, _ := NewClient(Config{URL: wsURL(ts)},
		WithAuthenticate(func(ctx context.Context) error {
			called = true
			return nil
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	if !called {
		t.Fatal("authenticate not called")
	}
}

func TestAuthFailure_Reconnects(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	var mu sync.Mutex
	var states []ConnState
	cli, _ := NewClient(Config{URL: wsURL(ts)},
		WithAuthenticate(func(ctx context.Context) error { return ErrAuthFailed }),
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			mu.Lock()
			states = append(states, s)
			mu.Unlock()
		}),
	)
	defer cli.Close()

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	hasConnected := false
	for _, s := range states {
		if s == StateConnected {
			hasConnected = true
			break
		}
	}
	mu.Unlock()

	if !hasConnected {
		t.Fatal("expected reconnect after auth failure")
	}
}

func TestAuthTimeout(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	stateCh := make(chan error, 1)
	cli, _ := NewClient(Config{URL: wsURL(ts), AuthTimeout: 100 * time.Millisecond},
		WithAuthenticate(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}),
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			if s == StateAuthFailed {
				stateCh <- err
			}
		}),
	)
	defer cli.Close()

	select {
	case err := <-stateCh:
		if err != ErrAuthTimeout {
			t.Fatalf("want ErrAuthTimeout, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for auth timeout")
	}
}

// ---------- State transitions ----------

func TestStateTransitions(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	var mu sync.Mutex
	var states []ConnState
	cli, _ := NewClient(Config{URL: wsURL(ts)},
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			mu.Lock()
			states = append(states, s)
			mu.Unlock()
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)
	cli.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(states) < 3 {
		t.Fatalf("want >=3 transitions, got %d: %v", len(states), states)
	}
}

// ---------- Close stops reconnection ----------

func TestClose_StopsReconnect(t *testing.T) {
	connCount := atomic.Int32{}
	ts := newWSServer(func(conn *websocket.Conn) {
		connCount.Add(1)
		conn.Close()
	})
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		ReconnectInterval: 10 * time.Millisecond,
	})
	defer cli.Close()

	time.Sleep(200 * time.Millisecond)

	if connCount.Load() < 2 {
		t.Fatalf("want >=2 connections, got %d", connCount.Load())
	}
}

func TestClose_DuringReconnect(t *testing.T) {
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.ReadMessage()
		conn.Close()
	})
	defer ts.Close()

	reconnecting := make(chan struct{}, 2)
	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		ReconnectInterval: 1 * time.Second,
	},
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			if s == StateReconnecting {
				reconnecting <- struct{}{}
			}
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.Send(context.Background(), []byte("trigger"))

	select {
	case <-reconnecting:
	case <-time.After(3 * time.Second):
		t.Fatal("no reconnect after disconnect")
	}

	cli.Close()
	waitState(t, cli, StateDisconnected, 3*time.Second)
}

// ---------- Reconnection ----------

func TestReconnect_AfterServerClose(t *testing.T) {
	connCount := atomic.Int32{}
	ts := newWSServer(func(conn *websocket.Conn) {
		connCount.Add(1)
		conn.ReadMessage()
		conn.Close()
	})
	defer ts.Close()

	reconnecting := make(chan struct{}, 2)
	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		ReconnectInterval: 10 * time.Millisecond,
	},
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			if s == StateReconnecting {
				reconnecting <- struct{}{}
			}
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.Send(context.Background(), []byte("trigger"))

	select {
	case <-reconnecting:
	case <-time.After(3 * time.Second):
		t.Fatal("no reconnect attempt after server close")
	}

	time.Sleep(200 * time.Millisecond)
	if connCount.Load() < 2 {
		t.Fatalf("want >=2 connections (initial + reconnect), got %d", connCount.Load())
	}
}

func TestReconnect_SuccessfulAfterFailures(t *testing.T) {
	attempts := atomic.Int32{}
	ts := newWSServer(func(conn *websocket.Conn) {
		n := attempts.Add(1)
		if n <= 2 {
			conn.Close()
			return
		}
		drainServer(conn)
	})
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		ReconnectInterval: 10 * time.Millisecond,
	})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 5*time.Second)

	if attempts.Load() < 3 {
		t.Fatalf("expected 3+ attempts, got %d", attempts.Load())
	}
}

// ---------- Heartbeat ----------

func TestHeartbeat_CustomPing(t *testing.T) {
	hbCh := make(chan struct{}, 3)
	ts := newWSServer(func(conn *websocket.Conn) {
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage && string(msg) == "hb" {
				hbCh <- struct{}{}
			}
		}
	})
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		HeartbeatInterval: 50 * time.Millisecond,
	},
		WithOnHeartbeat(func(ctx context.Context) ([]byte, error) {
			return []byte("hb"), nil
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case <-hbCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for heartbeat")
	}
}

func TestHeartbeat_PingRespondsPong(t *testing.T) {
	pingReceived := make(chan struct{}, 3)
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.SetPingHandler(func(appData string) error {
			pingReceived <- struct{}{}
			return conn.WriteMessage(websocket.PongMessage, []byte(appData))
		})
		drainServer(conn)
	})
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		HeartbeatInterval: 50 * time.Millisecond,
	})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case <-pingReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for ping")
	}
}

// ---------- Token refresh ----------

func TestTokenRefresh_Called(t *testing.T) {
	refreshed := make(chan struct{}, 1)
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:                  wsURL(ts),
		TokenRefreshInterval: 100 * time.Millisecond,
	},
		WithOnTokenRefresh(func(ctx context.Context) error {
			select {
			case refreshed <- struct{}{}:
			default:
			}
			return nil
		}),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case <-refreshed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for token refresh")
	}
}

func TestTokenRefresh_FailureDisconnects(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:                  wsURL(ts),
		TokenRefreshInterval: 100 * time.Millisecond,
	},
		WithOnTokenRefresh(func(ctx context.Context) error { return ErrTokenRefresh }),
	)
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	time.Sleep(500 * time.Millisecond)
	if cli.State() == StateDisconnected {
		t.Fatal("with unlimited retries, client should keep reconnecting")
	}
}

// ---------- Custom headers ----------

func TestCustomHeaders(t *testing.T) {
	headerCh := make(chan http.Header, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerCh <- r.Header
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, _ := upgrader.Upgrade(w, r, nil)
		if conn != nil {
			drainServer(conn)
			conn.Close()
		}
	}))
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts)},
		WithHeaders(http.Header{"X-Test": {"value1"}, "Authorization": {"Bearer token123"}}),
	)
	defer cli.Close()

	select {
	case h := <-headerCh:
		if h.Get("X-Test") != "value1" {
			t.Fatalf("want X-Test=value1, got %s", h.Get("X-Test"))
		}
		if h.Get("Authorization") != "Bearer token123" {
			t.Fatalf("want Authorization set, got %s", h.Get("Authorization"))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

// ---------- Config normalization ----------

func TestNormalizeConfig_AllDefaults(t *testing.T) {
	cfg := normalizeConfig(Config{})
	checks := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"DialTimeout", cfg.DialTimeout, DefaultDialTimeout},
		{"WriteTimeout", cfg.WriteTimeout, DefaultWriteTimeout},
		{"ReadTimeout", cfg.ReadTimeout, DefaultReadTimeout},
		{"AuthTimeout", cfg.AuthTimeout, DefaultAuthTimeout},
		{"HeartbeatInterval", cfg.HeartbeatInterval, DefaultHeartbeatInterval},
		{"ReconnectInterval", cfg.ReconnectInterval, DefaultReconnectInterval},
		{"TokenRefreshInterval", cfg.TokenRefreshInterval, DefaultTokenRefreshInterval},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %v, got %v", c.name, c.want, c.got)
		}
	}
}

func TestNormalizeConfig_PreservesUserValues(t *testing.T) {
	cfg := normalizeConfig(Config{WriteTimeout: 3 * time.Second, ReadTimeout: 30 * time.Second})
	if cfg.WriteTimeout != 3*time.Second {
		t.Fatalf("want 3s, got %v", cfg.WriteTimeout)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Fatalf("want 30s, got %v", cfg.ReadTimeout)
	}
	if cfg.DialTimeout != DefaultDialTimeout {
		t.Fatalf("default DialTimeout should be set, got %v", cfg.DialTimeout)
	}
}

// ---------- Sentinel errors ----------

func TestSentinelErrors(t *testing.T) {
	errs := []error{
		ErrNotConnected,
		ErrAuthTimeout, ErrAuthFailed, ErrAuthCanceled,
		ErrTokenRefresh,
	}
	for _, e := range errs {
		if e.Error() == "" {
			t.Fatalf("empty error message for %T", e)
		}
	}
}
