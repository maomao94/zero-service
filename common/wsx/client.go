package wsx

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// 常量定义
const (
	DefaultHeartbeatInterval    = 30 * time.Second
	DefaultReconnectInterval    = 5 * time.Second
	DefaultReconnectMaxRetries  = 0 // 0表示无限重连
	DefaultDialTimeout          = 10 * time.Second
	DefaultTokenRefreshInterval = 30 * time.Minute // 默认token刷新间隔
)

// Client 定义WebSocket客户端接口，新增Token刷新方法
type Client interface {
	// Connect 连接到WebSocket服务器
	Connect() error
	// Send 发送消息到服务器
	Send(message []byte) error
	// SendJSON 发送JSON消息到服务器
	SendJSON(data interface{}) error
	// Close 关闭WebSocket连接
	Close() error
	// IsConnected 检查是否已连接
	IsConnected() bool
	// RefreshToken 手动触发token刷新
	RefreshToken() error
}

// Config 定义WebSocket客户端配置，新增Token刷新间隔
type Config struct {
	URL                  string        `json:"url"`
	HeartbeatInterval    time.Duration `json:"heartbeatInterval" default:"30s"`
	ReconnectInterval    time.Duration `json:"reconnectInterval" default:"5s"`
	ReconnectMaxRetries  int           `json:"reconnectMaxRetries" default:"0"`
	DialTimeout          time.Duration `json:"dialTimeout" default:"10s"`
	TokenRefreshInterval time.Duration `json:"tokenRefreshInterval" default:"30m"`
}

// ClientOptions 定义客户端选项，新增认证和Token刷新回调
type ClientOptions struct {
	Headers                http.Header
	Dialer                 *websocket.Dialer
	OnMessage              func([]byte) error
	OnConnect              func()
	OnDisconnect           func(error)
	OnAuthenticate         func() (bool, error) // 连接成功后执行认证，返回是否成功
	OnAuthFailed           func(error)          // 认证失败回调
	OnRefreshToken         func() (bool, error) // 刷新token，返回是否成功
	ReconnectOnAuthFailed  bool                 // 认证失败时是否重连
	ReconnectOnTokenExpire bool                 // Token过期时是否重连
}

// ClientOption 定义自定义ClientOptions的方法
type ClientOption func(options *ClientOptions)

// client 是WebSocket客户端实现
type client struct {
	conn                   *websocket.Conn
	url                    string
	dialer                 *websocket.Dialer
	headers                http.Header
	onMessage              func([]byte) error
	onConnect              func()
	onDisconnect           func(error)
	onAuthenticate         func() (bool, error)
	onAuthFailed           func(error)
	onRefreshToken         func() (bool, error)
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
	running                int32 // 用原子变量存储运行状态
	tokenRefreshInterval   time.Duration
	tokenRefreshTicker     *time.Ticker
	logger                 logx.Logger
	connClosed             chan struct{} // 用于通知连接关闭
}

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

// WithOnMessage 设置消息处理回调
func WithOnMessage(fn func([]byte) error) ClientOption {
	return func(options *ClientOptions) {
		options.OnMessage = fn
	}
}

// WithOnConnect 设置连接成功回调
func WithOnConnect(fn func()) ClientOption {
	return func(options *ClientOptions) {
		options.OnConnect = fn
	}
}

// WithOnDisconnect 设置断开连接回调
func WithOnDisconnect(fn func(error)) ClientOption {
	return func(options *ClientOptions) {
		options.OnDisconnect = fn
	}
}

// WithOnAuthenticate 设置连接成功后的认证回调
// 认证成功返回true，失败返回false和错误
func WithOnAuthenticate(fn func() (bool, error)) ClientOption {
	return func(options *ClientOptions) {
		options.OnAuthenticate = fn
	}
}

// WithOnAuthFailed 设置认证失败回调
func WithOnAuthFailed(fn func(error)) ClientOption {
	return func(options *ClientOptions) {
		options.OnAuthFailed = fn
	}
}

