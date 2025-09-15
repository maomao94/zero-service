package wsx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

// 常量定义
const (
	DefaultHeartbeatInterval    = 30 * time.Second
	DefaultReconnectInterval    = 5 * time.Second
	DefaultDialTimeout          = 10 * time.Second
	DefaultTokenRefreshInterval = 30 * time.Minute // 默认token刷新间隔
	DefaultAuthTimeout          = 5 * time.Second  // 默认认证超时时间
	DefaultMaxReconnectInterval = 30 * time.Second // 默认最大重连间隔（指数退避上限）
)

// ConnStatus 连接状态枚举
type ConnStatus int

const (
	StatusDisconnected  ConnStatus = iota // 已断开连接
	StatusConnecting                      // 正在连接
	StatusConnected                       // 已连接（未认证）
	StatusAuthenticated                   // 已认证（就绪）
	StatusAuthFailed                      // 认证失败
	StatusReconnecting                    // 正在重连
)

// String 状态枚举字符串化（便于日志和调试）
func (s ConnStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "Disconnected"
	case StatusConnecting:
		return "Connecting"
	case StatusConnected:
		return "Connected(Unauthed)"
	case StatusAuthenticated:
		return "Authenticated(Ready)"
	case StatusAuthFailed:
		return "AuthFailed"
	case StatusReconnecting:
		return "Reconnecting"
	default:
		return "Unknown"
	}
}

// Client 定义WebSocket客户端接口（含所有增强功能）
type Client interface {
	// Connect 连接到WebSocket服务器
	Connect() error
	// Send 发送消息到服务器
	Send(message []byte) error
	// SendJSON 发送JSON消息到服务器
	SendJSON(data interface{}) error
	// Close 关闭WebSocket连接
	Close() error
	// IsConnected 检查是否已连接（含认证）
	IsConnected() bool
	// IsAuthenticated 检查是否已认证就绪
	IsAuthenticated() bool
	// RefreshToken 手动触发token刷新
	RefreshToken() error
}

// Config 定义WebSocket客户端配置（含所有增强配置）
type Config struct {
	URL                  string
	HeartbeatInterval    time.Duration `json:",default=30s"`
	ReconnectInterval    time.Duration `json:",default=5s"`
	ReconnectMaxRetries  int           `json:",default=0"`
	DialTimeout          time.Duration `json:",default=10s"`
	TokenRefreshInterval time.Duration `json:",default=30m"`
	AuthTimeout          time.Duration `json:",default=5s"`   // 认证超时
	ReconnectBackoff     bool          `json:",default=true"` // 是否启用重连指数退避
	MaxReconnectInterval time.Duration `json:",default=30s"`  // 最大重连间隔
}

// ClientOptions 定义客户端选项（含所有增强回调，已添加ctx支持）
type ClientOptions struct {
	Headers                http.Header
	Dialer                 *websocket.Dialer
	OnMessage              func(ctx context.Context, msg []byte) error // 消息接收回调（带ctx）
	metrics                *stat.Metrics
	OnStatusChange         func(ctx context.Context, status ConnStatus, err error) // 状态变化回调（带ctx）
	OnRefreshToken         func(ctx context.Context) (bool, error)                 // Token刷新回调（带ctx）
	OnHeartbeat            func(ctx context.Context) ([]byte, error)               // 自定义心跳内容回调（带ctx）
	ReconnectOnAuthFailed  bool                                                    // 认证失败是否重连
	ReconnectOnTokenExpire bool                                                    // Token过期是否重连
}

// ClientOption 定义自定义ClientOptions的方法
type ClientOption func(options *ClientOptions)

