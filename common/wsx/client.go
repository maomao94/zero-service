package wsx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

// Client is the websocket client interface.
type Client interface {
	Connect(ctx context.Context) error
	Send(ctx context.Context, msg []byte) error
	SendJSON(ctx context.Context, data any) error
	Close() error
	State() ConnState
}

type client struct {
	cfg    Config
	opts   clientOptions
	dialer *websocket.Dialer

	conn *websocket.Conn

	lifeCtx    context.Context
	lifeCancel context.CancelFunc

	connCtx    context.Context
	connCancel context.CancelFunc

	wg      sync.WaitGroup
	mu      sync.Mutex
	writeMu sync.Mutex

	running       atomicBool
	authenticated atomicBool

	reconnectIdx int

	logger  logx.Logger
	metrics *stat.Metrics
}

type atomicBool struct {
	v int32
}

func (b *atomicBool) store(val bool) {
	if val {
		atomic.StoreInt32(&b.v, 1)
	} else {
		atomic.StoreInt32(&b.v, 0)
	}
}

func (b *atomicBool) load() bool {
	return atomic.LoadInt32(&b.v) == 1
}

// MustNewClient creates a client, connects it, and panics on error.
// A shutdown listener is registered so Close is called automatically on SIGTERM.
func MustNewClient(cfg Config, opts ...ClientOption) Client {
	cli, err := NewClient(cfg, opts...)
	logx.Must(err)
	if err := cli.Connect(context.Background()); err != nil {
		logx.Must(err)
	}
	proc.AddShutdownListener(func() {
		cli.Close()
	})
	return cli
}

// NewClient creates a websocket client. Call Connect to start.
func NewClient(cfg Config, opts ...ClientOption) (Client, error) {
	if cfg.URL == "" {
		return nil, errors.New("[wsx] URL is required")
	}

	cfg = normalizeConfig(cfg)
	o := defaultClientOptions()
	for _, opt := range opts {
		opt(&o)
	}

	dialer := o.dialer
	if dialer == nil {
		dialer = &websocket.Dialer{
			HandshakeTimeout: cfg.DialTimeout,
		}
	}

	metrics := o.metrics
	if metrics == nil {
		metrics = stat.NewMetrics(fmt.Sprintf("wsx-%s", cryptor.Md5String(cfg.URL)[:8]))
	}

	return &client{
		cfg:     cfg,
		opts:    o,
		dialer:  dialer,
		metrics: metrics,
	}, nil
}

// Connect starts the connection lifecycle. It returns immediately;
// connection progress is reported via WithOnStateChange.
func (c *client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running.load() {
		return ErrAlreadyRunning
	}

	c.lifeCtx, c.lifeCancel = context.WithCancel(ctx)
	c.lifeCtx = logx.ContextWithFields(c.lifeCtx, logx.Field("url", c.cfg.URL))
	c.lifeCtx = logx.ContextWithFields(c.lifeCtx, logx.Field("session", cryptor.Md5String(c.cfg.URL)[:12]))
	c.logger = logx.WithContext(c.lifeCtx)

	c.running.store(true)

	c.wg.Add(1)
	go c.connectionManager()

	c.logger.Infof("[wsx] client started, target: %s", c.cfg.URL)
	c.opts.onStateChange(c.lifeCtx, StateConnecting, nil)
	return nil
}

// Send writes a binary message. The WriteTimeout from Config is used as the
// write deadline; the supplied ctx is for tracing and cancellation checks.
func (c *client) Send(ctx context.Context, msg []byte) error {
	return c.write(ctx, websocket.TextMessage, msg)
}

// SendJSON marshals data as JSON and sends it as a text message.
func (c *client) SendJSON(ctx context.Context, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	return c.write(ctx, websocket.TextMessage, raw)
}

func (c *client) write(ctx context.Context, msgType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil || !c.running.load() {
		return ErrNotConnected
	}

	if err := conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout)); err != nil {
		return err
	}

	if err := conn.WriteMessage(msgType, data); err != nil {
		c.logger.Errorf("[wsx] write failed: %v", err)
		c.cancelConn()
		return err
	}

	c.metrics.Add(stat.Task{Duration: timex.Since(timex.Now())})
	return nil
}

// Close shuts down the client and waits for all goroutines to exit.
func (c *client) Close() error {
	c.mu.Lock()
	if !c.running.load() {
		c.mu.Unlock()
		return nil
	}

	c.running.store(false)
	c.authenticated.store(false)

	c.logger.Info("[wsx] shutting down client")

	if c.lifeCancel != nil {
		c.lifeCancel()
	}

	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	if conn != nil {
		c.writeMu.Lock()
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client shutdown")
		conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
		_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
		_ = conn.Close()
		c.writeMu.Unlock()
	}

	c.wg.Wait()

	c.logger.Info("[wsx] client shutdown complete")
	c.opts.onStateChange(context.Background(), StateDisconnected, nil)
	return nil
}

