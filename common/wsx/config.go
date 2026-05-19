package wsx

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/stat"
)

const (
	DefaultDialTimeout          = 10 * time.Second
	DefaultWriteTimeout         = 10 * time.Second
	DefaultReadTimeout          = 60 * time.Second
	DefaultAuthTimeout          = 5 * time.Second
	DefaultHeartbeatInterval    = 30 * time.Second
	DefaultMinReconnectDelay    = 1 * time.Second
	DefaultMaxReconnectDelay    = 30 * time.Second
	DefaultTokenRefreshInterval = 30 * time.Minute
)

// ConnState represents the current websocket connection state.
type ConnState int

const (
	StateDisconnected ConnState = iota
	StateConnecting
	StateConnected
	StateAuthenticated
	StateAuthFailed
	StateReconnecting
)

func (s ConnState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateAuthenticated:
		return "Authenticated"
	case StateAuthFailed:
		return "AuthFailed"
	case StateReconnecting:
		return "Reconnecting"
	default:
		return "Unknown"
	}
}

// Config holds the websocket client configuration.
type Config struct {
	URL string // required

	DialTimeout          time.Duration `json:",default=10s"`
	WriteTimeout         time.Duration `json:",default=10s"`
	ReadTimeout          time.Duration `json:",default=60s"`
	AuthTimeout          time.Duration `json:",default=5s"`
	HeartbeatInterval    time.Duration `json:",default=30s"`
	MaxReconnectRetries  int           `json:",default=0"`   // 0 = unlimited
	MinReconnectDelay    time.Duration `json:",default=1s"`  // initial reconnect delay
	MaxReconnectDelay    time.Duration `json:",default=30s"` // backoff cap
	TokenRefreshInterval time.Duration `json:",default=30m"`
}

// MessageHandler handles incoming WebSocket messages.
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg []byte) error
}

// MessageHandlerFunc is an adapter that turns a function into a MessageHandler.
type MessageHandlerFunc func(ctx context.Context, msg []byte) error

func (f MessageHandlerFunc) HandleMessage(ctx context.Context, msg []byte) error {
	return f(ctx, msg)
}

// ClientOption is a functional option for configuring the client.
type ClientOption func(*clientOptions)

type clientOptions struct {
	headers http.Header
	dialer  *websocket.Dialer

	onAuthenticate func(ctx context.Context) error
	onMessage      func(ctx context.Context, msg []byte) error
	onStateChange  func(ctx context.Context, state ConnState, err error)
	onTokenRefresh func(ctx context.Context) error
	onHeartbeat    func(ctx context.Context) ([]byte, error)

	reconnectOnAuthFailed  bool
	reconnectOnTokenExpire bool

	metrics *stat.Metrics
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		headers:                make(http.Header),
		onAuthenticate:         func(ctx context.Context) error { return nil },
		onMessage:              func(ctx context.Context, msg []byte) error { return nil },
		onStateChange:          func(ctx context.Context, state ConnState, err error) {},
		onTokenRefresh:         func(ctx context.Context) error { return nil },
		reconnectOnAuthFailed:  true,
		reconnectOnTokenExpire: true,
	}
}

func normalizeConfig(cfg Config) Config {
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = DefaultDialTimeout
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = DefaultWriteTimeout
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = DefaultReadTimeout
	}
	if cfg.AuthTimeout <= 0 {
		cfg.AuthTimeout = DefaultAuthTimeout
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if cfg.MinReconnectDelay <= 0 {
		cfg.MinReconnectDelay = DefaultMinReconnectDelay
	}
	if cfg.MaxReconnectDelay <= 0 {
		cfg.MaxReconnectDelay = DefaultMaxReconnectDelay
	}
	if cfg.TokenRefreshInterval <= 0 {
		cfg.TokenRefreshInterval = DefaultTokenRefreshInterval
	}
	return cfg
}

// WithHeaders sets custom HTTP headers for the websocket handshake.
func WithHeaders(headers http.Header) ClientOption {
	return func(o *clientOptions) {
		o.headers = headers
	}
}

// WithDialer sets a custom websocket dialer.
func WithDialer(dialer *websocket.Dialer) ClientOption {
	return func(o *clientOptions) {
		o.dialer = dialer
	}
}

// WithAuthenticate sets the authentication callback invoked after dial.
// The context carries an AuthTimeout deadline; waiting beyond it fails the auth.
func WithAuthenticate(fn func(ctx context.Context) error) ClientOption {
	return func(o *clientOptions) {
		o.onAuthenticate = fn
	}
}

// WithOnMessage sets the message handler for incoming text/binary frames.
func WithOnMessage(fn func(ctx context.Context, msg []byte) error) ClientOption {
	return func(o *clientOptions) {
		o.onMessage = fn
	}
}

// WithMessageHandler sets the MessageHandler interface for incoming messages.
// Use this when you have a stateful handler that implements MessageHandler.
func WithMessageHandler(h MessageHandler) ClientOption {
	return func(o *clientOptions) {
		o.onMessage = h.HandleMessage
	}
}

// WithOnStateChange sets the callback invoked on every state transition.
func WithOnStateChange(fn func(ctx context.Context, state ConnState, err error)) ClientOption {
	return func(o *clientOptions) {
		o.onStateChange = fn
	}
}

// WithOnTokenRefresh sets the periodic token-refresh callback.
// Called at TokenRefreshInterval after successful authentication.
func WithOnTokenRefresh(fn func(ctx context.Context) error) ClientOption {
	return func(o *clientOptions) {
		o.onTokenRefresh = fn
	}
}

// WithOnHeartbeat sets a custom heartbeat payload callback.
// When unset, standard WebSocket Ping frames are used.
func WithOnHeartbeat(fn func(ctx context.Context) ([]byte, error)) ClientOption {
	return func(o *clientOptions) {
		o.onHeartbeat = fn
	}
}

// WithMetrics sets the metrics collector for message processing.
func WithMetrics(m *stat.Metrics) ClientOption {
	return func(o *clientOptions) {
		o.metrics = m
	}
}

// WithReconnectOnAuthFailed controls whether the client reconnects after auth failure.
func WithReconnectOnAuthFailed(v bool) ClientOption {
	return func(o *clientOptions) {
		o.reconnectOnAuthFailed = v
	}
}

// WithReconnectOnTokenExpire controls whether the client reconnects after token refresh failure.
func WithReconnectOnTokenExpire(v bool) ClientOption {
	return func(o *clientOptions) {
		o.reconnectOnTokenExpire = v
	}
}
