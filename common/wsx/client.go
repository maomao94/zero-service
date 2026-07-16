package wsx

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Client interface {
	Send(ctx context.Context, msg []byte) error
	SendJSON(ctx context.Context, data any) error
	Close() error
	State() ConnState
}

type client struct {
	cfg  Config
	opts clientOptions

	closeCtx    context.Context
	closeCancel context.CancelFunc

	conn    atomic.Pointer[websocket.Conn]
	writeMu sync.Mutex
	state   atomic.Int32
	closed  atomic.Bool
	wg      sync.WaitGroup

	logger  logx.Logger
	metrics *stat.Metrics
	tracer  oteltrace.Tracer
}

func MustNewClient(cfg Config, opts ...ClientOption) Client {
	cli, err := NewClient(cfg, opts...)
	logx.Must(err)
	proc.AddWrapUpListener(func() { cli.Close() })
	return cli
}

func NewClient(cfg Config, opts ...ClientOption) (Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("wsx: URL is required")
	}

	cfg = normalizeConfig(cfg)
	o := defaultClientOptions()
	for _, opt := range opts {
		opt(&o)
	}

	c := &client{cfg: cfg, opts: o, metrics: o.metrics, tracer: otel.Tracer(trace.TraceName)}
	if c.metrics == nil {
		sum := md5.Sum([]byte(cfg.URL))
		c.metrics = stat.NewMetrics(fmt.Sprintf("wsx-%x", sum[:4]))
	}

	c.closeCtx, c.closeCancel = context.WithCancel(context.Background())
	sum := md5.Sum([]byte(cfg.URL))
	c.closeCtx = logx.ContextWithFields(c.closeCtx,
		logx.Field("url", cfg.URL),
		logx.Field("session", fmt.Sprintf("%x", sum[:6])),
	)
	c.logger = logx.WithContext(c.closeCtx)

	c.wg.Add(1)
	go c.running()

	c.logger.Infof("[wsx] connecting to %s", cfg.URL)
	return c, nil
}

func (c *client) Send(ctx context.Context, msg []byte) error {
	return c.writeMessage(websocket.TextMessage, msg)
}

func (c *client) SendJSON(ctx context.Context, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("wsx: marshal json: %w", err)
	}
	return c.writeMessage(websocket.TextMessage, raw)
}

func (c *client) writeMessage(msgType int, data []byte) error {
	conn := c.conn.Load()
	if conn == nil {
		return ErrNotConnected
	}
	if ConnState(c.state.Load()) != StateAuthenticated {
		return ErrNotAuthenticated
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout)); err != nil {
		return err
	}
	return conn.WriteMessage(msgType, data)
}

func (c *client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.closeCancel()

	if conn := c.conn.Swap(nil); conn != nil {
		c.writeMu.Lock()
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = conn.Close()
		c.writeMu.Unlock()
	}

	c.wg.Wait()
	c.state.Store(int32(StateDisconnected))
	c.opts.onStateChange(context.Background(), StateDisconnected, nil)
	c.logger.Info("[wsx] shutdown complete")
	return nil
}

func (c *client) State() ConnState {
	return ConnState(c.state.Load())
}

func (c *client) running() {
	defer c.wg.Done()

	for {
		if c.closeCtx.Err() != nil {
			return
		}

		conn, err := c.dial()
		if err != nil {
			c.logger.Errorf("[wsx] dial failed: %v", err)
			if !c.sleepReconnect() {
				return
			}
			continue
		}

		c.state.Store(int32(StateConnected))
		c.opts.onStateChange(c.closeCtx, StateConnected, nil)

		connCtx, connCancel := context.WithCancel(c.closeCtx)
		c.startConn(conn, connCancel)

		authOK := c.authenticate()
		if authOK {
			c.state.Store(int32(StateAuthenticated))
			c.opts.onStateChange(c.closeCtx, StateAuthenticated, nil)

			c.wg.Add(1)
			go c.heartbeater(conn, connCtx)
			c.startTokenRefresher(connCtx, connCancel)

			select {
			case <-connCtx.Done():
			case <-c.closeCtx.Done():
			}
		}

		connCancel()
		c.teardownConn()
		c.state.Store(int32(StateDisconnected))
		c.opts.onStateChange(c.closeCtx, StateDisconnected, nil)
		c.logger.Info("[wsx] connection closed")

		if !c.sleepReconnect() {
			return
		}
	}
}

func (c *client) dial() (*websocket.Conn, error) {
	c.state.Store(int32(StateConnecting))
	c.opts.onStateChange(c.closeCtx, StateConnecting, nil)
	c.logger.Info("[wsx] dialing...")

	dialer := c.opts.dialer
	if dialer == nil {
		dialer = &websocket.Dialer{HandshakeTimeout: c.cfg.DialTimeout}
	}

	ctx, cancel := context.WithTimeout(c.closeCtx, c.cfg.DialTimeout)
	defer cancel()

	conn, resp, err := dialer.DialContext(ctx, c.cfg.URL, c.opts.headers)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	return conn, nil
}