// State returns the current connection state.
func (c *client) State() ConnState {
	if !c.running.load() || (c.lifeCtx != nil && c.lifeCtx.Err() != nil) {
		return StateDisconnected
	}
	if c.authenticated.load() {
		return StateAuthenticated
	}
	c.mu.Lock()
	hasConn := c.conn != nil
	c.mu.Unlock()
	if hasConn {
		return StateConnected
	}
	return StateConnecting
}

func (c *client) connectionManager() {
	defer c.wg.Done()
	defer c.running.store(false)
	c.logger.Info("[wsx] connection manager started")

	for c.running.load() && c.lifeCtx.Err() == nil {
		conn, err := c.dial()
		if err != nil {
			c.logger.Errorf("[wsx] dial failed: %v", err)
			c.opts.onStateChange(c.lifeCtx, StateDisconnected, err)
			if !c.shouldReconnect() {
				return
			}
			c.waitBeforeReconnect()
			continue
		}

		c.setConnection(conn)
		c.opts.onStateChange(c.lifeCtx, StateConnected, nil)

		if !c.authenticate() {
			if c.opts.reconnectOnAuthFailed && c.shouldReconnect() {
				c.waitBeforeReconnect()
				continue
			}
			return
		}

		c.authenticated.store(true)
		c.opts.onStateChange(c.lifeCtx, StateAuthenticated, nil)
		c.startTokenRefresh()

		select {
		case <-c.connCtx.Done():
		case <-c.lifeCtx.Done():
		}

		c.authenticated.store(false)
		c.cancelConn()
		c.cleanupConnection()
		c.opts.onStateChange(c.lifeCtx, StateDisconnected, nil)

		if !c.shouldReconnect() {
			return
		}
		c.waitBeforeReconnect()
	}

	c.opts.onStateChange(c.lifeCtx, StateDisconnected, nil)
	c.logger.Info("[wsx] connection manager exited")
}

func (c *client) dial() (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(c.lifeCtx, c.cfg.DialTimeout)
	defer cancel()

	c.opts.onStateChange(c.lifeCtx, StateConnecting, nil)
	c.logger.Info("[wsx] dialing...")

	conn, resp, err := c.dialer.DialContext(ctx, c.cfg.URL, c.opts.headers)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	return conn, nil
}

func (c *client) setConnection(conn *websocket.Conn) {
	c.mu.Lock()
	c.connCtx, c.connCancel = context.WithCancel(c.lifeCtx)
	c.conn = conn
	c.mu.Unlock()

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
	})

	c.wg.Add(2)
	go c.readLoop(conn)
	go c.heartbeatLoop(conn)
}

func (c *client) cancelConn() {
	c.mu.Lock()
	if c.connCancel != nil {
		c.connCancel()
		c.connCancel = nil
	}
	c.mu.Unlock()
}

func (c *client) cleanupConnection() {
	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}

func (c *client) authenticate() bool {
	ctx, cancel := context.WithTimeout(c.lifeCtx, c.cfg.AuthTimeout)
	defer cancel()

	c.logger.Info("[wsx] authenticating...")

	err := c.opts.onAuthenticate(ctx)
	if err == nil {
		c.logger.Info("[wsx] authentication succeeded")
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) {
		c.logger.Errorf("[wsx] authentication timeout after %v", c.cfg.AuthTimeout)
		c.opts.onStateChange(c.lifeCtx, StateAuthFailed, ErrAuthTimeout)
	} else if errors.Is(err, context.Canceled) {
		c.logger.Error("[wsx] authentication canceled")
		c.opts.onStateChange(c.lifeCtx, StateAuthFailed, ErrAuthCanceled)
	} else {
		c.logger.Errorf("[wsx] authentication failed: %v", err)
		c.opts.onStateChange(c.lifeCtx, StateAuthFailed, err)
	}

	c.cancelConn()
	return false
}