// client 是WebSocket客户端实现（整合所有增强功能）
type client struct {
	conn                   *websocket.Conn
	url                    string
	dialer                 *websocket.Dialer
	headers                http.Header
	onMessage              func(ctx context.Context, msg []byte) error
	metrics                *stat.Metrics
	onStatusChange         func(ctx context.Context, status ConnStatus, err error)
	onRefreshToken         func(ctx context.Context) (bool, error)
	onHeartbeat            func(ctx context.Context) ([]byte, error)
	reconnectOnAuthFailed  bool
	reconnectOnTokenExpire bool
	ctx                    context.Context
	cancel                 context.CancelFunc
	wg                     sync.WaitGroup
	mu                     sync.Mutex
	heartbeatInterval      time.Duration
	reconnectInterval      time.Duration
	reconnectMaxRetries    int
	reconnectCount         int
	running                int32 // 原子变量：客户端运行状态
	authenticated          int32 // 原子变量：是否已认证
	tokenRefreshInterval   time.Duration
	tokenRefreshTicker     *time.Ticker
	authTimeout            time.Duration
	reconnectBackoff       bool
	maxReconnectInterval   time.Duration
	logger                 logx.Logger
	connClosed             chan struct{} // 单次连接关闭通知
}

// ------------------------------ 选项构造函数 ------------------------------
// WithHeaders 设置WebSocket连接头信息
func WithHeaders(headers http.Header) ClientOption {
	return func(options *ClientOptions) {
		options.Headers = headers
	}
}

// WithDialer 设置自定义的WebSocket拨号器
func WithDialer(dialer *websocket.Dialer) ClientOption {
	return func(options *ClientOptions) {
		options.Dialer = dialer
	}
}

// WithOnMessage 设置消息处理回调（带ctx支持）
func WithOnMessage(fn func(ctx context.Context, msg []byte) error) ClientOption {
	return func(options *ClientOptions) {
		options.OnMessage = fn
	}
}

func WithMetrics(metrics *stat.Metrics) ClientOption {
	return func(options *ClientOptions) {
		options.metrics = metrics
	}
}

// WithOnStatusChange 设置连接状态变化统一回调（带ctx支持）
func WithOnStatusChange(fn func(ctx context.Context, status ConnStatus, err error)) ClientOption {
	return func(options *ClientOptions) {
		options.OnStatusChange = fn
	}
}

// WithOnRefreshToken 设置Token刷新回调（带ctx支持）
func WithOnRefreshToken(fn func(ctx context.Context) (bool, error)) ClientOption {
	return func(options *ClientOptions) {
		options.OnRefreshToken = fn
	}
}

// WithOnHeartbeat 设置自定义心跳内容回调（带ctx支持）
func WithOnHeartbeat(fn func(ctx context.Context) ([]byte, error)) ClientOption {
	return func(options *ClientOptions) {
		options.OnHeartbeat = fn
	}
}

// WithReconnectOnAuthFailed 设置认证失败时是否重连
func WithReconnectOnAuthFailed(reconnect bool) ClientOption {
	return func(options *ClientOptions) {
		options.ReconnectOnAuthFailed = reconnect
	}
}

// WithReconnectOnTokenExpire 设置Token过期时是否重连
func WithReconnectOnTokenExpire(reconnect bool) ClientOption {
	return func(options *ClientOptions) {
		options.ReconnectOnTokenExpire = reconnect
	}
}

// ------------------------------ 客户端构造函数 ------------------------------
// MustNewClient 创建客户端（失败panic，go-zero风格）
func MustNewClient(conf Config, opts ...ClientOption) Client {
	cli, err := NewClient(conf, opts...)
	logx.Must(err)
	return cli
}

