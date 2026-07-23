package isp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"zero-service/common/gnetx"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// ClientHandler 注册客户端方向 ISP 指令 handler。
type ClientHandler func(*ClientRouter)

// ClientRouter 注册客户端方向 ISP 协议指令 handler。
type ClientRouter struct {
	router *gnetx.Router
	client *Client
}

// Handle 为单个 messageId 注册客户端方向 handler。
func (r *ClientRouter) Handle(messageID int, fn IspHandler) {
	clientHandleAsync(r.router, messageID, r.client, fn)
}

// HandlePairs 为多个 Type/Command 对注册同一个客户端方向 handler。
func (r *ClientRouter) HandlePairs(pairs []MessageIDPair, fn IspHandler) {
	for _, pair := range pairs {
		r.Handle(EncodeMessageID(pair.Type, pair.Cmd), fn)
	}
}

// ClientOption 调整 Client 构造行为。
type ClientOption func(*clientOptions)

type clientOptions struct {
	handler    ClientHandler
	onRegister func(*Message)
}

// WithClientHandler 注册 server→client 入站指令 handler。
func WithClientHandler(handler ClientHandler) ClientOption {
	return func(o *clientOptions) { o.handler = handler }
}

// WithClientOnRegister 在注册响应被接受后执行回调。
func WithClientOnRegister(fn func(*Message)) ClientOption {
	return func(o *clientOptions) { o.onRegister = fn }
}

// Client 管理 ISP TCP 客户端连接、注册、心跳和请求应答。
type Client struct {
	cfg ClientConfig

	mu            sync.RWMutex
	transport     *gnetx.Client
	ctx           context.Context
	cancel        context.CancelFunc
	receiveCode   string
	heartbeat     time.Duration
	lastHeartbeat time.Time
	sessionAck    atomic.Value
	onRegister    func(*Message)
}

type ackState struct {
	sessionID string
	recvSeq   uint64
}

// MustNewClient 创建 ISP TCP 客户端，初始化失败时 panic。
func MustNewClient(cfg ClientConfig, opts ...ClientOption) *Client {
	client, err := NewClient(cfg, opts...)
	logx.Must(err)
	return client
}

// NewClient 创建 ISP TCP 客户端，连接初始化成功后启动注册/心跳轮询。
// 业务 handler 通过 WithClientHandler 注册。
func NewClient(cfg ClientConfig, opts ...ClientOption) (*Client, error) {
	cfg.ApplyDefaults()
	o := &clientOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		cfg:        cfg,
		ctx:        ctx,
		cancel:     cancel,
		heartbeat:  cfg.HeartbeatInterval,
		onRegister: o.onRegister,
	}
	c.sessionAck.Store(ackState{})
	cli, err := c.connect(o.handler)
	if err != nil {
		cancel()
		return nil, err
	}
	c.transport = cli
	go c.run()
	return c, nil
}

// Close 关闭后台轮询和底层 TCP 连接。
func (c *Client) Close() {
	c.cancel()
	c.transport.Close()
	c.mu.Lock()
	c.receiveCode = ""
	c.mu.Unlock()
	c.sessionAck.Store(ackState{})
}

// Context 返回客户端生命周期 context。
func (c *Client) Context() context.Context { return c.ctx }

// Execute 发送 ISP 请求并等待 251-3/251-4 应答。
func (c *Client) Execute(ctx context.Context, typ, command int32, code string, items []Item) (*Message, error) {
	if typ <= 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidMessageType, typ)
	}
	c.mu.RLock()
	sess := c.transport.Session()
	if sess == nil || sess.ClientID() != c.cfg.SendCode {
		c.mu.RUnlock()
		return nil, ErrClientNotRegistered
	}
	rootName, sendCode, receiveCode := c.cfg.RootName, c.cfg.SendCode, c.receiveCode
	c.mu.RUnlock()
	msg := &Message{
		RootName:      rootName,
		SessionSource: SessionSourceClient,
		SendCode:      sendCode,
		ReceiveCode:   receiveCode,
		Type:          typ,
		Code:          code,
		Command:       command,
		Time:          defaultTime(""),
		Items:         items,
	}
	return c.requestOnSession(ctx, sess, msg)
}