// WithOnRefreshToken 设置token刷新回调
func WithOnRefreshToken(fn func() (bool, error)) ClientOption {
	return func(options *ClientOptions) {
		options.OnRefreshToken = fn
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

// MustNewClient 创建一个新的WebSocket客户端，如果失败则panic
func MustNewClient(conf Config, opts ...ClientOption) Client {
	cli, err := NewClient(conf, opts...)
	logx.Must(err)
	return cli
}

// NewClient 创建一个新的WebSocket客户端
func NewClient(conf Config, opts ...ClientOption) (Client, error) {
	// 初始化客户端选项
	options := ClientOptions{
		Headers:                make(http.Header),
		OnMessage:              func([]byte) error { return nil },
		OnConnect:              func() {},
		OnDisconnect:           func(error) {},
		OnAuthenticate:         func() (bool, error) { return true, nil }, // 默认认证成功
		OnAuthFailed:           func(error) {},
		OnRefreshToken:         func() (bool, error) { return true, nil }, // 默认刷新成功
		ReconnectOnAuthFailed:  true,
		ReconnectOnTokenExpire: true,
	}

	// 应用客户端选项
	for _, opt := range opts {
		opt(&options)
	}

	// 如果没有提供自定义拨号器，使用默认的
	dialer := options.Dialer
	if dialer == nil {
		dialer = &websocket.Dialer{
			HandshakeTimeout: conf.DialTimeout,
		}
	}

	// 设置默认值
	heartbeatInterval := conf.HeartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = DefaultHeartbeatInterval
	}

	reconnectInterval := conf.ReconnectInterval
	if reconnectInterval <= 0 {
		reconnectInterval = DefaultReconnectInterval
	}

	dialTimeout := conf.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = DefaultDialTimeout
	}

	tokenRefreshInterval := conf.TokenRefreshInterval
	if tokenRefreshInterval <= 0 {
		tokenRefreshInterval = DefaultTokenRefreshInterval
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &client{
		url:                    conf.URL,
		dialer:                 dialer,
		headers:                options.Headers,
		onMessage:              options.OnMessage,
		onConnect:              options.OnConnect,
		onDisconnect:           options.OnDisconnect,
		onAuthenticate:         options.OnAuthenticate,
		onAuthFailed:           options.OnAuthFailed,
		onRefreshToken:         options.OnRefreshToken,
		reconnectOnAuthFailed:  options.ReconnectOnAuthFailed,
		reconnectOnTokenExpire: options.ReconnectOnTokenExpire,
		ctx:                    ctx,
		cancel:                 cancel,
		heartbeatInterval:      heartbeatInterval,
		reconnectInterval:      reconnectInterval,
		reconnectMaxRetries:    conf.ReconnectMaxRetries,
		tokenRefreshInterval:   tokenRefreshInterval,
		logger:                 logx.WithContext(ctx),
		connClosed:             make(chan struct{}),
	}

	return c, nil
}

// Connect 连接到WebSocket服务器
func (c *client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning() {
		c.logger.Errorf("websocket client already running")
		return errors.New("websocket client already running")
	}

	atomic.StoreInt32(&c.running, 1)
	c.reconnectCount = 0

	// 启动连接管理器
	c.wg.Add(1)
	go c.connectionManager()

	c.logger.Infof("WebSocket client started, connecting to: %s", c.url)
	return nil
}

// connectionManager 管理连接的生命周期，包括连接、认证、重连和清理
func (c *client) connectionManager() {
	defer c.wg.Done()
	c.logger.Info("Connection manager started")

	for c.isRunning() {
		// 尝试建立连接
		conn, err := c.dial()
		if err != nil {
			c.handleConnectionError(err)
			if !c.shouldReconnect() {
				break
			}
			c.waitBeforeReconnect()
			continue
		}

		// 连接成功，设置连接并启动相关goroutine
		c.setConnection(conn)
		c.onConnect()

		// 执行认证
		authSuccess, err := c.performAuthentication()
		if !authSuccess {
			c.logger.Errorf("Authentication failed: %v", err)
			c.onAuthFailed(err)
			c.closeConnection()

			if c.reconnectOnAuthFailed && c.shouldReconnect() {
				c.waitBeforeReconnect()
				continue
			} else {
				break
			}
		}

		// 启动token刷新（如果设置了刷新回调）
		c.startTokenRefresh()

		// 等待连接关闭
		<-c.connClosed

		// 连接关闭后清理
		c.clearConnection()
		c.stopTokenRefresh()

		// 检查是否应该重连
		if !c.shouldReconnect() {
			break
		}

		c.waitBeforeReconnect()
	}

	c.logger.Info("Connection manager exiting")
}

// performAuthentication 执行连接认证
func (c *client) performAuthentication() (bool, error) {
	if c.onAuthenticate == nil {
		// 没有设置认证回调，默认成功
		return true, nil
	}

	c.logger.Info("Performing authentication")
	return c.onAuthenticate()
}

// startTokenRefresh 启动token刷新定时器
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
		c.logger.Infof("Token refresh loop started with interval: %v", c.tokenRefreshInterval)

		for {
			select {
			case <-c.ctx.Done():
				c.logger.Info("Context canceled, stopping token refresh")
				return
			case <-c.connClosed:
				c.logger.Info("Connection closed, stopping token refresh")
				return
			case <-ticker.C:
				if !c.IsConnected() {
					return
				}

				// 执行token刷新
				success, err := c.onRefreshToken()
				if !success {
					c.logger.Errorf("Token refresh failed: %v", err)
					// 处理token刷新失败
					c.handleTokenRefreshFailure(err)
					return
				}
				c.logger.Info("Token refreshed successfully")
			}
		}
	}()
}