// NewClient 创建客户端（核心构造函数）
func NewClient(conf Config, opts ...ClientOption) (Client, error) {
	// 初始化默认选项（带ctx的空实现）
	options := ClientOptions{
		Headers:                make(http.Header),
		OnMessage:              func(ctx context.Context, msg []byte) error { return nil },
		metrics:                stat.NewMetrics(conf.URL),
		OnStatusChange:         func(ctx context.Context, status ConnStatus, err error) {},
		OnRefreshToken:         func(ctx context.Context) (bool, error) { return true, nil },
		OnHeartbeat:            nil,
		ReconnectOnAuthFailed:  true,
		ReconnectOnTokenExpire: true,
	}

	// 应用用户自定义选项
	for _, opt := range opts {
		opt(&options)
	}

	// 初始化拨号器（默认或自定义）
	dialer := options.Dialer
	if dialer == nil {
		dialer = &websocket.Dialer{
			HandshakeTimeout: conf.DialTimeout,
		}
	}

	// 填充默认配置（优先级：用户配置 > 默认值）
	conf = fillDefaultConfig(conf)

	// 创建根上下文
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logx.ContextWithFields(ctx, logx.Field("url", conf.URL))
	ctx = logx.ContextWithFields(ctx, logx.Field("session", cryptor.Md5String(conf.URL)))
	// 初始化客户端实例
	c := &client{
		url:                    conf.URL,
		dialer:                 dialer,
		headers:                options.Headers,
		onMessage:              options.OnMessage,
		metrics:                options.metrics,
		onStatusChange:         options.OnStatusChange,
		onRefreshToken:         options.OnRefreshToken,
		onHeartbeat:            options.OnHeartbeat,
		reconnectOnAuthFailed:  options.ReconnectOnAuthFailed,
		reconnectOnTokenExpire: options.ReconnectOnTokenExpire,
		ctx:                    ctx,
		cancel:                 cancel,
		heartbeatInterval:      conf.HeartbeatInterval,
		reconnectInterval:      conf.ReconnectInterval,
		reconnectMaxRetries:    conf.ReconnectMaxRetries,
		tokenRefreshInterval:   conf.TokenRefreshInterval,
		authTimeout:            conf.AuthTimeout,
		reconnectBackoff:       conf.ReconnectBackoff,
		maxReconnectInterval:   conf.MaxReconnectInterval,
		logger:                 logx.WithContext(ctx),
		connClosed:             make(chan struct{}),
	}

	// 初始状态通知（带ctx）
	c.onStatusChange(c.ctx, StatusDisconnected, nil)
	return c, nil
}

// fillDefaultConfig 填充配置默认值
func fillDefaultConfig(conf Config) Config {
	if conf.HeartbeatInterval <= 0 {
		conf.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if conf.ReconnectInterval <= 0 {
		conf.ReconnectInterval = DefaultReconnectInterval
	}
	if conf.DialTimeout <= 0 {
		conf.DialTimeout = DefaultDialTimeout
	}
	if conf.TokenRefreshInterval <= 0 {
		conf.TokenRefreshInterval = DefaultTokenRefreshInterval
	}
	if conf.AuthTimeout <= 0 {
		conf.AuthTimeout = DefaultAuthTimeout
	}
	if conf.MaxReconnectInterval <= 0 {
		conf.MaxReconnectInterval = DefaultMaxReconnectInterval
	}

	return conf
}

// ------------------------------ 核心方法 ------------------------------
// Connect 启动客户端（非阻塞）
func (c *client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning() {
		err := errors.New("websocket client already running")
		c.logger.WithContext(c.ctx).Errorf(err.Error())
		return err
	}

	// 初始化状态
	atomic.StoreInt32(&c.running, 1)
	atomic.StoreInt32(&c.authenticated, 0)
	c.reconnectCount = 0

	// 启动连接管理器
	c.wg.Add(1)
	go c.connectionManager()

	c.logger.WithContext(c.ctx).Infof("WebSocket client started, target: %s", c.url)
	c.onStatusChange(c.ctx, StatusConnecting, nil) // 带ctx的状态通知
	return nil
}

// Send 发送消息到服务器
func (c *client) Send(message []byte) error {
	if !c.IsConnected() {
		return errors.New("not connected to server")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 设置写入超时
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.heartbeatInterval)); err != nil {
		return err
	}

	// 发送文本消息
	if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		c.logger.WithContext(c.ctx).Errorf("Failed to send message: %v", err)
		// 发送失败时关闭连接，触发重连
		go c.closeConnection()
		return err
	}

	c.logger.WithContext(c.ctx).Debugf("Message sent successfully (size: %d bytes)", len(message))
	return nil
}

