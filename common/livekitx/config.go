// Package livekitx 提供基于 LiveKit Server SDK v2 的机制层封装：
// 统一配置与生命周期、Twirp 管理 API 访问、Token 构造、Room API
// （加入/创建/创建并加入/删除房间/踢人/邀请，实时事件由业务用 SDK
// 原生 RoomCallback 处理）、Webhook 验签 KeyProvider。
// 本包只提供机制，不包含业务会议单据、数据库模型或授权策略。
package livekitx

import (
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/livekit/protocol/livekit"
)

// Config 描述 LiveKit 管理 API 和共享 HTTP 传输配置。
// 它只保存"给 New 做初始化"的配置项。
type Config struct {
	URL       string
	APIKey    string
	APISecret string
	// HTTPClient 可选注入管理 API 的 HTTP 传输（即生成的 twirp client
	// 所需的 livekit.HTTPClient，*http.Client 天然满足；go-zero
	// httpc.Service 用 WithHTTPService 注入）：TLS/观测由注入方配置。
	// nil 时使用 SDK 内部传输——TLS 对自签证书容错，http/https 均可
	// 直连，调用方无需安装证书。
	HTTPClient livekit.HTTPClient
}

// Option 直接作用于 Client：配置类选项写入 c.config。nil option 会被
// 安全忽略。
type Option func(*Client)

// WithURL 设置 LiveKit 管理 API 地址，支持 http/https。
func WithURL(url string) Option { return func(c *Client) { c.config.URL = url } }

// WithAPIKey 设置管理 API 使用的 key/secret；它们不会被日志输出。
func WithAPIKey(key, secret string) Option {
	return func(c *Client) { c.config.APIKey, c.config.APISecret = key, secret }
}

// WithHTTPClient 注入管理 API 的 HTTP 传输（livekit.HTTPClient，
// *http.Client 天然满足），TLS 由注入方自行配置。
func WithHTTPClient(client livekit.HTTPClient) Option {
	return func(c *Client) { c.config.HTTPClient = client }
}

// WithHTTPService 注入 go-zero httpc.Service（便捷入口）：复用业务侧的
// 传输和观测配置，TLS 由注入的 service 决定。
func WithHTTPService(service HTTPService) Option {
	return func(c *Client) { c.config.HTTPClient = serviceHTTPClient{service: service} }
}

// Client 是 livekitx 的统一入口：持有复用的管理 API 与配置；
// Close 后不能再创建房间或发起管理请求。
type Client struct {
	api       *API
	config    Config
	closed    atomic.Bool
	closeOnce sync.Once
}

// New 按配置创建并复用一个 LiveKit client：逐个应用 Option 到 Client，
// 再校验配置（URL 非空合法、key/secret 成对、注入的传输互斥），
// 最后构造管理 API。默认 logger 注入为包级全局行为（幂等）。
func New(opts ...Option) (*Client, error) {
	c := &Client{}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	cfg := &c.config
	cfg.URL = strings.TrimSpace(cfg.URL)
	u, err := url.Parse(cfg.URL)
	if cfg.URL == "" || err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") ||
		strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.APISecret) == "" {
		return nil, ErrInvalidConfig
	}
	api, err := newAPI(*cfg)
	if err != nil {
		return nil, err
	}
	// 默认把 SDK/protocol 全局日志接到 go-zero logx（包级全局行为，
	// 幂等；业务可后续自行覆盖，见 initDefaultLogger）。
	initDefaultLogger()
	c.api = api
	return c, nil
}

// API 返回复用的 LiveKit 管理 API。返回的子 client 使用 SDK 原生 Protocol 类型。
func (c *Client) API() *API { return c.api }

// Room 返回底层 RoomService，供业务直接使用 SDK API。
func (c *Client) Room() livekit.RoomService { return c.api.Room() }

// SIP 返回底层 SIP Service，供业务直接使用 SDK SIP API。
func (c *Client) SIP() livekit.SIP { return c.api.SIP() }

// Config 返回构造时的配置副本；HTTPClient 指针由调用方拥有并负责关闭。
func (c *Client) Config() Config { return c.config }

func (c *Client) isClosed() bool { return c != nil && c.closed.Load() }

// Close 幂等标记 Client 为已关闭；调用方注入的 HTTP client 不被关闭。
// 已建立的实时连接由业务自行 room.Disconnect()，本方法不管理连接资源。
func (c *Client) Close() error {
	c.closeOnce.Do(func() { c.closed.Store(true) })
	return nil
}