// stopTokenRefresh 停止token刷新定时器
func (c *client) stopTokenRefresh() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenRefreshTicker != nil {
		c.tokenRefreshTicker.Stop()
		c.tokenRefreshTicker = nil
	}
}

// handleTokenRefreshFailure 处理token刷新失败
func (c *client) handleTokenRefreshFailure(err error) {
	// 刷新失败，如果配置了需要重连，则关闭当前连接触发重连
	if c.reconnectOnTokenExpire {
		c.logger.Info("Token refresh failed, initiating reconnect")
		c.closeConnection()
	}
}

// RefreshToken 手动触发token刷新
func (c *client) RefreshToken() error {
	if c.onRefreshToken == nil {
		return errors.New("no token refresh handler set")
	}

	if !c.IsConnected() {
		return errors.New("not connected to server")
	}

	success, err := c.onRefreshToken()
	if !success {
		return err
	}

	return nil
}

// dial 尝试与WebSocket服务器建立连接
func (c *client) dial() (*websocket.Conn, error) {
	c.logger.Infof("Connecting to %s", c.url)
	conn, _, err := c.dialer.Dial(c.url, c.headers)
	if err != nil {
		c.logger.Errorf("Failed to connect: %v", err)
		return nil, err
	}
	return conn, nil
}

// handleConnectionError 处理连接错误
func (c *client) handleConnectionError(err error) {
	c.logger.Errorf("Connection error: %v", err)
	c.onDisconnect(err)
}

// shouldReconnect 判断是否应该尝试重连
func (c *client) shouldReconnect() bool {
	if !c.isRunning() {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果最大重连次数为0，则无限重连
	if c.reconnectMaxRetries == 0 {
		return true
	}

	// 检查是否已达到最大重连次数
	return c.reconnectCount < c.reconnectMaxRetries
}

// waitBeforeReconnect 等待重连间隔
func (c *client) waitBeforeReconnect() {
	c.mu.Lock()
	reconnectCount := c.reconnectCount
	reconnectInterval := c.reconnectInterval
	c.reconnectCount++
	c.mu.Unlock()

	c.logger.Infof("Attempting reconnect %d in %v", reconnectCount+1, reconnectInterval)

	// 创建可复用的定时器
	timer := time.NewTimer(reconnectInterval)
	defer timer.Stop() // 确保定时器最终会被停止

	select {
	case <-c.ctx.Done():
		c.logger.Info("Context canceled, stopping reconnect")
		// 此时定时器未触发，手动停止避免资源泄漏
		if !timer.Stop() {
			<-timer.C // 确保通道被清空，避免内存泄漏
		}
		return
	case <-timer.C:
		// 等待重连间隔结束，定时器自动停止
	}
}

// setConnection 设置连接并启动相关goroutine
func (c *client) setConnection(conn *websocket.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 重置连接关闭通道
	c.connClosed = make(chan struct{})
	c.conn = conn

	// 启动消息接收循环
	c.wg.Add(1)
	go c.receiveLoop()

	// 启动心跳循环
	c.wg.Add(1)
	go c.heartbeatLoop()
}

// clearConnection 清理连接资源
func (c *client) clearConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.conn = nil
}