// SendJSON 发送JSON消息到服务器
func (c *client) SendJSON(data interface{}) error {
	if !c.IsConnected() {
		return errors.New("not connected to server")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 设置写入超时
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.heartbeatInterval)); err != nil {
		return err
	}

	// 序列化为JSON并发送
	if err := c.conn.WriteJSON(data); err != nil {
		// 区分序列化错误和发送错误
		var jsonErr *json.MarshalerError
		if errors.As(err, &jsonErr) {
			c.logger.WithContext(c.ctx).Errorf("Failed to marshal JSON: %v", err)
			return err
		}

		c.logger.WithContext(c.ctx).Errorf("Failed to send JSON message: %v", err)
		// 发送失败时关闭连接，触发重连
		go c.closeConnection()
		return err
	}

	c.logger.WithContext(c.ctx).Debug("JSON message sent successfully")
	return nil
}

// connectionManager 连接生命周期管理器（核心循环）
func (c *client) connectionManager() {
	defer c.wg.Done()
	c.logger.WithContext(c.ctx).Info("Connection manager started")

	for c.isRunning() {
		// 1. 尝试建立连接
		conn, err := c.dial()
		if err != nil {
			c.handleConnectError(err)
			if !c.shouldReconnect() {
				break
			}
			c.waitBeforeReconnect()
			continue
		}

		// 2. 连接成功（未认证）
		c.setConnection(conn)
		c.onStatusChange(c.ctx, StatusConnected, nil) // 带ctx的状态通知

		// 3. 执行认证（带超时）
		authSuccess, authErr := c.performAuthentication()
		if !authSuccess {
			c.handleAuthFailed(authErr)
			if c.reconnectOnAuthFailed && c.shouldReconnect() {
				c.waitBeforeReconnect()
				continue
			}
			break
		}

		// 4. 认证成功（就绪状态）
		atomic.StoreInt32(&c.authenticated, 1)
		c.onStatusChange(c.ctx, StatusAuthenticated, nil) // 带ctx的状态通知
		c.startTokenRefresh()

		// 5. 等待连接关闭
		<-c.connClosed

		// 6. 连接关闭后清理
		c.clearConnection()
		atomic.StoreInt32(&c.authenticated, 0)
		c.stopTokenRefresh()
		c.onStatusChange(c.ctx, StatusDisconnected, nil) // 带ctx的状态通知

		// 7. 决定是否重连
		if !c.shouldReconnect() {
			break
		}
		c.waitBeforeReconnect()
	}

	// 8. 管理器退出
	c.logger.WithContext(c.ctx).Info("Connection manager exiting")
	c.onStatusChange(c.ctx, StatusDisconnected, nil) // 带ctx的状态通知
}

// ------------------------------ 连接相关 ------------------------------
// dial 建立WebSocket连接
func (c *client) dial() (*websocket.Conn, error) {
	c.logger.WithContext(c.ctx).Info("Trying to connect to WebSocket server")
	conn, resp, err := c.dialer.Dial(c.url, c.headers)
	if err != nil {
		// 处理响应可能为nil的情况
		var bodyContent string
		var readErr error

		// 仅在resp和resp.Body都不为空时才尝试读取
		if resp != nil && resp.Body != nil {
			// 确保响应体被关闭
			defer resp.Body.Close()

			// 读取响应体内容
			var body []byte
			body, readErr = io.ReadAll(resp.Body)
			if readErr != nil {
				bodyContent = "failed to read response body: " + readErr.Error()
			} else {
				bodyContent = string(body)
			}
		} else {
			if resp == nil {
				bodyContent = "no response received from server"
			} else {
				bodyContent = "response has no body"
			}
		}

		// 记录详细错误信息
		c.logger.WithContext(c.ctx).Errorf(
			"Connect failed: %v, body: %s",
			err,
			bodyContent,
		)
		return nil, err
	}
	return conn, nil
}