func (c *client) startConn(conn *websocket.Conn, connCancel context.CancelFunc) {
	c.conn.Store(conn)

	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
		return nil
	})

	c.logger.Info("[wsx] connected")

	c.wg.Add(1)
	go c.readLoop(conn, connCancel)
}

func (c *client) teardownConn() {
	if conn := c.conn.Swap(nil); conn != nil {
		conn.Close()
	}
}

func (c *client) authenticate() bool {
	ctx, cancel := context.WithTimeout(c.closeCtx, c.cfg.AuthTimeout)
	defer cancel()

	c.logger.Info("[wsx] authenticating...")
	err := c.opts.onAuthenticate(ctx)

	if err == nil {
		c.logger.Info("[wsx] auth success")
		return true
	}

	c.logger.Errorf("[wsx] auth failed: %v", err)

	switch {
	case ctx.Err() == context.DeadlineExceeded:
		c.opts.onStateChange(c.closeCtx, StateAuthFailed, ErrAuthTimeout)
	case ctx.Err() == context.Canceled:
		c.opts.onStateChange(c.closeCtx, StateAuthFailed, ErrAuthCanceled)
	default:
		c.opts.onStateChange(c.closeCtx, StateAuthFailed, err)
	}
	return false
}

func (c *client) readLoop(conn *websocket.Conn, connCancel context.CancelFunc) {
	defer c.wg.Done()
	defer func() {
		connCancel()
		c.teardownConn()
	}()

	for {
		if err := conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout)); err != nil {
			return
		}

		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.Info("[wsx] receive closed normally")
			} else if c.closeCtx.Err() != nil {
				c.logger.Info("[wsx] read loop: client stopped")
			} else {
				c.logger.Errorf("[wsx] read error: %v", err)
			}
			return
		}

		if msgType == websocket.PingMessage || msgType == websocket.PongMessage {
			continue
		}

		msg := make([]byte, len(data))
		copy(msg, data)

		threading.GoSafe(func() {
			ctx := context.WithoutCancel(c.closeCtx)
			ctx, span := c.tracer.Start(ctx, "wsx-receive",
				oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
				oteltrace.WithAttributes(
					attribute.String("ws.url", c.cfg.URL),
					attribute.Int("ws.msg_size", len(msg)),
				),
			)
			defer span.End()

			start := timex.Now()
			if err := c.opts.onMessage(ctx, msg); err != nil {
				c.logger.Errorf("[wsx] message handler error: %v", err)
				c.metrics.AddDrop()
				return
			}
			c.metrics.Add(stat.Task{Duration: timex.Since(start)})
		})
	}
}

func (c *client) heartbeater(conn *websocket.Conn, connCtx context.Context) {
	defer c.wg.Done()
	c.logger.Infof("[wsx] heartbeat started (interval: %v)", c.cfg.HeartbeatInterval)

	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeCtx.Done():
			return
		case <-connCtx.Done():
			return
		case <-ticker.C:
			var payload []byte
			if c.opts.onHeartbeat != nil {
				var err error
				payload, err = c.opts.onHeartbeat(c.closeCtx)
				if err != nil {
					c.logger.Errorf("[wsx] heartbeat generate failed: %v", err)
					return
				}
			}

			c.writeMu.Lock()
			var err error
			_ = conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
			if c.opts.onHeartbeat != nil {
				err = conn.WriteMessage(websocket.TextMessage, payload)
			} else {
				err = conn.WriteMessage(websocket.PingMessage, nil)
			}
			c.writeMu.Unlock()

			if err != nil {
				c.logger.Errorf("[wsx] heartbeat write failed: %v", err)
				return
			}
		}
	}
}

func (c *client) startTokenRefresher(connCtx context.Context, connCancel context.CancelFunc) {
	if c.opts.onTokenRefresh == nil || c.cfg.TokenRefreshInterval <= 0 {
		return
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.logger.Infof("[wsx] token refresh started (interval: %v)", c.cfg.TokenRefreshInterval)

		ticker := time.NewTicker(c.cfg.TokenRefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.closeCtx.Done():
				return
			case <-connCtx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(c.closeCtx, 10*time.Second)
				err := c.opts.onTokenRefresh(ctx)
				cancel()

				if err != nil {
					c.logger.Errorf("[wsx] token refresh failed: %v", err)
					connCancel()
					return
				}
				c.logger.Info("[wsx] token refreshed")
			}
		}
	}()
}

func (c *client) sleepReconnect() bool {
	c.state.Store(int32(StateReconnecting))
	c.opts.onStateChange(c.closeCtx, StateReconnecting, nil)
	c.logger.Infof("[wsx] reconnect in %v", c.cfg.ReconnectInterval)

	timer := time.NewTimer(c.cfg.ReconnectInterval)
	defer timer.Stop()

	select {
	case <-c.closeCtx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		return false
	case <-timer.C:
		return true
	}
}
