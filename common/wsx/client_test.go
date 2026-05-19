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

// ---------- test helpers ----------

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

// echoServer returns a handler that echoes received messages.
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

// drainServer reads and discards all messages until connection closes.
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
	cli, err := NewClient(Config{URL: "ws://localhost:9999"})
	if err != nil {
		t.Fatal(err)
	}
	if cli.State() != StateDisconnected {
		t.Fatalf("expected StateDisconnected before Connect, got %s", cli.State())
	}
}

func TestNewClient_FillsDefaults(t *testing.T) {
	cli, _ := NewClient(Config{URL: "ws://localhost:9999"})
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

	cli := MustNewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	defer cli.Close()
	waitState(t, cli, StateAuthenticated, 3*time.Second)
}

// ---------- Connect / Close ----------

func TestConnectAndClose(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	if err := cli.Close(); err != nil {
		t.Fatal(err)
	}
	if cli.State() != StateDisconnected {
		t.Fatalf("expected StateDisconnected after Close, got %s", cli.State())
	}
}

func TestConnect_AlreadyRunning(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	if err := cli.Connect(context.Background()); err != ErrAlreadyRunning {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
	cli.Close()
}

func TestDoubleClose_Noop(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.Close()
	if err := cli.Close(); err != nil {
		t.Fatalf("second close should be no-op, got: %v", err)
	}
}

func TestClose_BeforeConnect(t *testing.T) {
	cli, _ := NewClient(Config{URL: "ws://localhost:9999", MaxReconnectRetries: 0})
	if err := cli.Close(); err != nil {
		t.Fatalf("Close before Connect should succeed, got: %v", err)
	}
}

// ---------- Send / Receive (bidirectional) ----------

func TestSend_ReceiveByServer(t *testing.T) {
	received := make(chan []byte, 1)
	ts := newWSServer(func(conn *websocket.Conn) {
		_, msg, _ := conn.ReadMessage()
		received <- msg
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	cli.Connect(context.Background())
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
	cli.Close()
}

func TestOnMessage_ReceiveFromServer(t *testing.T) {
	got := make(chan []byte, 1)
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte("from-server"))
		drainServer(conn)
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0},
		WithOnMessage(func(ctx context.Context, msg []byte) error {
			got <- msg
			return nil
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case msg := <-got:
		if string(msg) != "from-server" {
			t.Fatalf("want 'from-server', got '%s'", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for OnMessage callback")
	}
	cli.Close()
}

func TestOnMessage_OnMessageError_MetricsDrop(t *testing.T) {
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte("boom"))
		drainServer(conn)
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0},
		WithOnMessage(func(ctx context.Context, msg []byte) error {
			return ErrNotConnected // trigger drop path
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	// wait for message processing
	time.Sleep(200 * time.Millisecond)
	cli.Close()
}

func TestSendJSON(t *testing.T) {
	received := make(chan []byte, 1)
	ts := newWSServer(func(conn *websocket.Conn) {
		_, msg, _ := conn.ReadMessage()
		received <- msg
	})
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	cli.Connect(context.Background())
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
	cli.Close()
}

func TestSend_NotConnected(t *testing.T) {
	cli, _ := NewClient(Config{URL: "ws://localhost:9999", MaxReconnectRetries: 0})
	if err := cli.Send(context.Background(), []byte("x")); err != ErrNotConnected {
		t.Fatalf("want ErrNotConnected, got %v", err)
	}
}

func TestSend_AfterClose(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	cli.Connect(context.Background())
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

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0},
		WithOnMessage(func(ctx context.Context, msg []byte) error {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
			return nil
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	for i := 0; i < 5; i++ {
		if err := cli.Send(context.Background(), []byte("ping")); err != nil {
			t.Fatalf("send %d failed: %v", i, err)
		}
	}

	// Wait for echoes to arrive
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	count := len(received)
	mu.Unlock()
	// Server echoes back, so OnMessage should receive echoed messages
	// (5 sent + 5 echoed = 10 total, but OnMessage only captures incoming)
	if count < 5 {
		t.Fatalf("expected at least 5 echoed messages received by client, got %d", count)
	}
	cli.Close()
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

	cli, _ := NewClient(Config{URL: wsURL(ts), WriteTimeout: 5 * time.Second, MaxReconnectRetries: 0})
	cli.Connect(context.Background())
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
	cli.Close()
}

// ---------- Authentication ----------

func TestAuthSuccess(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	called := false
	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0},
		WithAuthenticate(func(ctx context.Context) error {
			called = true
			return nil
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	if !called {
		t.Fatal("authenticate not called")
	}
	cli.Close()
}

func TestAuthFailure_NoReconnect(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	stateCh := make(chan ConnState, 3)
	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0},
		WithAuthenticate(func(ctx context.Context) error { return ErrAuthFailed }),
		WithReconnectOnAuthFailed(false),
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			if s == StateAuthFailed {
				stateCh <- s
			}
		}),
	)
	cli.Connect(context.Background())

	select {
	case s := <-stateCh:
		if s != StateAuthFailed {
			t.Fatalf("want StateAuthFailed, got %s", s)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for auth failure")
	}
	cli.Close()
}

func TestAuthTimeout(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	stateCh := make(chan ConnState, 3)
	cli, _ := NewClient(Config{URL: wsURL(ts), AuthTimeout: 100 * time.Millisecond, MaxReconnectRetries: 0},
		WithAuthenticate(func(ctx context.Context) error {
			<-ctx.Done() // block until timeout
			return ctx.Err()
		}),
		WithReconnectOnAuthFailed(false),
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			if s == StateAuthFailed {
				stateCh <- s
			}
		}),
	)
	cli.Connect(context.Background())

	select {
	case s := <-stateCh:
		if s != StateAuthFailed {
			t.Fatalf("want StateAuthFailed, got %s", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for auth timeout")
	}
	cli.Close()
}

// ---------- State transitions ----------

func TestStateTransitions(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	var mu sync.Mutex
	var states []ConnState
	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0},
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			mu.Lock()
			states = append(states, s)
			mu.Unlock()
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)
	cli.Close()

	mu.Lock()
	defer mu.Unlock()
	// Expect: Connecting → Connected → Authenticated → Disconnected
	if len(states) < 3 {
		t.Fatalf("want >=3 transitions, got %d: %v", len(states), states)
	}
}

// ---------- Context cancellation ----------

func TestContextCancel_Disconnects(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0})
	cli.Connect(ctx)
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cancel()
	waitState(t, cli, StateDisconnected, 3*time.Second)
}