// NewItemsResponse 使用客户端当前端点编码构造 251-4 应答。
func (c *Client) NewItemsResponse(req *Message, items []Item) *Message {
	resp := NewItemsResponse(req, SessionSourceClient, items)
	c.applyResponseState(resp)
	return resp
}

// NewSuccessResponse 使用客户端当前端点编码构造 251-3 成功应答。
func (c *Client) NewSuccessResponse(req *Message) *Message {
	resp := NewSuccessResponse(req, SessionSourceClient)
	c.applyResponseState(resp)
	return resp
}

// NewErrorResponse 使用客户端当前端点编码构造 251-3 错误应答。
func (c *Client) NewErrorResponse(req *Message, err error) *Message {
	resp := NewErrorResponse(req, SessionSourceClient, err)
	c.applyResponseState(resp)
	return resp
}

// Response 根据 error 和 items 构造客户端方向通用应答。
func (c *Client) Response(ctx context.Context, req *Message, err error, items []Item) *Message {
	if err != nil {
		LogErrorResponse(ctx, req, err, ResponseCode(err))
		return c.NewErrorResponse(req, err)
	}
	if len(items) > 0 {
		return c.NewItemsResponse(req, items)
	}
	return c.NewSuccessResponse(req)
}

func (c *Client) connect(register ClientHandler) (*gnetx.Client, error) {
	codec := NewCodec(c.cfg.RootName, c.cfg.MaxFrameLength, c.cfg.DebugLog)
	router := gnetx.NewRouter()
	if register != nil {
		register(&ClientRouter{router: router, client: c})
	}
	clientFallbackAsync(router, c, func(ctx context.Context, conn gnetx.Conn, req *Message) (*Message, error) {
		LogFallback(ctx, req)
		return nil, ErrUnimplemented
	})
	cli, err := gnetx.NewClient(c.cfg.ServerAddr,
		gnetx.WithClientCodec(codec),
		gnetx.WithClientHandler(router),
		gnetx.WithClientMaxFrameLength(c.cfg.MaxFrameLength),
		gnetx.WithClientReconnectInterval(c.cfg.ReconnectInterval),
	)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func (c *Client) run() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.tick()
		}
	}
}

func (c *Client) tick() {
	c.mu.Lock()
	sess := c.transport.Session()
	if sess == nil {
		c.receiveCode = ""
		c.sessionAck.Store(ackState{})
		c.mu.Unlock()
		return
	}
	sessID := sess.SessionID()
	ack := c.sessionAck.Load().(ackState)
	if ack.sessionID != sessID {
		c.receiveCode = ""
		c.sessionAck.Store(ackState{sessionID: sessID})
	}
	interval := c.heartbeat
	elapsed := time.Since(c.lastHeartbeat)
	c.mu.Unlock()

	if sess.ClientID() != c.cfg.SendCode {
		c.doRegister(sess)
		return
	}
	if elapsed >= interval {
		c.sendHeartbeat()
	}
}

func (c *Client) doRegister(sess gnetx.ClientConn) {
	sessID := sess.SessionID()

	c.mu.RLock()
	hb := c.heartbeat
	c.mu.RUnlock()

	msg := &Message{
		RootName:      c.cfg.RootName,
		SessionSource: SessionSourceClient,
		SendCode:      c.cfg.SendCode,
		ReceiveCode:   c.cfg.RegisterReceiveCode,
		Type:          TypeSystem,
		Command:       CommandRegister,
		Time:          defaultTime(""),
	}
	reqCtx, cancel := context.WithTimeout(c.ctx, c.cfg.RequestTimeout)
	defer cancel()

	resp, err := c.requestOnSession(reqCtx, sess, msg)
	if err != nil {
		logx.Errorf("[isp] 注册失败: %v", err)
		_ = sess.Close()
		return
	}
	if resp.Code != StatusSuccess {
		logx.Errorf("[isp] 注册被拒绝: code=%s", resp.Code)
		_ = sess.Close()
		return
	}
	if len(resp.Items) > 0 {
		hb = ParseItemInterval(resp.Items[0], "heart_beat_interval", hb)
	}
	c.mu.Lock()
	current := c.transport.Session()
	if current == nil || current.SessionID() != sessID {
		c.mu.Unlock()
		logx.Infof("[isp] 注册绑定完成时会话已切换，丢弃注册状态")
		return
	}
	if err := sess.BindClientID(c.cfg.SendCode); err != nil {
		c.mu.Unlock()
		logx.Errorf("[isp] 注册身份绑定失败: %v", err)
		_ = sess.Close()
		return
	}
	if resp.SendCode != "" {
		c.receiveCode = resp.SendCode
	}
	c.heartbeat = hb
	c.lastHeartbeat = time.Now()
	receiveCode := c.receiveCode
	heartbeat := c.heartbeat
	c.mu.Unlock()

	if c.onRegister != nil {
		c.onRegister(resp)
	}

	logx.Infof("[isp] 注册成功, receiveCode=%s, heartbeat=%s", receiveCode, heartbeat)
}

