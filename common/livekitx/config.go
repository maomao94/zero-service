// Package livekitx 提供基于 LiveKit Server SDK v2 的机制层封装：
// 统一配置与生命周期、Twirp 管理 API 访问、Token 构造、实时参与者连接、
// Data/聊天/RPC、Webhook 验签和可注销的 typed Hook 分发。
// 本包只提供机制，不包含业务会议单据、数据库模型或授权策略。
package livekitx

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	// ErrInvalidConfig 表示配置缺失、非法或互斥选项同时设置。
	ErrInvalidConfig = errors.New("livekitx: invalid configuration")
	// ErrClosed 表示 Client 已关闭，不能再创建或分发资源。
	ErrClosed = errors.New("livekitx: client is closed")
)

// Config 描述 LiveKit 管理 API 和共享 HTTP 传输配置。
type Config struct {
	URL        string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
	// HTTPService 与 HTTPClient 只能二选一；都未设置时使用标准默认客户端。
	HTTPService   HTTPService
	httpClientSet bool
	store         Store
}

// Option 修改 Client 配置；nil option 会被安全忽略。
type Option func(*Config)

// WithURL 设置 LiveKit 管理 API 地址，支持 http/https。
func WithURL(url string) Option { return func(c *Config) { c.URL = url } }

// WithAPIKey 设置管理 API 使用的 key/secret；它们不会被日志输出。
func WithAPIKey(key, secret string) Option {
	return func(c *Config) { c.APIKey, c.APISecret = key, secret }
}

// WithHTTPClient 注入标准库 HTTP 客户端。
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
		c.httpClientSet = true
	}
}

// WithHTTPService 注入 go-zero httpc.Service，复用业务侧的传输和观测配置。
func WithHTTPService(service HTTPService) Option { return func(c *Config) { c.HTTPService = service } }

// WithStore 注入活动连接索引存储；未设置时使用进程内存实现。
func WithStore(store Store) Option { return func(c *Config) { c.store = store } }

// Client 是 livekitx 的统一入口：持有复用的管理 API、Hook 分发器和
// 活动连接索引；Close 后不能再创建连接或分发事件。
type Client struct {
	api       *API
	config    Config
	hooks     *hookSet
	stateMu   sync.Mutex
	closed    bool
	store     Store
	realtime  sync.Map
	closeOnce sync.Once
	closeErr  error
}

// New 按配置创建并复用一个 LiveKit client。
func New(opts ...Option) (*Client, error) {
	cfg := Config{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	cfg.URL = strings.TrimSpace(cfg.URL)
	u, err := url.Parse(cfg.URL)
	if cfg.URL == "" || err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") ||
		strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.APISecret) == "" {
		return nil, ErrInvalidConfig
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	if cfg.HTTPService != nil && cfg.httpClientSet {
		return nil, ErrInvalidConfig
	}
	api, err := newAPI(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.store == nil {
		cfg.store = NewMemoryStore()
	}
	return &Client{api: api, config: cfg, hooks: newHookSet(), store: cfg.store}, nil
}

// API 返回复用的 LiveKit 管理 API。返回的子 client 使用 SDK 原生 Protocol 类型。
func (c *Client) API() *API { return c.api }

// Config 返回构造时的配置副本；HTTPClient 指针由调用方拥有并负责关闭。
func (c *Client) Config() Config { return c.config }

// Store 返回活动连接状态存储。存储内容是可序列化索引，不包含 SDK Room 指针。
func (c *Client) Store() Store {
	if c == nil {
		return nil
	}
	return c.store
}

func (c *Client) isClosed() bool {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.closed
}

// Close 幂等关闭 Client：停止全部 Hook 分发、断开本进程所有实时连接、
// 关闭 Store；调用方注入的 HTTP client 不被关闭。
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.stateMu.Lock()
		c.closed = true
		c.stateMu.Unlock()
		c.hooks.close()
		c.realtime.Range(func(key, _ any) bool {
			if room, ok := key.(*RealtimeRoom); ok {
				_ = room.Close()
			}
			return true
		})
		_ = c.store.Close()
	})
	return c.closeErr
}