func TestContextCancel_DuringReconnect(t *testing.T) {
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.ReadMessage() // block, then server closes after first msg
		conn.Close()
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		MinReconnectDelay: 1 * time.Second,
		MaxReconnectDelay: 2 * time.Second,
	},
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			if s == StateReconnecting {
				cancel() // cancel during reconnect wait
			}
		}),
	)
	cli.Connect(ctx)
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	// Trigger disconnect
	cli.Send(context.Background(), []byte("trigger"))

	waitState(t, cli, StateDisconnected, 5*time.Second)
}

// ---------- Reconnection ----------

func TestReconnect_AfterServerClose(t *testing.T) {
	connCount := atomic.Int32{}
	ts := newWSServer(func(conn *websocket.Conn) {
		connCount.Add(1)
		conn.ReadMessage() // wait for message
		conn.Close()       // then close
	})
	defer ts.Close()

	reconnecting := make(chan struct{}, 2)
	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		MinReconnectDelay: 10 * time.Millisecond,
		MaxReconnectDelay: 50 * time.Millisecond,
	},
		WithOnStateChange(func(ctx context.Context, s ConnState, err error) {
			if s == StateReconnecting {
				reconnecting <- struct{}{}
			}
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	cli.Send(context.Background(), []byte("trigger"))

	// Verify at least one reconnect attempt
	select {
	case <-reconnecting:
	case <-time.After(3 * time.Second):
		t.Fatal("no reconnect attempt after server close")
	}

	time.Sleep(200 * time.Millisecond)
	if connCount.Load() < 2 {
		t.Fatalf("want >=2 connections (initial + reconnect), got %d", connCount.Load())
	}
	cli.Close()
}

func TestReconnect_MaxRetriesDialFail(t *testing.T) {
	// Server not started → dial always fails, testing MaxReconnectRetries
	cli, _ := NewClient(Config{
		URL:                 "ws://127.0.0.1:1", // unreachable
		DialTimeout:         100 * time.Millisecond,
		MinReconnectDelay:   5 * time.Millisecond,
		MaxReconnectDelay:   10 * time.Millisecond,
		MaxReconnectRetries: 2,
	})
	cli.Connect(context.Background())
	waitState(t, cli, StateDisconnected, 5*time.Second)
	cli.Close()
}

func TestServerCloses_ClientStopsAfterRetries(t *testing.T) {
	ts := newWSServer(func(conn *websocket.Conn) {
		conn.Close() // immediately close every connection
	})
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:                 wsURL(ts),
		DialTimeout:         200 * time.Millisecond,
		MinReconnectDelay:   5 * time.Millisecond,
		MaxReconnectDelay:   15 * time.Millisecond,
		MaxReconnectRetries: 3,
	})
	cli.Connect(context.Background())
	waitState(t, cli, StateDisconnected, 5*time.Second)
	cli.Close()
}

