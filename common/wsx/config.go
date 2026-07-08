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
	DefaultReconnectInterval    = 1 * time.Second
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
	URL string

	DialTimeout          time.Duration `json:",default=10s"`
	WriteTimeout         time.Duration `json:",default=10s"`
	ReadTimeout          time.Duration `json:",default=60s"`
	AuthTimeout          time.Duration `json:",default=5s"`
	HeartbeatInterval    time.Duration `json:",default=30s"`
	ReconnectInterval    time.Duration `json:",default=1s"`
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

	metrics *stat.Metrics
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		headers:        make(http.Header),
		onAuthenticate: func(ctx context.Context) error { return nil },
		onMessage:      func(ctx context.Context, msg []byte) error { return nil },
		onStateChange:  func(ctx context.Context, state ConnState, err error) {},
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
	if cfg.ReconnectInterval <= 0 {
		cfg.ReconnectInterval = DefaultReconnectInterval
	}
	if cfg.TokenRefreshInterval <= 0 {
		cfg.TokenRefreshInterval = DefaultTokenRefreshInterval
	}
	return cfg
}

func WithHeaders(headers http.Header) ClientOption {
	return func(o *clientOptions) {
		o.headers = headers
	}
}

func WithDialer(dialer *websocket.Dialer) ClientOption {
	return func(o *clientOptions) {
		o.dialer = dialer
	}
}

func WithAuthenticate(fn func(ctx context.Context) error) ClientOption {
	return func(o *clientOptions) {
		o.onAuthenticate = fn
	}
}

func WithOnMessage(fn func(ctx context.Context, msg []byte) error) ClientOption {
	return func(o *clientOptions) {
		o.onMessage = fn
	}
}

func WithMessageHandler(h MessageHandler) ClientOption {
	return func(o *clientOptions) {
		o.onMessage = h.HandleMessage
	}
}

func WithOnStateChange(fn func(ctx context.Context, state ConnState, err error)) ClientOption {
	return func(o *clientOptions) {
		o.onStateChange = fn
	}
}

func WithOnTokenRefresh(fn func(ctx context.Context) error) ClientOption {
	return func(o *clientOptions) {
		o.onTokenRefresh = fn
	}
}

func WithOnHeartbeat(fn func(ctx context.Context) ([]byte, error)) ClientOption {
	return func(o *clientOptions) {
		o.onHeartbeat = fn
	}
}

func WithMetrics(m *stat.Metrics) ClientOption {
	return func(o *clientOptions) {
		o.metrics = m
	}
}
