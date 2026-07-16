package isp

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"zero-service/common/gnetx"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	cli           *gnetx.Client
	ctx           context.Context
	cancel        context.CancelFunc
	receiveCode   string
	registered    bool
	lastSessID    string
	heartbeat     time.Duration
	lastHeartbeat time.Time
	lastRecvSeq   atomic.Value
	registering   atomic.Bool
	heartbeating  atomic.Bool
	onRegister    func(*Message)
}

type recvSeq struct {
	sessionID string
	seq       uint64
}

// NewClient 创建 ISP TCP 客户端，并启动注册/心跳轮询。
// 业务 handler 通过 WithClientHandler 注册。
func NewClient(cfg ClientConfig, opts ...ClientOption) *Client {
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
	c.lastRecvSeq.Store(recvSeq{})
	if err := c.connect(o.handler); err != nil {
		logx.Errorf("[isp] 创建连接失败: %v", err)
	}
	go c.run()
	return c
}

// Close 关闭后台轮询和底层 TCP 连接。
func (c *Client) Close() {
	c.cancel()
	c.mu.Lock()
	cli := c.cli
	c.cli = nil
	c.mu.Unlock()
	if cli != nil {
		cli.Close()
	}
}

// Context 返回客户端生命周期 context。
func (c *Client) Context() context.Context { return c.ctx }

// Execute 发送 ISP 请求并等待 251-3/251-4 应答。
func (c *Client) Execute(ctx context.Context, typ, command int32, code string, items []Item) (*Message, error) {
	if typ <= 0 {
		return nil, status.Error(codes.InvalidArgument, "type 必须大于 0")
	}
	if !c.isRegistered() {
		return nil, status.Error(codes.Unavailable, "isp tcp 未注册，请等待连接就绪")
	}
	rootName, sendCode, receiveCode := c.endpointCodes()
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
	return c.request(ctx, msg)
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

func (c *Client) connect(register ClientHandler) error {
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
		return err
	}
	c.mu.Lock()
	c.cli = cli
	c.mu.Unlock()
	return nil
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
	sess := c.currentSession()
	if sess == nil {
		return
	}
	sessID := sess.ID()

	c.mu.Lock()
	if c.lastSessID != sessID {
		c.lastSessID = sessID
		c.registered = false
		c.lastRecvSeq.Store(recvSeq{})
		c.receiveCode = ""
	}
	registered := c.registered
	interval := c.heartbeat
	elapsed := time.Since(c.lastHeartbeat)
	c.mu.Unlock()

	if !registered {
		c.doRegister()
		return
	}
	if elapsed >= interval {
		c.sendHeartbeat()
	}
}

func (c *Client) doRegister() {
	if !c.registering.CompareAndSwap(false, true) {
		return
	}
	defer c.registering.Store(false)
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

	resp, err := c.request(reqCtx, msg)
	if err != nil {
		logx.Errorf("[isp] 注册失败: %v", err)
		c.closeCurrentConn()
		return
	}
	if resp.Code != StatusSuccess {
		logx.Errorf("[isp] 注册被拒绝: code=%s", resp.Code)
		c.closeCurrentConn()
		return
	}
	hb := c.heartbeat
	if len(resp.Items) > 0 {
		hb = ParseItemInterval(resp.Items[0], "heart_beat_interval", hb)
	}

	c.mu.Lock()
	if resp.SendCode != "" {
		c.receiveCode = resp.SendCode
	}
	c.heartbeat = hb
	c.registered = true
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
	if !c.heartbeating.CompareAndSwap(false, true) {
		return
	}
	defer c.heartbeating.Store(false)

	c.mu.Lock()
	c.lastHeartbeat = time.Now()
	c.mu.Unlock()

	reqCtx, cancel := context.WithTimeout(c.ctx, c.cfg.RequestTimeout)
	defer cancel()
	if _, err := c.Execute(reqCtx, TypeSystem, CommandHeartbeat, "", nil); err != nil {
		logx.Errorf("[isp] 心跳失败: %v", err)
	}
}

func (c *Client) request(ctx context.Context, msg *Message) (*Message, error) {
	sess := c.currentSession()
	if sess == nil {
		return nil, status.Error(codes.Unavailable, "isp tcp 会话未就绪")
	}
	msg.SendSeq = sess.NextSendSeq()
	rs, _ := c.lastRecvSeq.Load().(recvSeq)
	msg.RecvSeq = rs.seq
	sessID := sess.ID()
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
		return nil, status.Errorf(codes.Unavailable, "isp tcp 请求失败: %v", err)
	}
	resp, ok := respAny.(*Message)
	if !ok {
		return nil, status.Errorf(codes.Internal, "isp tcp 响应类型异常: %T", respAny)
	}
	c.trackRecvSeq(resp.SendSeq, sessID)
	return resp, nil
}

func (c *Client) currentSession() gnetx.ClientConn {
	c.mu.RLock()
	cli := c.cli
	c.mu.RUnlock()
	if cli == nil {
		return nil
	}
	return cli.Session()
}

// IsRegistered 返回当前 TCP 会话是否已完成 ISP 注册。
func (c *Client) IsRegistered() bool {
	return c.isRegistered()
}

func (c *Client) isRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registered
}

// Connected 返回客户端是否存在已注册的活跃 TCP 会话。
func (c *Client) Connected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cli != nil && c.cli.Session() != nil && c.registered
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

func (c *Client) trackRecvSeq(seq uint64, sessionID string) {
	if seq == 0 || sessionID == "" {
		return
	}
	for {
		old := c.lastRecvSeq.Load().(recvSeq)
		if old.sessionID == sessionID && seq <= old.seq {
			return
		}
		if c.lastRecvSeq.CompareAndSwap(old, recvSeq{sessionID: sessionID, seq: seq}) {
			return
		}
	}
}

func (c *Client) applyResponseState(resp *Message) {
	resp.RootName, resp.SendCode, resp.ReceiveCode = c.endpointCodes()
}

func (c *Client) closeCurrentConn() {
	c.mu.RLock()
	cli := c.cli
	c.mu.RUnlock()
	if cli == nil {
		return
	}
	sess := cli.Session()
	if sess != nil {
		_ = sess.Close()
	}
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
