package netx

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// Engine 是底层 HTTP 执行引擎的抽象接口，可被标准库 http.Client、go-zero httpc.Service 等实现。
type Engine interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultEngine 基于标准库 http.Client 实现 Engine 接口。
type DefaultEngine struct {
	client *http.Client
}

func (e *DefaultEngine) Do(req *http.Request) (*http.Response, error) {
	return e.client.Do(req)
}

// TransportOption 用于配置 http.Transport 和 http.Client 的创建。
type TransportOption func(*transportConfig)

type transportConfig struct {
	tlsConfig *tls.Config
}

func WithTransportTLS(cfg *tls.Config) TransportOption {
	return func(c *transportConfig) { c.tlsConfig = cfg }
}

func NewTransport(opts ...TransportOption) *http.Transport {
	cfg := &transportConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if cfg.tlsConfig != nil {
		transport.TLSClientConfig = cfg.tlsConfig
	}
	return transport
}

func NewHTTPClient(opts ...TransportOption) *http.Client {
	return newHTTPClient(opts...)
}

func newHTTPClient(opts ...TransportOption) *http.Client {
	return &http.Client{Transport: NewTransport(opts...)}
}

func InitHTTPC(name string, opts ...TransportOption) httpc.Service {
	return httpc.NewServiceWithClient(name, newHTTPClient(opts...))
}

// HTTPCEngine 基于 go-zero httpc.Service 实现 Engine 接口，可复用熔断/中间件能力。
type HTTPCEngine struct {
	svc httpc.Service
}

func (e *HTTPCEngine) Do(req *http.Request) (*http.Response, error) {
	return e.svc.DoRequest(req)
}

func NewHTTPEngine(svc httpc.Service) Engine {
	return &HTTPCEngine{svc: svc}
}