// receiveLoop 接收消息循环
func (c *client) receiveLoop() {
	defer c.wg.Done()
	c.logger.Info("Receive loop started")

	for c.IsConnected() {
		// 设置读取超时
		if err := c.conn.SetReadDeadline(time.Now().Add(2 * c.heartbeatInterval)); err != nil {
			c.logger.Errorf("Failed to set read deadline: %v", err)
			break
		}

		// 读取消息
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Errorf("Error reading message: %v", err)
			break
		}

		// 处理消息
		if err := c.onMessage(message); err != nil {
			c.logger.Errorf("Error handling message: %v", err)
		}
	}

	c.logger.Info("Receive loop exiting")
	c.closeConnection()
}

// heartbeatLoop 心跳循环
func (c *client) heartbeatLoop() {
	defer c.wg.Done()
	c.logger.Infof("Heartbeat loop started with interval: %v", c.heartbeatInterval)

	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Context canceled, stopping heartbeat")
			return
		case <-c.connClosed:
			c.logger.Info("Connection closed, stopping heartbeat")
			return
		case <-ticker.C:
			if !c.IsConnected() {
				return
			}

			if err := c.sendPing(); err != nil {
				c.logger.Errorf("Error sending ping: %v", err)
				return
			}
		}
	}
}

// sendPing 发送心跳消息
func (c *client) sendPing() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.New("connection is nil")
	}

	// 设置写入超时
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.heartbeatInterval)); err != nil {
		return err
	}

	// 发送ping消息
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

// Send 发送消息到服务器
func (c *client) Send(message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected() {
		return errors.New("websocket client not connected")
	}

	return c.conn.WriteMessage(websocket.TextMessage, message)
}

// SendJSON 发送JSON消息到服务器
func (c *client) SendJSON(data interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected() {
		return errors.New("websocket client not connected")
	}

	return c.conn.WriteJSON(data)
}

// Close 关闭WebSocket连接
func (c *client) Close() error {
	c.mu.Lock()
	if !c.isRunning() {
		c.mu.Unlock()
		return nil
	}

	c.logger.Info("Closing WebSocket client")

	// 停止运行状态
	atomic.StoreInt32(&c.running, 0)

	// 取消上下文
	c.cancel()

	// 关闭连接
	var err error
	if c.conn != nil {
		// 发送关闭消息
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		err = c.conn.WriteMessage(websocket.CloseMessage, closeMessage)
		// 关闭连接
		_ = c.conn.Close()
		c.conn = nil
	}

	// 停止token刷新
	c.stopTokenRefresh()

	// 关闭通道（如果尚未关闭）
	safeClose(c.connClosed)

	c.mu.Unlock()

	// 等待所有goroutine结束
	c.wg.Wait()

	c.logger.Info("WebSocket client closed")
	return err
}

// IsConnected 检查是否已连接
func (c *client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected()
}

// isConnected 内部检查连接状态，需在已加锁情况下调用
func (c *client) isConnected() bool {
	return c.conn != nil && c.isRunning()
}

// isRunning 检查客户端是否处于运行状态
func (c *client) isRunning() bool {
	return atomic.LoadInt32(&c.running) == 1
}

// closeConnection 关闭当前连接并通知
func (c *client) closeConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return
	}

	// 关闭连接
	_ = c.conn.Close()
	c.conn = nil

	// 通知连接已关闭
	safeClose(c.connClosed)

	c.logger.Info("Connection closed")
}

// safeClose 安全关闭通道（确保只关闭一次）
func safeClose(ch chan struct{}) {
	select {
	case <-ch:
		// 已关闭，不做操作
	default:
		close(ch)
	}
}