// setConnection 设置连接并启动子goroutine
func (c *client) setConnection(conn *websocket.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 重置连接关闭通道
	c.connClosed = make(chan struct{})
	c.conn = conn

	// 设置默认 PongHandler（刷新 ReadDeadline）
	c.conn.SetPongHandler(func(appData string) error {
		c.logger.WithContext(c.ctx).Debug("Received Pong, refresh ReadDeadline")
		return c.conn.SetReadDeadline(time.Now().Add(2 * c.heartbeatInterval))
	})

	// 启动消息接收和心跳
	c.wg.Add(2)
	go c.receiveLoop()
	go c.heartbeatLoop()
}

// clearConnection 清理连接资源
func (c *client) clearConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn = nil
}

// closeConnection 关闭当前连接（安全）
func (c *client) closeConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return
	}

	// 发送关闭帧（标准协议）
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client close")
	_ = c.conn.WriteMessage(websocket.CloseMessage, closeMsg)
	_ = c.conn.Close()
	c.conn = nil

	// 通知连接关闭
	safeClose(c.connClosed)
	c.logger.WithContext(c.ctx).Info("Current connection closed")
}

// ------------------------------ 认证相关 ------------------------------
// performAuthentication 执行认证（带超时）
func (c *client) performAuthentication() (bool, error) {
	c.logger.WithContext(c.ctx).Info("Starting authentication")

	// 用带超时的上下文包装认证逻辑
	authCtx, authCancel := context.WithTimeout(c.ctx, c.authTimeout)
	defer authCancel()

	// 异步执行认证（避免回调阻塞）
	resultCh := make(chan struct {
		success bool
		err     error
	}, 1) // 缓冲通道避免goroutine泄漏

	go func() {
		// 传递带超时的authCtx，确保认证能及时终止
		success, err := c.onRefreshToken(authCtx)
		resultCh <- struct {
			success bool
			err     error
		}{success, err}
	}()

	// 等待结果或超时
	select {
	case res := <-resultCh:
		if res.success {
			c.logger.WithContext(c.ctx).Info("Authentication succeeded")
			return true, nil
		}
		c.logger.WithContext(c.ctx).Errorf("Authentication failed: %v", res.err)
		return false, res.err
	case <-authCtx.Done():
		err := errors.New("authentication timeout")
		c.logger.WithContext(c.ctx).Errorf(err.Error())
		return false, err
	}
}

// handleAuthFailed 处理认证失败
func (c *client) handleAuthFailed(err error) {
	c.onStatusChange(c.ctx, StatusAuthFailed, err) // 带ctx的状态通知
	c.closeConnection()
}

// ------------------------------ 重连相关 ------------------------------
// shouldReconnect 判断是否需要重连
func (c *client) shouldReconnect() bool {
	if !c.isRunning() {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 无限重连（max=0）或未达最大次数
	if c.reconnectMaxRetries == 0 || c.reconnectCount < c.reconnectMaxRetries {
		c.onStatusChange(c.ctx, StatusReconnecting, nil) // 带ctx的状态通知
		return true
	}
	c.logger.WithContext(c.ctx).Errorf("Reach max reconnect times (%d), stop reconnect", c.reconnectMaxRetries)
	return false
}

// waitBeforeReconnect 重连前等待（支持指数退避）
func (c *client) waitBeforeReconnect() {
	c.mu.Lock()
	currentCount := c.reconnectCount
	baseInterval := c.reconnectInterval
	useBackoff := c.reconnectBackoff
	maxInterval := c.maxReconnectInterval
	c.reconnectCount++
	c.mu.Unlock()

	// 计算等待间隔（指数退避：base * 2^count，不超过max）
	waitInterval := baseInterval
	if useBackoff {
		waitInterval = baseInterval * time.Duration(1<<currentCount)
		if waitInterval > maxInterval {
			waitInterval = maxInterval
		}
	}

	c.logger.WithContext(c.ctx).Infof("Reconnect %d after %v (base: %v, backoff: %v)",
		currentCount+1, waitInterval, baseInterval, useBackoff)

	// 安全等待（支持ctx取消）
	timer := time.NewTimer(waitInterval)
	defer timer.Stop()

	select {
	case <-c.ctx.Done():
		c.logger.WithContext(c.ctx).Info("Context canceled, skip reconnect wait")
		if !timer.Stop() {
			<-timer.C // 排空通道避免泄漏
		}
	case <-timer.C:
		// 等待结束，执行重连
	}
}

// handleConnectError 处理连接错误
func (c *client) handleConnectError(err error) {
	c.onStatusChange(c.ctx, StatusDisconnected, err) // 带ctx的状态通知
}

// ------------------------------ 心跳相关 ------------------------------
// heartbeatLoop 心跳循环（支持自定义心跳）
func (c *client) heartbeatLoop() {
	defer c.wg.Done()
	c.logger.WithContext(c.ctx).Infof("Heartbeat loop started (interval: %v)", c.heartbeatInterval)

	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.WithContext(c.ctx).Info("Context canceled, stop heartbeat")
			return
		case <-c.connClosed:
			c.logger.WithContext(c.ctx).Info("Connection closed, stop heartbeat")
			return
		case <-ticker.C:
			if !c.IsConnected() {
				return
			}
			// 发送心跳（自定义或默认）
			if err := c.sendHeartbeat(); err != nil {
				c.logger.WithContext(c.ctx).Errorf("Heartbeat failed: %v", err)
				return
			}
		}
	}
}