func TestReconnect_UnlimitedRetries(t *testing.T) {
	connCount := atomic.Int32{}
	ts := newWSServer(func(conn *websocket.Conn) {
		connCount.Add(1)
		conn.Close()
	})
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cli, _ := NewClient(Config{
		URL:               wsURL(ts),
		MinReconnectDelay: 10 * time.Millisecond,
		MaxReconnectDelay: 30 * time.Millisecond,
	})
	cli.Connect(ctx)

	// Wait a bit for multiple reconnects, then cancel
	time.Sleep(200 * time.Millisecond)
	cancel()
	waitState(t, cli, StateDisconnected, 3*time.Second)

	if connCount.Load() < 2 {
		t.Fatalf("expected multiple connections, got %d", connCount.Load())
	}
	cli.Close()
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
		URL:                 wsURL(ts),
		MinReconnectDelay:   10 * time.Millisecond,
		MaxReconnectDelay:   50 * time.Millisecond,
		MaxReconnectRetries: 5,
	})
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 5*time.Second)

	if attempts.Load() < 3 {
		t.Fatalf("expected 3+ attempts, got %d", attempts.Load())
	}
	cli.Close()
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
		URL:                 wsURL(ts),
		HeartbeatInterval:   50 * time.Millisecond,
		MaxReconnectRetries: 0,
	},
		WithOnHeartbeat(func(ctx context.Context) ([]byte, error) {
			return []byte("hb"), nil
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case <-hbCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for heartbeat")
	}
	cli.Close()
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
		URL:                 wsURL(ts),
		HeartbeatInterval:   50 * time.Millisecond,
		MaxReconnectRetries: 0,
	})
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case <-pingReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for ping")
	}
	cli.Close()
}

// ---------- Token refresh ----------

func TestTokenRefresh_Called(t *testing.T) {
	refreshed := make(chan struct{}, 1)
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:                  wsURL(ts),
		TokenRefreshInterval: 100 * time.Millisecond,
		MaxReconnectRetries:  0,
	},
		WithOnTokenRefresh(func(ctx context.Context) error {
			select {
			case refreshed <- struct{}{}:
			default:
			}
			return nil
		}),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	select {
	case <-refreshed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for token refresh")
	}
	cli.Close()
}

func TestTokenRefresh_FailureDisconnects(t *testing.T) {
	ts := newWSServer(drainServer)
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:                  wsURL(ts),
		TokenRefreshInterval: 100 * time.Millisecond,
		MaxReconnectRetries:  0,
	},
		WithOnTokenRefresh(func(ctx context.Context) error { return ErrTokenRefresh }),
		WithReconnectOnTokenExpire(true),
		WithReconnectOnAuthFailed(false),
	)
	cli.Connect(context.Background())
	waitState(t, cli, StateAuthenticated, 3*time.Second)

	// Token refresh fails, connection drops. With unlimited retries client
	// reconnects, re-authenticates, and refreshes again — infinite cycle.
	// Verify the client is still running (reconnecting).
	time.Sleep(500 * time.Millisecond)
	if cli.State() == StateDisconnected {
		t.Fatal("with unlimited retries, client should keep reconnecting")
	}
	cli.Close()
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

	cli, _ := NewClient(Config{URL: wsURL(ts), MaxReconnectRetries: 0},
		WithHeaders(http.Header{"X-Test": {"value1"}, "Authorization": {"Bearer token123"}}),
	)
	cli.Connect(context.Background())

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
	cli.Close()
}

// ---------- Server shutdown ----------

func TestServerDisconnect_ReconnectAndStop(t *testing.T) {
	connCount := atomic.Int32{}
	ts := newWSServer(func(conn *websocket.Conn) {
		connCount.Add(1)
		conn.Close()
	})
	defer ts.Close()

	cli, _ := NewClient(Config{
		URL:                 wsURL(ts),
		DialTimeout:         200 * time.Millisecond,
		MinReconnectDelay:   5 * time.Millisecond,
		MaxReconnectDelay:   15 * time.Millisecond,
		MaxReconnectRetries: 3,
	})
	cli.Connect(context.Background())
	waitState(t, cli, StateDisconnected, 5*time.Second)
	if connCount.Load() < 2 {
		t.Fatalf("want >=2 connections, got %d", connCount.Load())
	}
	cli.Close()
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
		{"MinReconnectDelay", cfg.MinReconnectDelay, DefaultMinReconnectDelay},
		{"MaxReconnectDelay", cfg.MaxReconnectDelay, DefaultMaxReconnectDelay},
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

// ---------- backoffDelay ----------

func TestBackoffDelay_Capped(t *testing.T) {
	for i := 0; i < 100; i++ {
		delay := backoffDelay(100, 10*time.Millisecond, 30*time.Millisecond)
		if delay > 30*time.Millisecond {
			t.Fatalf("delay %v exceeds max", delay)
		}
	}
}

func TestBackoffDelay_StatisticallyIncreases(t *testing.T) {
	var sum0, sum5 int64
	for i := 0; i < 1000; i++ {
		sum0 += int64(backoffDelay(0, 10*time.Millisecond, 5*time.Second))
		sum5 += int64(backoffDelay(5, 10*time.Millisecond, 5*time.Second))
	}
	if sum5/1000 <= sum0/1000 {
		t.Fatalf("average delay should increase: attempt0=%d attempt5=%d", sum0/1000, sum5/1000)
	}
}

// ---------- Sentinel errors ----------

func TestSentinelErrors(t *testing.T) {
	errs := []error{
		ErrNotConnected, ErrNotRunning, ErrAlreadyRunning,
		ErrAuthTimeout, ErrAuthFailed, ErrAuthCanceled,
		ErrTokenRefresh, ErrMaxReconnect, ErrConnNil, ErrNotAuthenticated,
	}
	for _, e := range errs {
		if e.Error() == "" {
			t.Fatalf("empty error message for %T", e)
		}
	}
}