func (c *Client) sendHeartbeat() {
	c.mu.Lock()
	c.lastHeartbeat = time.Now()
	c.mu.Unlock()

	reqCtx, cancel := context.WithTimeout(c.ctx, c.cfg.RequestTimeout)
	defer cancel()
	if _, err := c.Execute(reqCtx, TypeSystem, CommandHeartbeat, "", nil); err != nil {
		logx.Errorf("[isp] 心跳失败: %v", err)
	}
}

func (c *Client) requestOnSession(ctx context.Context, sess gnetx.ClientConn, msg *Message) (*Message, error) {
	msg.SendSeq = sess.NextSendSeq()
	sessID := sess.SessionID()
	msg.RecvSeq = 0
	ack := c.sessionAck.Load().(ackState)
	if ack.sessionID == sessID {
		msg.RecvSeq = ack.recvSeq
	}
	if msg.Time == "" {
		msg.Time = defaultTime("")
	}
	ttl := c.cfg.RequestTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if d := time.Until(deadline); d > 0 && d < ttl {
			ttl = d
		}
	}
	LogOutbound(ctx, msg)
	respAny, err := sess.Request(ctx, msg, ttl)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}
	resp, ok := respAny.(*Message)
	if !ok {
		return nil, fmt.Errorf("%w: %T", ErrUnexpectedResponse, respAny)
	}
	c.trackRecvSeq(resp.SendSeq, sessID)
	return resp, nil
}

// IsRegistered 返回当前 TCP 会话是否已完成 ISP 注册。
func (c *Client) IsRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	sess := c.transport.Session()
	return sess != nil && sess.ClientID() == c.cfg.SendCode
}

// Connected 返回客户端是否存在已注册的活跃 TCP 会话。
func (c *Client) Connected() bool {
	return c.IsRegistered()
}

func (c *Client) endpointCodes() (rootName, sendCode, receiveCode string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg.RootName, c.cfg.SendCode, c.receiveCode
}

// SendCode 返回本端配置的 ISP 编码。
func (c *Client) SendCode() string {
	_, sendCode, _ := c.endpointCodes()
	return sendCode
}

// RequestTimeout 返回单次请求超时时间。
func (c *Client) RequestTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg.RequestTimeout
}

// ReceiveCode 返回注册后学习到的对端 ISP 编码。
func (c *Client) ReceiveCode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.receiveCode
}

func (c *Client) trackRecvSeq(recvSeq uint64, sessionID string) {
	if recvSeq == 0 || sessionID == "" {
		return
	}
	for {
		old := c.sessionAck.Load().(ackState)
		if old.sessionID != sessionID || recvSeq <= old.recvSeq {
			return
		}
		if c.sessionAck.CompareAndSwap(old, ackState{sessionID: sessionID, recvSeq: recvSeq}) {
			return
		}
	}
}

func (c *Client) applyResponseState(resp *Message) {
	resp.RootName, resp.SendCode, resp.ReceiveCode = c.endpointCodes()
}

func defaultTime(v string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return carbon.Now().ToDateTimeString()
}

// ParseItemInterval 从 Item 中解析秒级间隔字段。
func ParseItemInterval(item Item, key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(item[key])
	if raw == "" {
		return fallback
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return fallback
	}
	return time.Duration(sec) * time.Second
}