// sendHeartbeat 发送心跳消息（支持自定义）
func (c *client) sendHeartbeat() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.New("connection is nil")
	}

	// 设置写入超时
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.heartbeatInterval)); err != nil {
		return err
	}

	// 优先使用自定义心跳（传递客户端根ctx）
	if c.onHeartbeat != nil {
		data, err := c.onHeartbeat(c.ctx)
		if err != nil {
			return err
		}
		c.logger.WithContext(c.ctx).Debugf("Send custom heartbeat (size: %d bytes)", len(data))
		return c.conn.WriteMessage(websocket.TextMessage, data)
	}

	// 默认Ping消息（标准协议）
	c.logger.WithContext(c.ctx).Debug("Send default Ping heartbeat")
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

// ------------------------------ Token刷新相关 ------------------------------
// startTokenRefresh 启动Token刷新循环
func (c *client) startTokenRefresh() {
	if c.onRefreshToken == nil || c.tokenRefreshInterval <= 0 {
		return
	}

	c.mu.Lock()
	c.tokenRefreshTicker = time.NewTicker(c.tokenRefreshInterval)
	ticker := c.tokenRefreshTicker
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.logger.WithContext(c.ctx).Infof("Token refresh loop started (interval: %v)", c.tokenRefreshInterval)

		for {
			select {
			case <-c.ctx.Done():
				c.logger.WithContext(c.ctx).Info("Context canceled, stop token refresh")
				return
			case <-c.connClosed:
				c.logger.WithContext(c.ctx).Info("Connection closed, stop token refresh")
				return
			case <-ticker.C:
				if !c.IsAuthenticated() {
					return
				}
				// 执行刷新
				if err := c.doRefreshToken(); err != nil {
					c.logger.WithContext(c.ctx).Errorf("Token refresh loop failed: %v", err)
					return
				}
			}
		}
	}()
}

// stopTokenRefresh 停止Token刷新
func (c *client) stopTokenRefresh() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenRefreshTicker != nil {
		c.tokenRefreshTicker.Stop()
		c.tokenRefreshTicker = nil
	}
}

// doRefreshToken 执行Token刷新（内部）
func (c *client) doRefreshToken() error {
	// 传递客户端根ctx，支持取消
	success, err := c.onRefreshToken(c.ctx)
	if success {
		c.logger.WithContext(c.ctx).Info("Token refreshed successfully")
		return nil
	}
	c.logger.WithContext(c.ctx).Errorf("Token refresh failed: %v", err)
	// 刷新失败处理
	if c.reconnectOnTokenExpire {
		c.closeConnection()
	}
	return err
}