func (c *client) readLoop(conn *websocket.Conn) {
	defer c.wg.Done()
	defer c.cancelConn()

	c.logger.Info("[wsx] read loop started")

	for c.running.load() && c.lifeCtx.Err() == nil {
		if err := conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout)); err != nil {
			c.logger.Errorf("[wsx] set read deadline failed: %v", err)
			return
		}

		msgType, msgData, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.Info("[wsx] server closed connection normally")
			} else if !c.running.load() {
				c.logger.Info("[wsx] read loop: client stopped")
			} else {
				c.logger.Errorf("[wsx] read error: %v", err)
			}
			return
		}

		if msgType == websocket.PingMessage || msgType == websocket.PongMessage {
			continue
		}

		msgCopy := make([]byte, len(msgData))
		copy(msgCopy, msgData)

		threading.GoSafe(func() {
			startTime := timex.Now()
			if err := c.opts.onMessage(c.lifeCtx, msgCopy); err != nil {
				c.logger.Errorf("[wsx] message handler error: %v", err)
				c.metrics.AddDrop()
				return
			}
			c.metrics.Add(stat.Task{Duration: timex.Since(startTime)})
		})
	}

	c.logger.Info("[wsx] read loop exited")
}

func (c *client) heartbeatLoop(conn *websocket.Conn) {
	defer c.wg.Done()

	c.logger.Infof("[wsx] heartbeat loop started (interval: %v)", c.cfg.HeartbeatInterval)

	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.lifeCtx.Done():
			c.logger.Info("[wsx] heartbeat loop: context canceled")
			return
		case <-c.connCtx.Done():
			c.logger.Info("[wsx] heartbeat loop: connection closed")
			return
		case <-ticker.C:
			if !c.running.load() {
				return
			}
			if !c.authenticated.load() {
				continue
			}

			deadline := time.Now().Add(c.cfg.WriteTimeout)
			c.writeMu.Lock()
			if c.opts.onHeartbeat != nil {
				data, err := c.opts.onHeartbeat(c.lifeCtx)
				if err != nil {
					c.writeMu.Unlock()
					c.logger.Errorf("[wsx] custom heartbeat failed: %v", err)
					return
				}
				conn.SetWriteDeadline(deadline)
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					c.writeMu.Unlock()
					c.logger.Errorf("[wsx] heartbeat write failed: %v", err)
					return
				}
			} else {
				conn.SetWriteDeadline(deadline)
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.writeMu.Unlock()
					c.logger.Errorf("[wsx] ping failed: %v", err)
					return
				}
			}
			c.writeMu.Unlock()
		}
	}
}

func (c *client) startTokenRefresh() {
	if c.opts.onTokenRefresh == nil || c.cfg.TokenRefreshInterval <= 0 {
		return
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.logger.Infof("[wsx] token refresh loop started (interval: %v)", c.cfg.TokenRefreshInterval)

		ticker := time.NewTicker(c.cfg.TokenRefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.lifeCtx.Done():
				c.logger.Info("[wsx] token refresh: context canceled")
				return
			case <-c.connCtx.Done():
				c.logger.Info("[wsx] token refresh: connection closed")
				return
			case <-ticker.C:
				if !c.authenticated.load() {
					return
				}

				ctx, cancel := context.WithTimeout(c.lifeCtx, 10*time.Second)
				err := c.opts.onTokenRefresh(ctx)
				cancel()

				if err != nil {
					c.logger.Errorf("[wsx] token refresh failed: %v", err)
					if c.opts.reconnectOnTokenExpire {
						c.cancelConn()
					}
					return
				}
				c.logger.Info("[wsx] token refreshed")
			}
		}
	}()
}

func (c *client) shouldReconnect() bool {
	if !c.running.load() || c.lifeCtx.Err() != nil {
		return false
	}

	c.mu.Lock()
	idx := c.reconnectIdx
	c.reconnectIdx++
	c.mu.Unlock()

	if c.cfg.MaxReconnectRetries > 0 && idx >= c.cfg.MaxReconnectRetries {
		c.logger.Errorf("[wsx] reached max reconnect retries (%d)", c.cfg.MaxReconnectRetries)
		c.opts.onStateChange(c.lifeCtx, StateDisconnected, ErrMaxReconnect)
		return false
	}

	c.opts.onStateChange(c.lifeCtx, StateReconnecting, nil)
	return true
}

func (c *client) waitBeforeReconnect() {
	c.mu.Lock()
	idx := c.reconnectIdx
	c.mu.Unlock()

	delay := backoffDelay(idx, c.cfg.MinReconnectDelay, c.cfg.MaxReconnectDelay)
	c.logger.Infof("[wsx] reconnect attempt %d, waiting %v", idx+1, delay)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-c.lifeCtx.Done():
		c.logger.Info("[wsx] context canceled during reconnect wait")
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
	}
}

func backoffDelay(attempt int, min, max time.Duration) time.Duration {
	base := min
	for i := 0; i < attempt; i++ {
		base *= 2
		if base > max {
			base = max
			break
		}
	}
	if base <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(base)))
}