// RefreshToken 手动触发Token刷新（外部接口）
func (c *client) RefreshToken() error {
	if !c.IsAuthenticated() {
		return errors.New("client not authenticated")
	}
	if c.onRefreshToken == nil {
		return errors.New("token refresh handler not set")
	}
	return c.doRefreshToken()
}

// ------------------------------ 消息接收相关 ------------------------------
// receiveLoop 消息接收循环
func (c *client) receiveLoop() {
	defer c.wg.Done()
	c.logger.WithContext(c.ctx).Info("Receive loop started")

	for c.IsConnected() {
		// 设置读取超时（2倍心跳间隔，确保能检测静默断开）
		if err := c.conn.SetReadDeadline(time.Now().Add(2 * c.heartbeatInterval)); err != nil {
			c.logger.WithContext(c.ctx).Errorf("Set read deadline failed: %v", err)
			break
		}

		// 读取消息
		msgType, msgData, err := c.conn.ReadMessage()
		if err != nil {
			// 区分正常关闭和异常错误
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.WithContext(c.ctx).Info("Server closed connection normally")
			} else {
				c.logger.WithContext(c.ctx).Errorf("Read message error: %v", err)
			}
			break
		}

		// 处理Ping/Pong（交给gorilla/websocket自动处理，这里仅日志）
		if msgType == websocket.PingMessage || msgType == websocket.PongMessage {
			c.logger.WithContext(c.ctx).Debugf("Received control message (type: %d)", msgType)
			continue
		}

		threading.GoSafe(func() {
			// 处理业务消息（传递客户端根ctx）
			c.logger.WithContext(c.ctx).Debugf("Received message (size: %d bytes, type: %d)", len(msgData), msgType)
			startTime := timex.Now()
			if err := c.onMessage(c.ctx, msgData); err != nil {
				c.logger.WithContext(c.ctx).Errorf("Handle message error: %v", err)
				c.metrics.AddDrop()
			}
			c.metrics.Add(stat.Task{
				Duration: timex.Since(startTime),
			})
		})
	}

	c.logger.WithContext(c.ctx).Info("Receive loop exiting")
	c.closeConnection()
}

// ------------------------------ 关闭相关 ------------------------------
// Close 关闭客户端（安全清理）
func (c *client) Close() error {
	c.mu.Lock()
	if !c.isRunning() {
		c.mu.Unlock()
		return nil
	}

	// 1. 标记停止状态
	atomic.StoreInt32(&c.running, 0)
	atomic.StoreInt32(&c.authenticated, 0)
	c.logger.WithContext(c.ctx).Info("Starting to close WebSocket client")

	// 2. 取消上下文（触发所有goroutine退出）
	c.cancel()

	// 3. 关闭当前连接
	var closeErr error
	if c.conn != nil {
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client shutdown")
		closeErr = c.conn.WriteMessage(websocket.CloseMessage, closeMsg)
		_ = c.conn.Close()
		c.conn = nil
	}

	// 4. 停止定时器
	c.stopTokenRefresh()

	// 5. 通知连接关闭
	safeClose(c.connClosed)

	c.mu.Unlock()

	// 6. 等待所有goroutine退出
	c.wg.Wait()

	// 7. 最终状态通知
	c.logger.WithContext(c.ctx).Info("WebSocket client closed completely")
	c.onStatusChange(c.ctx, StatusDisconnected, closeErr) // 带ctx的状态通知
	return closeErr
}

// ------------------------------ 状态查询相关 ------------------------------
// IsConnected 检查是否已连接（含物理连接，不含认证）
func (c *client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && c.isRunning()
}

// IsAuthenticated 检查是否已认证就绪
func (c *client) IsAuthenticated() bool {
	return atomic.LoadInt32(&c.authenticated) == 1 && c.IsConnected()
}

// isRunning 检查客户端是否运行中（原子读取）
func (c *client) isRunning() bool {
	return atomic.LoadInt32(&c.running) == 1
}

// ------------------------------ 工具函数 ------------------------------
// safeClose 安全关闭通道（避免重复关闭）
func safeClose(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}
